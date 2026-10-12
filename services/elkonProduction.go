package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/dronm/crudifier/v2/fields"
	"github.com/dronm/eurobeton/models"

	"github.com/dronm/ds/v4"
	"github.com/dronm/session"

	"github.com/dronm/gobizapp/errs"
	"github.com/dronm/gobizapp/logger"
	"github.com/dronm/gobizapp/ws"

	gsrv "github.com/dronm/gobizapp/services"
)

const defaultReglUserID = 1 // for failover
const dateFromElkonFormatLayout = "2006-01-02 15:04:05"

var errElkonProdNotFound = errs.NewPublicErrorCustom(http.StatusBadRequest, "2000", "завод не найден")

type ElkonMsgType = int

const (
	ElkonMsgTypeErr  ElkonMsgType = 0
	ElkonMsgTypePing ElkonMsgType = 1
	ElkonMsgTypeConf ElkonMsgType = 2
	ElkonMsgTypeLog  ElkonMsgType = 3
)

const (
	materialInsertQuery = `INSERT INTO material_fact_consumptions
	(
		production_site_id,
		upload_date_time,
		upload_user_id,
		date_time,
		concrete_type_production_descr,
		concrete_type_id,
		raw_material_production_descr,
		raw_material_id,
		vehicle_production_descr,
		vehicle_id,
		vehicle_schedule_state_id,
		concrete_quant,
		material_quant,
		material_quant_req,
		cement_silo_id,
		production_id
	) VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
	$11, $12, $13, $14, $15, $16
	)`

	materialDeleteQuery = `DELETE FROM material_fact_consumptions 
		WHERE production_site_id = $1 AND production_id = $2`

	productionInsertQuery = `INSERT INTO productions
		(
			production_site_id, production_id, 
			production_dt_start, production_dt_end, 
			production_user, production_vehicle_descr, production_concrete_type_descr,
			dt_start_set, dt_end_set,
			concrete_type_id,
			concrete_quant
		)
		VALUES 
			($1, $2, 
			$3, $4, 
			$5, $6, $7,
			$8, $9, 
			$10, $11
		)
		ON CONFLICT (production_site_id, production_id) DO UPDATE SET
			production_dt_start				= EXCLUDED.production_dt_start, 
			production_dt_end				= EXCLUDED.production_dt_end, 
			production_user					= EXCLUDED.production_user,
			production_vehicle_descr		= EXCLUDED.production_vehicle_descr, 
			production_concrete_type_descr	= EXCLUDED.production_concrete_type_descr,
			dt_start_set					= EXCLUDED.dt_start_set,
			dt_end_set						= EXCLUDED.dt_end_set,
			concrete_type_id				= EXCLUDED.concrete_type_id,
			concrete_quant					= EXCLUDED.concrete_quant
		RETURNING 
			vehicle_id, 
			vehicle_schedule_state_id, 
			shipment_id
		`
)

type ElkonProductionService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string

	FromFailoverFile bool
}

func NewElkonProductionService(db ds.Provider, sess session.Session) *ElkonProductionService {
	return &ElkonProductionService{DB: db, Session: sess}
}

func (s *ElkonProductionService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *ElkonProductionService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *ElkonProductionService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

type Nil *struct{}

// AddElkonProductionSite adds or updates a site with the given ID and session ID.
func (s *ElkonProductionService) AddElkonProductionSite(ctx context.Context, siteID int, sessionID string) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	return s.addElkonProductionSiteWithConn(ctx, poolConn.Conn(), siteID, sessionID)
}

func (s *ElkonProductionService) addElkonProductionSiteWithConn(ctx context.Context, conn ds.Conn, siteID int, sessionID string) error {
	if _, err := conn.Exec(ctx,
		`UPDATE production_sites SET
		elkon_session_id = $2
		WHERE id = $1`,
		siteID,
		sessionID,
	); err != nil {
		return fmt.Errorf("conn.Exec(): %w", err)
	}
	return nil
}

