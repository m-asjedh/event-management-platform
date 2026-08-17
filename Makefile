# Read .env if it exists, so these targets use the same credentials as
# docker-compose.yml. The leading dash means "carry on if the file is missing".
-include .env

# Fallbacks, matching the defaults in docker-compose.yml, so every target works
# with no .env file present.
POSTGRES_USER ?= emp_user
POSTGRES_DB   ?= event_management_platform

.PHONY: up down reset migrate migrate-status migrate-down lint seed test contract agent agent-scenarios gen-api check-generated frontend frontend-build frontend-test psql db-version

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

## lint: check openapi/openapi.yaml with Redocly (file validity only, not the live server)
lint:
	docker compose run --rm lint

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

## test: run Go tests inside Compose, against the local database
# Does not start the API. Contract checks against the live server are `make contract`.
test:
	docker compose run --rm --build test

## contract: start the API and check live responses against openapi/openapi.yaml
# Seeded DB + real session cookies. Not lint — each case hits the running server.
contract:
	docker compose up -d --wait postgres
	$(MAKE) migrate
	docker compose run --rm --build seed
	docker compose up -d --wait --build api
	docker compose run --rm --build contract

## agent: read-only questions against the running API, as a real user
# Sign-in is Better Auth HTTP. The binary only issues GET to the API.
# QUESTION, AGENT_EMAIL, AGENT_PASSWORD are optional env vars.
# No model key. The three scripted questions are `make agent-scenarios`.
agent:
	docker compose up -d --wait --build auth api
	docker compose run --rm --build -e QUESTION -e AGENT_EMAIL -e AGENT_PASSWORD agent $(ARGS)

## agent-scenarios: the three B8 questions against the seeded API
agent-scenarios:
	docker compose up -d --wait --build auth api
	docker compose run --rm --build -T agent -scenarios

## gen-api: TypeScript types from openapi/openapi.yaml (never hand-edit the output)
gen-api:
	docker compose run --rm gen-api

## check-generated: fail if committed types are stale vs the spec
check-generated:
	$(MAKE) gen-api
	git diff --exit-code -- frontend/src/generated/api.ts

## frontend: Vite dev server on :5173 (not part of make up)
frontend:
	docker compose --profile web up -d --wait --build frontend

## frontend-build: tsc + vite build, for CI
frontend-build:
	docker compose --profile tools build frontend-build

## frontend-test: Vitest schedule tests (not part of make test)
frontend-test:
	docker compose --profile tools run --rm frontend-test

## psql: open a psql shell on the database
psql:
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

## db-version: print the PostgreSQL server version
db-version:
	@docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) \
		-tAc "SELECT version();"
