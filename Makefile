# Read .env if it exists, so these targets use the same credentials as
# docker-compose.yml. The leading dash means "carry on if the file is missing".
-include .env

# Fallbacks, matching the defaults in docker-compose.yml, so every target works
# with no .env file present.
POSTGRES_USER ?= emp_user
POSTGRES_DB   ?= event_management_platform

.PHONY: up down reset psql db-version

## up: start the stack and wait until it is healthy
up:
	docker compose up -d --wait

## down: stop the stack, keep the data
down:
	docker compose down

## reset: stop the stack and delete the data, so the next `up` initialises from scratch
reset:
	docker compose down -v

## psql: open a psql shell on the database
psql:
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

## db-version: print the PostgreSQL server version
db-version:
	@docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) \
		-tAc "SELECT version();"