func (s *ElkonProductionService) Ping(ctx context.Context, productionSiteID int, msg *string) error {
	return s.writeElkonLog(ctx, productionSiteID, "pong", ElkonMsgTypePing, "ElkonLog.Ping")
}

// Error receives an error command from Elkon and records it for subscribers.
func (s *ElkonProductionService) Error(ctx context.Context, productionSiteID int, msg string) error {
	return s.writeElkonLog(ctx, productionSiteID, msg, ElkonMsgTypeErr, "ElkonLog.Insert")
}

func (s *ElkonProductionService) writeElkonLog(ctx context.Context, productionSiteID int, msg string, msgType ElkonMsgType, eventID string) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary(): %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	var sessionID string
	if s.Session != nil {
		sessionID = s.Session.SessionID()
		if err := s.addElkonProductionSiteWithConn(ctx, conn, productionSiteID, sessionID); err != nil {
			return err
		}
	}
	if _, err := conn.Exec(ctx,
		`SELECT elkon_log_upsert( ROW(NULL, NOW(), $1, $2, $3)::elkon_log )`,
		productionSiteID,
		msg,
		msgType,
	); err != nil {
		return fmt.Errorf("elkon_log_upsert(): %w", err)
	}
	if err := gsrv.PublishEventWithConn(ctx, conn, sessionID, eventID, nil); err != nil {
		return fmt.Errorf("PublishEventWithConn(): %w", err)
	}
	return nil
}

// PublishResponse sends command as an event for all servers to return request to the client that initiated the command.
func (s *ElkonProductionService) PublishResponse(ctx context.Context, productionSiteID int, eventID string, payload any) error {
	if err := s.AddElkonProductionSite(ctx, productionSiteID, s.Session.SessionID()); err != nil {
		return err
	}

	distrServ := NewDistribService(s.DB, s.Session)
	clientSessID, err := distrServ.SessionID(ctx, s.QueryID)
	if err != nil {
		return fmt.Errorf("AddQuery(): %w", err)
	}

	resp := &ws.SrvResponse{EventID: eventID, QueryID: s.QueryID, Payload: payload}
	logger.Logger.Debugf("PublishResponse: %v, clientSessID: %s", *resp, clientSessID)

	PublishEvent(
		ctx,
		s.Session,
		"Distrib.Command",
		models.DistribCommand{
			RecipientSessionID: clientSessID,
			Payload:            resp,
		},
	)

	return nil
}

// Get recieves 'get' command from Elkon. It issues ws query to the calling client.
func (s *ElkonProductionService) Get(ctx context.Context, productionSiteID int, paramValue any) error {
	return s.PublishResponse(ctx, productionSiteID, "ElkonProduction.Get", paramValue)
}

func (s *ElkonProductionService) FetchConfig(ctx context.Context, productionSiteID int, conf models.ElkonConfig) error {
	return s.PublishResponse(ctx, productionSiteID, "ElkonProduction.FetchConfig", conf)
}

// Log executes recieves 'log' command from Elkon. It issues ws query to the calling client.
func (s *ElkonProductionService) Log(ctx context.Context, productionSiteID int, tail models.LogTail) error {
	return nil
}

