#!make

CGO_ENABLED?=0
GOCMD=CGO_ENABLED=$(CGO_ENABLED) go
GOTEST=gotestsum -f=testname --
GOVET=$(GOCMD) vet

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all
all: help

.PHONY: test
test: ## Run tests.
	@echo "${YELLOW}Running tests...${RESET}"
	$(GOTEST) -v ./...

.PHONY: integration_tests
integration_tests: ## Run integrations tests. Requires LLM env variables to be set.
	@echo "${YELLOW}Running tests + integration tests...${RESET}"
	$(GOTEST) -tags=integration -v ./...

.PHONY: integration_anthropic
integration_anthropic: ## Run integrations tests for Anthropic. Requires ANTHROPIC_API_KEY env variable to be set.
	@echo "${YELLOW}Running tests + integration tests...${RESET}"
	$(GOTEST) -tags=integration -v ./anthropic/...


## Help:
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
