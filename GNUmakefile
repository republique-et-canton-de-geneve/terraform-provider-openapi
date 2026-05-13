default: testacc

VENV     := testacc/server/.venv
VENV_PY  := $(VENV)/bin/python
VENV_PIP := $(VENV)/bin/pip

# Format all Go source files
.PHONY: fmt
fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

# Install golangci-lint v2 if missing, then lint
.PHONY: lint
lint:
	@which golangci-lint > /dev/null 2>&1 || \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	golangci-lint run --timeout 10m ./...

# Run unit tests
.PHONE: test
test:
	go test ./...

# Start the acceptance-test API server manually (requires Python 3.12+).
# Useful for interactive debugging; testacc manages the server lifecycle automatically.
.PHONY: server
server:
	python3 -m venv $(VENV)
	$(VENV_PIP) install -q -r testacc/server/requirements.txt
	cd testacc/server && ../../$(VENV_PY) manage.py migrate --run-syncdb -v 0
	cd testacc/server && ../../$(VENV_PY) manage.py runserver 0.0.0.0:8000

# Run acceptance tests: creates a venv, starts the server, waits for it to be ready,
# runs the tests, then stops the server and removes the SQLite database regardless of outcome.
# The venv is kept between runs; only the database is wiped on each run.
# Override OPENAPI_URL to point at a remote server — local server is not started in that case.
.PHONY: testacc
testacc:
	@_root=$$(pwd); \
	_url=$${OPENAPI_URL:-http://localhost:8000}; \
	if [ -z "$$OPENAPI_URL" ]; then \
		echo "==> Creating/updating venv"; \
		python3 -m venv $(VENV); \
		$$_root/$(VENV_PIP) install -q -r testacc/server/requirements.txt; \
		echo "==> Wiping existing database"; \
		rm -f testacc/server/db.sqlite3; \
		echo "==> Running migrations"; \
		cd testacc/server && $$_root/$(VENV_PY) manage.py migrate --run-syncdb -v 0 && cd $$_root; \
		echo "==> Starting server"; \
		cd testacc/server && $$_root/$(VENV_PY) manage.py runserver 0.0.0.0:8000 >/dev/null 2>&1 & _pid=$$! && cd $$_root; \
		trap "echo; echo '==> Stopping server'; kill $$_pid 2>/dev/null || true; echo '==> Removing database'; rm -f $$_root/testacc/server/db.sqlite3" EXIT INT TERM; \
		printf '==> Waiting for server'; \
		until curl --silent --fail $$_url/health/ >/dev/null 2>&1; do printf '.'; sleep 1; done; \
		echo ' ready.'; \
	fi; \
	OPENAPI_SPEC=$$_url/api/schema/ OPENAPI_URL=$$_url TF_ACC=1 \
		go test ./... -v $(TESTARGS) -timeout 120m