func (s *ElkonProductionService) Upload(ctx context.Context, productionSiteID int, msg models.ElkonProductionList) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		logger.Logger.Errorf("GetPrimary(): %v", err)
		return s.uploadError(ctx, nil, productionSiteID, &msg, fmt.Errorf("GetPrimary(): %w", err))
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	if s.Session != nil {
		if err := s.addElkonProductionSiteWithConn(ctx, conn, productionSiteID, s.Session.SessionID()); err != nil {
			return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("AddElkonProductionSite(): %w", err))
		}
	}

	// might be a mistake if loaded from failover
	var userID int
	if s.FromFailoverFile {
		//static user
		userID = defaultReglUserID
	} else {
		userID, err = LoggedUserID(s.Session)
		if err != nil {
			return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("LoggedUserID(): %w", err))
		}
	}
	//?????
	if userID == 0 {
		userID = defaultReglUserID
	}

	productionStmt, err := conn.Prepare(ctx, "productionInsertQuery", productionInsertQuery)
	if err != nil {
		return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("conn.Prepare() productionInsertQuery: %w", err))
	}

	materialDeleteStmt, err := conn.Prepare(ctx, "materialDeleteQuery", materialDeleteQuery)
	if err != nil {
		return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("conn.Prepare() materialDeleteQuery: %w", err))
	}
	materialInsertStmt, err := conn.Prepare(ctx, "materialInsertQuery", materialInsertQuery)
	if err != nil {
		return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("conn.Prepare() materialInsertQuery: %w", err))
	}

	concrServ := NewConcreteTypeMapToProductionService(s.DB, s.Session)
	cementSiloServ := NewCementSiloService(s.DB, s.Session)
	matMapToProdServ := NewMaterialMapToProductionService(s.DB, s.Session)

	// material local cache, key is a production description, val is the database ID.
	// Local cache just for one production batch as it can change.
	materials := make(map[string]int)

	for _, prod := range msg {
		if prod.ConcreteDescr == "" || prod.ProductionDtStart == nil {
			continue
		}

		productionID := fmt.Sprintf("%d", prod.ID)

		concreteTypeID, err := concrServ.mapToProductionWithConn(ctx, conn, prod.ConcreteDescr)
		if err != nil {
			return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("concrServ.MapToProduction(): %w", err))
		}

		now := time.Now()
		var dtStartSet *time.Time
		var dtEndSet *time.Time
		if prod.ProductionDtStart != nil {
			dtStartSet = &now
		}
		if prod.ProductionDtEnd != nil {
			dtEndSet = &now
		}

		var vehicleID *int
		var vehicleSchStateID *int
		var shipmentID *int

		if err := productionStmt.QueryRow(ctx,
			productionSiteID, productionID,
			elkonForBeregFormatDatePtr(prod.ProductionDtStart),
			elkonForBeregFormatDatePtr(prod.ProductionDtEnd),
			prod.UserDescr, prod.VehicleDescr, prod.ConcreteDescr,
			elkonForBeregFormatDatePtr(dtStartSet),
			elkonForBeregFormatDatePtr(dtEndSet),
			concreteTypeID,
			prod.ConcreteQuant,
		).Scan(&vehicleID, &vehicleSchStateID, &shipmentID); err != nil {
			return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("productionInsertQuery Query(): %w", err))
		}

		if prod.ProductionDtEnd == nil {
			continue // production not closed - no farther actions
		}

		materialArgs := make([][]any, 0, models.ElkonProductionMaxMaterialCount)

		// add all materials to a batch
		for matInd := range models.ElkonProductionMaxMaterialCount {
			mat, err := GetMaterialValue(prod, matInd+1)
			if err != nil {
				return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("GetMaterialValue(): %w", err))
			}

			if mat.MatQuant == 0 && mat.MatQuantReq == 0 && mat.MatQuantCorrected == 0 {
				continue
			}
			if mat.MatQuant > 0 {
				mat.MatQuant = mat.MatQuant / 1000
			}
			if mat.MatQuantReq > 0 {
				mat.MatQuantReq = mat.MatQuantReq / 1000
			}
			if mat.MatQuantCorrected > 0 {
				mat.MatQuantCorrected = mat.MatQuantCorrected / 1000
			}

			var cementSiloID *int // can be null in database
			if mat.MatCement {
				// need silo ID
				id, err := cementSiloServ.mapToProductionWithConn(ctx, conn, productionSiteID, mat.MatDescr)
				if err != nil {
					return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("cementSiloServ.MapToProduction(): %w", err))
				}
				cementSiloID = &id
			}

			materialID, ok := materials[mat.MatDescr]
			if !ok {
				// retrieve material ID from database
				keys, err := matMapToProdServ.upsertWithConn(ctx, conn,
					models.MaterialMapToProduction{
						ProductionSiteID: fields.NewFieldIntVal(int64(productionSiteID)),
						ProductionDescr:  fields.NewFieldTextVal(mat.MatDescr),
					},
					*prod.ProductionDtEnd,
				)
				if err != nil {
					return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("matMapToProdServ.Upsert(): %w", err))
				}
				matIntf, ok := keys["id"]
				if !ok {
					return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("Upsert() result has no id field"))
				}
				materialID, ok = matIntf.(int)
				if !ok {
					return s.uploadError(ctx, conn, productionSiteID, &msg, fmt.Errorf("Upsert() result id cast to int error"))
				}
				materials[mat.MatDescr] = materialID
			}
			/*
				production_site_id,
				upload_date_time,
				upload_user_id,
				date_time,
				concrete_type_production_descr,
				concrete_type_id,
				material_production_descr,
				material_id,
				vehicle_production_descr,
				vehicle_id,
				vehicle_schedule_state_id,
				concrete_quant,
				material_quant,
				material_quant_req,
				cement_silo_id,
				production_id

			*/
			materialArgs = append(materialArgs, []any{
				productionSiteID,
				elkonForBeregFormatDate(time.Now()),
				userID,
				elkonForBeregFormatDatePtr(prod.ProductionDtEnd),
				prod.ConcreteDescr,
				concreteTypeID,
				mat.MatDescr,
				materialID,
				prod.VehicleDescr,
				vehicleID,
				vehicleSchStateID,
				prod.ConcreteQuant,
				mat.MatQuant,
				mat.MatQuantReq,
				cementSiloID,
				productionID,
			})
		}

		// pgx batches ran the delete and inserts in one implicit transaction.
		// Keep that atomic replacement while using the ds interfaces.
		if err := ds.WithTx(ctx, conn, func(ctx context.Context, tx ds.Tx) error {
			if _, err := tx.Exec(ctx, materialDeleteStmt.Name(), productionSiteID, productionID); err != nil {
				return fmt.Errorf("materialDeleteQuery: %w", err)
			}
			for _, args := range materialArgs {
				if _, err := tx.Exec(ctx, materialInsertStmt.Name(), args...); err != nil {
					return fmt.Errorf("materialInsertQuery: %w", err)
				}
			}
			return nil
		}); err != nil {
			return s.uploadError(ctx, conn, productionSiteID, &msg, err)
		}
	}

	// trigger
	// _ = EvHandler.PublishEvent(s.Session.SessionID(), "ElkonProduction.Production", nil)

	return nil
}

