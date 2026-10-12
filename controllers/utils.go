package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	crud "github.com/dronm/crudifier/v2"
	"github.com/dronm/gobizapp/errs"
	"github.com/dronm/gobizapp/logger"
	"github.com/dronm/session"
)

// ReportErrors should be set at initialization.
var ReportErrors bool

func ServeError(c *gin.Context, httpErr int, fnName string, err error) {
	errText := fmt.Sprintf("%s: %v", fnName, err)

	// log real message here
	logger.Logger.Error(errText)

	var usrMsg string
	var usrCode errs.ErrorCode

	var pubErr errs.PublicError
	var validErr *crud.ValidationError // all validation errors

	if errors.As(err, &pubErr) {
		usrMsg = pubErr.Error()
		usrCode = pubErr.Code()

	} else if errors.As(err, &validErr) {
		usrMsg = validErr.Error()
		usrCode = errs.ValidationFailed

	}else {
		switch httpErr {
		case http.StatusInternalServerError:
			usrCode = errs.InternalError
		case http.StatusBadRequest:
			usrCode = errs.BadRequest
		default:
			usrCode = errs.UnknownError
		}
		if ReportErrors {
			usrMsg = errText
		}else{
			usrMsg = errs.ErrorDescr(usrCode)
		}
	}

	c.JSON(httpErr, gin.H{
		"error": usrMsg,
		"code":  string(usrCode),
	})
}

func ParamAsInt(c *gin.Context, key string) (int, error) {
	paramStr := c.Param(key)
	return strconv.Atoi(paramStr)
}

func ParamAsDate(c *gin.Context, key string) (time.Time, error) {
	paramStr := c.Param(key)
	return time.Parse("2006-01-02", paramStr)
}

func ParamAsDateTimeTZ(c *gin.Context, key string) (time.Time, error) {
	paramStr := c.Param(key)
	return time.Parse(time.RFC3339, paramStr)
}

func ParseCollectionParams(c *gin.Context) (crud.CollectionParams, error) {
	var params crud.CollectionParams

	encodedParam := c.DefaultQuery("params", "")
	decodedParam, err := url.QueryUnescape(encodedParam)
	if err != nil {
		return params, err
	}
	if decodedParam != "" {
		if err := json.Unmarshal([]byte(decodedParam), &params); err != nil {
			return params, err
		}
	}
	return params, nil
}

func getTypedSession(c *gin.Context, funcName string, sess any) session.Session {
	sessTyped, ok := sess.(session.Session)
	if !ok {
		ServeError(c, http.StatusInternalServerError,
			funcName+" casting to session.Session: gin session param could not be cast to session.Session",
			errs.NewPublicError(http.StatusUnauthorized, "NO_SESSION"),
		)
		return nil
	}

	return sessTyped
}

func GetSessionOrNil(c *gin.Context, funcName string) session.Session {
	sess, exists := c.Get("session")
	if !exists {
		return nil // no session
	}
	return getTypedSession(c, funcName, sess)
}

// GetSession retrieves session from manager, if it is not loaded yet,
// it loads it.
// This function should be used for modifying sessions as it makes the
// session actually start.
func GetSession(c *gin.Context, funcName string) session.Session {
	sess, started := c.Get("session")
	if !started {
		loader, ok := c.Get("session_loader")
		if !ok {
			ServeError(c, http.StatusInternalServerError, funcName+"c.Get()", fmt.Errorf("session_loader not found"))
			return nil
		}

		var err error
		sess, err = loader.(func() (session.Session, error))()
		if err != nil {
			ServeError(c, http.StatusInternalServerError, funcName+"loader", fmt.Errorf("session_loader failed: %v", err))
			return nil
		}
	}

	return getTypedSession(c, funcName, sess)
}

// GetCollectionParams returns params pointer or nil on error, sends error headers
func GetCollectionParams(c *gin.Context, funcName string) *crud.CollectionParams {
	params, err := ParseCollectionParams(c)
	if err != nil {
		ServeError(c, http.StatusBadRequest, funcName+" json.Unmarshal()", err)
		return nil
	}

	return &params
}

// GetDetailID returns object ID pointer or nil on error, sends error headers
func GetDetailID(c *gin.Context, funcName string) *int {
	id, err := ParamAsInt(c, "id")
	if err != nil {
		ServeError(c, http.StatusBadRequest, funcName+" ParamsAsInt()", err)
		return nil
	}

	return &id
}

func CheckBindModel(c *gin.Context, funcName string, model any) bool {
	if err := c.ShouldBind(model); err != nil {
		ServeError(c, http.StatusInternalServerError, funcName+" ShouldBind()", err)
		return false
	}

	return true
}

func ValidateModel(c *gin.Context, funcName string, model any) bool {
	if !CheckBindModel(c, funcName, model) {
		return false
	}
	// if err := crud.ValidateModel(model); err != nil {
	// 	if valErr, ok := err.(*crud.ValidationError); ok {
	// 		//full error to client
	// 		ServeError(c, http.StatusBadRequest, valErr.Error(), valErr)
	// 		return false
	// 	}
	// 	ServeError(c, http.StatusInternalServerError, funcName+" ValidateModel()", err)
	// 	return false
	// }
	return true
}

