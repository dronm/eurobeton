#!make
include .env

# compute dynamic values
COMMIT := $(shell git rev-parse --short HEAD)
DATE   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

prod:
	@go build -ldflags "-s -w \
		-X 'main.Version=$(VERSION)' \
		-X 'main.Commit=$(COMMIT)' \
		-X 'main.BuildDate=$(DATE)'" \
	-o ebeton .
migup: 
	@migrate -database "$(DB_CONN)" -path migrations up 1
runsql:
	@php ./deploy/run_sql.php "$(DB_CONN)" $(word 2, $(MAKECMDGOALS)) "./migrations/sql"
migdown: 
	@migrate -database "$(DB_CONN)" -path migrations down 1
migfix: 
	@migrate -database "$(DB_CONN)" -path migrations force $(word 2, $(MAKECMDGOALS))
migadd: 
	@migrate create -ext sql -dir migrations -seq $(word 2, $(MAKECMDGOALS))
migver: 
	@migrate -database "$(DB_CONN)" -path migrations version

enum:
	@php ./md/enums.php $(word 2, $(MAKECMDGOALS)) $(CURDIR) $(CLIENT_DIR)
model:
	@php ./md/models.php $(word 2, $(MAKECMDGOALS)) $(CURDIR) $(CLIENT_DIR)
controller:
	@php ./md/controllers.php $(word 2, $(MAKECMDGOALS)) $(CURDIR) $(CLIENT_DIR)
component:
	@php ./md/components.php $(word 2, $(MAKECMDGOALS)) $(CURDIR) $(CLIENT_DIR)
trigger:
	@php ./md/sql_trigger.php $(CURDIR) $(word 2, $(MAKECMDGOALS)) $(word 3, $(MAKECMDGOALS))
ref:
	@php ./md/refs.php $(word 2, $(MAKECMDGOALS)) $(CURDIR) $(CLIENT_DIR)
permission:
	@php ./md/role_permissions.php $(word 2, $(MAKECMDGOALS)) $(CURDIR) $(CLIENT_DIR)	
%:
	@:

