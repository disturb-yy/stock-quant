SHELL := /bin/bash
.DEFAULT_GOAL := help
GO_BUILDVCS_FLAG := $(if $(shell git rev-parse --show-toplevel 2>/dev/null),,-buildvcs=false)

.PHONY: help setup-python doctor test test-go test-python test-tools test-config test-boundaries check build run

help:
	@printf '%s\n' \
	  'Stock Quant development targets:' \
	  '  setup-python  Create the Python virtual environment and install dependencies' \
	  '  doctor        Check required tool versions and provider configuration' \
	  '  test          Run Go, Python, tool, and boundary checks' \
	  '  check         Run the complete local CI quality gate' \
	  '  build         Build Go packages (available as packages are added)' \
	  '  run           Run the process-level health command'

setup-python:
	python3 -m venv .venv
	.venv/bin/python -m pip install --requirement python/requirements.txt

doctor:
	./tools/env-check.sh

test: test-go test-python test-tools test-config test-boundaries

test-go:
	@packages=$$(go list $(GO_BUILDVCS_FLAG) ./... 2>/dev/null) || { go list $(GO_BUILDVCS_FLAG) ./...; exit 1; }; \
	if [[ -n "$$packages" ]]; then go test $(GO_BUILDVCS_FLAG) ./...; else \
	  printf '%s\n' 'No Go packages exist yet; continue with the next scaffold ticket.'; \
	fi

test-python:
	@if [[ -d python/tests ]]; then \
	  if [[ ! -x .venv/bin/python ]]; then \
	    printf '%s\n' 'Python test environment missing; run: make setup-python' >&2; exit 1; \
	  fi; \
	  PYTHONPATH=python .venv/bin/python -m unittest discover -s python/tests; \
	else \
	  printf '%s\n' 'No Python tests exist yet; Python tests will be added with the worker ticket.'; \
	fi

test-tools:
	python3 -m unittest discover -s tools -p 'test_*.py'
	bash tools/check_pipeline_test.sh

test-config:
	./tools/env-check-test.sh

test-boundaries:
	./tools/check-domain-imports.sh
	./tools/check-domain-imports-test.sh

check:
	./tools/check.sh

build:
	go build $(GO_BUILDVCS_FLAG) ./...

run:
	go run $(GO_BUILDVCS_FLAG) ./cmd/stockquant health