// ElkonProductionSiteSessionID retrieves the session ID for the given site ID.
// Returns the session ID or error.
func (s *ElkonProductionService) ElkonProductionSiteSessionID(ctx context.Context, siteID int) (string, error) {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return "", fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)

	conn := poolConn.Conn()
	var sessionID string
	if err := conn.QueryRow(ctx,
		`SELECT elkon_session_id
		FROM production_sites
		WHERE id = $1`,
		siteID,
	).Scan(&sessionID); errors.Is(err, ds.ErrNoRows) {
		return "", errElkonProdNotFound
	} else if err != nil {
		return "", fmt.Errorf("conn.QueryRow(): %w", err)
	}
	return sessionID, nil
}

type ElkonMaterialValue struct {
	MatDescr          string
	MatQuant          float64
	MatQuantReq       float64
	MatCement         bool
	MatQuantCorrected float64
}

func GetMaterialValue(s models.ElkonProduction, ind int) (ElkonMaterialValue, error) {
	var err error
	matValue := ElkonMaterialValue{}

	matValue.MatDescr, err = GetFieldStringByName(s, fmt.Sprintf("Mat%dDescr", ind))
	if err != nil {
		return matValue, err
	}

	matValue.MatQuant, err = GetFieldFloatByName(s, fmt.Sprintf("Mat%dQuant", ind))
	if err != nil {
		return matValue, err
	}
	matValue.MatQuantReq, err = GetFieldFloatByName(s, fmt.Sprintf("Mat%dQuantReq", ind))
	if err != nil {
		return matValue, err
	}

	matValue.MatCement, err = GetFieldBoolByName(s, fmt.Sprintf("Mat%dCement", ind))
	if err != nil {
		return matValue, err
	}

	matValue.MatQuantCorrected, err = GetFieldFloatByName(s, fmt.Sprintf("Mat%dQuantCorrected", ind))
	if err != nil {
		return matValue, err
	}

	return matValue, nil
}

