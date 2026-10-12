package enums

type RoleTypes = string

const (
	RoleTypesOwner         RoleTypes = "owner"
	RoleTypesBoss          RoleTypes = "boss"
	RoleTypesOperator      RoleTypes = "operator"
	RoleTypesManager       RoleTypes = "manager"
	RoleTypesDispatcher    RoleTypes = "dispatcher"
	RoleTypesAccountant    RoleTypes = "accountant"
	RoleTypesLabWorker     RoleTypes = "lab_worker"
	RoleTypesSupplies      RoleTypes = "supplies"
	RoleTypesSales         RoleTypes = "sales"
	RoleTypesPlantDirector RoleTypes = "plant_director"
	RoleTypesSupervisor    RoleTypes = "supervisor"
	RoleTypesVehicleOwner  RoleTypes = "vehicle_owner"
	RoleTypesClient        RoleTypes = "client"
	RoleTypesWeighing      RoleTypes = "weighing"
)
