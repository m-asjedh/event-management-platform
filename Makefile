# Read .env if it exists, so these targets use the same credentials as
# docker-compose.yml. The leading dash means "carry on if the file is missing".
-include .env

# Fallbacks, matching the defaults in docker-compose.yml, so every target works
# with no .env file present.
POSTGRES_USER ?= emp_user
POSTGRES_DB   ?= event_management_platform

.PHONY: up down reset migrate migrate-status migrate-down seed psql db-version

## up: start the database, bring the schema up to date, then start auth and the API
# Order matters: auth reads tables the migrations create, so it must not start first.
up:
	docker compose up -d --wait postgres
	$(MAKE) migrate
	docker compose up -d --wait --build auth api

## down: stop the stack, keep the data
down:
	docker compose down

## reset: stop the stack and delete the data, so the next `up` initialises from scratch
reset:
	docker compose down -v

## migrate: apply any migrations that have not run yet
# Safe to repeat: goose records applied versions in goose_db_version and skips them, so
# there is no "is this the first boot" check to get wrong.
migrate:
	docker compose run --rm --build migrate up

## migrate-status: show which migrations have been applied
migrate-status:
	docker compose run --rm migrate status

## migrate-down: roll back the most recent migration
migrate-down:
	docker compose run --rm migrate down

## seed: fill the migrated database with the scale dataset
# Safe to repeat: the seeder truncates domain and auth rows first, then COPY.
# Only the Compose postgres service: DATABASE_URL is hardcoded to host `postgres`
# in docker-compose.yml, and the binary refuses any other host.
seed:
	docker compose run --rm --build seed

## psql: open a psql shell on the database
psql:
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

## db-version: print the PostgreSQL server version
db-version:
	@docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) \
		-tAc "SELECT version();"
