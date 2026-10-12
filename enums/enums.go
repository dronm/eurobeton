// Package enums provides access to application enums.
package enums

import (
	md "github.com/dronm/crudifier/v2/metadata"
)

// InitEnums is used for crudifier enum initialization.
func InitEnums() {
	md.Enums = map[string][]string{
		"locales": {
			"ru",
		},
		"data_types": {
			"users",
			"clients",
			"suppliers",
			"pump_vehicles",
			"drivers",
		},
		"role_types": {
			"owner",
			"boss",
			"operator",
			"manager",
			"dispatcher",
			"accountant",
			"lab_worker",
			"supplies",
			"sales",
			"plant_director",
			"supervisor",
			"vehicle_owner",
			"client",
			"weighing",
		},
		"unload_types": {
			"pump",
			"band",
			"none",
		},
		"vehicle_states": {
			"shift",
			"free",
			"assigned",
			"busy",
			"left_for_dest",
			"at_dest",
			"left_for_base",
			"out_from_shift",
			"out",
			"shift_added",
		},
		"sms_types": {
			"order",
			"ship",
			"remind",
			"procur",
			"order_for_pump_ins",
			"order_for_pump_upd",
			"order_for_pump_del",
			"order_for_pump_ship",
			"remind_for_pump",
			"client_thank",
			"vehicle_zone_violation",
			"vehicle_tracker_malfunction",
			"efficiency_warn",
			"material_balance",
			"mixer_route",
			"order_cancel",
			"tm_invite",
			"new_pwd",
		},
		"email_types": {
			"new_account",
			"reset_pwd",
		},
		"doc_types": {
			"order",
			"material_procurement",
			"shipment",
			"material_fact_consumption",
			"material_fact_consumption_correction",
			"material_fact_balance_correction",
			"cement_silo_reset",
			"cement_silo_balance_reset",
			"material_movement",
		},
		"reg_types": {
			"material",
			"material_fact",
			"cement",
			"material_consumption",
		},
		"call_types": {
			"in",
			"out",
		},
		"client_kinds": {
			"buyer",
			"acc",
			"else",
		},
		"offer_results": {
			"no",
			"seems_no",
			"will_think",
			"make_order",
		},
		"production_plant_types": {
			"elkon",
			"ammann",
		},
		"period_value_types": {
			"destination_price",
			"destination_price_for_driver",
			"demurrage_cost_per_hour",
			"water_ship_cost",
			"min_quant_for_ship_cost",
		},
		"inform_types": {
			"email",
			"telegram",
			"whatsup",
			"sms",
		},
	}
}