func GetFieldValueByName(s any, fieldName string) (any, error) {
	v := reflect.ValueOf(s)
	// If it's a pointer, get the underlying value
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check if the field exists
	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return "", fmt.Errorf("no such field: %s", fieldName)
	}

	return f.Interface(), nil
}

func GetFieldStringByName(s any, fieldName string) (string, error) {
	intfVal, err := GetFieldValueByName(s, fieldName)
	if err != nil {
		return "", err
	}
	strValPtr, ok := intfVal.(*string)
	if !ok {
		strVal, ok := intfVal.(string)
		if !ok {
			return "", fmt.Errorf("GetFieldStringByName(): unable to cast to string, field: %s, intfVal: %v", fieldName, intfVal)
		}
		strValPtr = &strVal
	}
	if strValPtr == nil {
		return "", nil
	}

	return *strValPtr, nil
}

func GetFieldFloatByName(s any, fieldName string) (float64, error) {
	intfVal, err := GetFieldValueByName(s, fieldName)
	if err != nil {
		return 0, err
	}
	flValPtr, ok := intfVal.(*float64)
	if !ok {
		flVal, ok := intfVal.(float64)
		if !ok {
			return 0, fmt.Errorf("GetFieldStringByName(): unable to cast to float64, field: %s", fieldName)
		}
		flValPtr = &flVal
	}

	if flValPtr == nil {
		return 0, nil
	}

	return *flValPtr, nil
}

func GetFieldBoolByName(s any, fieldName string) (bool, error) {
	intfVal, err := GetFieldValueByName(s, fieldName)
	if err != nil {
		return false, err
	}
	bValPtr, ok := intfVal.(*bool)
	if !ok {
		bVal, ok := intfVal.(bool)
		if !ok {
			return false, fmt.Errorf("GetFieldStringByName(): unable to cast to bool, field: %s", fieldName)
		}
		bValPtr = &bVal
	}

	if bValPtr == nil {
		return false, nil
	}

	return *bValPtr, nil
}

// Reuse the upload lease even on error; acquiring another connection here can
// deadlock an exhausted pool. Saving failover data does not depend on DB logging.
func (s *ElkonProductionService) uploadError(ctx context.Context, conn ds.Conn, productionSiteID int, msg *models.ElkonProductionList, uploadErr error) error {
	logMessage := uploadErr.Error()
	if s.FromFailoverFile {
		logMessage = "restoring from failover:" + logMessage
	}
	if conn != nil {
		if _, errLog := conn.Exec(ctx,
			`SELECT elkon_log_upsert( ROW(NULL, NOW(), $1, $2, $3)::elkon_log )`,
			productionSiteID,
			logMessage,
			ElkonMsgTypeErr,
		); errLog != nil {
			logger.Logger.Errorf("error inserting error: %v", errLog)
		}
	}
	if !s.FromFailoverFile {
		if saveErr := ElkonFailoverSrv.Save(&ElkonPacket{ProductionSiteID: productionSiteID, Packet: msg}); saveErr != nil {
			logger.Logger.Errorf("ElkonFailoverSrv.Save(): %v", saveErr)
		}
	}
	return uploadErr
}

func elkonForBeregFormatDate(tm time.Time) string {
	return tm.Format("2006-01-02 15:04:05")
}

func elkonForBeregFormatDatePtr(tm *time.Time) string {
	if tm == nil {
		return time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).Format(dateFromElkonFormatLayout)
	}
	return tm.Format(dateFromElkonFormatLayout)
}
