# TestAcc Session — 2026-05-11

## Goal

Add acceptance tests (`TestAcc*`) to the provider so CI can validate CRUD behaviour
against a real HTTP server.  The work lives on branch **`feat/acceptance-tests`**.

---

## What was built this session

### `testacc/server/` — Django 5 + DRF API server

Minimal server that implements the Widgets resource exactly as the provider expects.
It also serves its own OAS3 schema via **drf-spectacular**, which the provider loads at
startup through `OPENAPI_SPEC`.

```
testacc/server/
├── manage.py
├── requirements.txt          # django, djangorestframework, drf-spectacular
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── config/
│   ├── settings.py           # SQLite, no auth, COMPONENT_SPLIT_REQUEST=True
│   ├── urls.py               # /health/  /api/schema/  + widgets routes
│   └── wsgi.py
└── widgets/
    ├── models.py             # Widget(name, size, created_at)
    ├── serializers.py        # id + created_at read-only
    ├── views.py              # create / retrieve / partial_update / destroy
    └── urls.py               # POST /api/v1/widgets/
                              # GET|PATCH|DELETE /api/v1/widgets/<pk>/
```

**Key design decision — `COMPONENT_SPLIT_REQUEST = True`**
Without this flag drf-spectacular includes `id` and `created_at` in the POST request
body schema.  The provider's `extractRequestBodyFields()` treats every property in
that schema as writable, so `id` and `created_at` would be wrongly user-settable in
Terraform.  With the flag, `WidgetRequest` only contains `name` and `size`.

**`/health/`** — used by the CI readiness probe (`until curl … /health/`).

**`/api/schema/`** — serves the live OAS3 YAML consumed by the provider via
`OPENAPI_SPEC`.

### `GNUmakefile` — two targets added/updated

```makefile
make server    # install deps, migrate, runserver :8000
make testacc   # sets OPENAPI_SPEC + OPENAPI_URL then runs TF_ACC=1 go test
```

`OPENAPI_SPEC` is derived automatically from `OPENAPI_URL` (defaults to
`http://localhost:8000`), so contributors only need to start the server and run
`make testacc`.

### `.github/workflows/testacc.yml`

Matrix over Terraform 1.9 and 1.10.  Steps:

1. Checkout, setup-go, setup-terraform, setup-python (3.12, pip-cached)
2. `pip install -r testacc/server/requirements.txt`
3. `python manage.py migrate --run-syncdb && python manage.py runserver 0.0.0.0:8000 &`
4. `until curl --silent --fail http://localhost:8000/health/; do sleep 1; done`
5. `go mod download`
6. `make testacc` with `OPENAPI_URL=http://localhost:8000`

---

## What still needs to be done

### 1. Write the actual `TestAcc*` Go test functions

No acceptance tests exist yet — the infrastructure is in place but there are zero
`TestAcc*` functions.  They belong in a new file, suggested path:

```
internal/provider/provider_acc_test.go
```

Typical structure (hashicorp/terraform-plugin-testing pattern):

```go
//go:build acctest

package provider_test

import (
    "testing"
    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWidget_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: `
                    resource "openapi_widget" "test" {
                        name = "alpha"
                        size = 3
                    }
                `,
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("openapi_widget.test", "name", "alpha"),
                    resource.TestCheckResourceAttr("openapi_widget.test", "size", "3"),
                    resource.TestCheckResourceAttrSet("openapi_widget.test", "id"),
                    resource.TestCheckResourceAttrSet("openapi_widget.test", "created_at"),
                ),
            },
        },
    })
}
```

The provider factories helper needs to be added (see existing unit test files for
the pattern used — check `internal/provider/*_test.go`).

Test cases to cover:
- `TestAccWidget_basic` — create, verify computed fields
- `TestAccWidget_update` — create then update `name` / `size`
- `TestAccWidget_disappears` — delete outside Terraform, verify plan shows re-create
- `TestAccWidget_importState` — `terraform import`
- `TestAccWidget_nullSize` — omit `size`, verify it stays null

### 2. Check `TestAcc` build tag strategy

The `GNUmakefile` currently runs `go test ./...` with `TF_ACC=1`.  If acceptance
tests use a `//go:build acctest` tag (common pattern to keep them out of unit test
runs), the Makefile and workflow need `-tags acctest`.  Alternatively, guard with
`if os.Getenv("TF_ACC") == "" { t.Skip(...) }` (no build tag needed).

### 3. Add more resources to the test server (optional)

The server currently only implements `widgets`.  To test more provider features
(e.g. `x-immutable`, `x-sensitive`, `x-computed`, nested objects, arrays), add more
Django apps mirroring the other `testdata/` fixtures:

- `internal/spec/testdata/validation.yaml` → add `validations` app
- `internal/spec/testdata/schema_components.yaml` → add a `components` app

Each new app needs its own `models.py`, `serializers.py`, `views.py`, `urls.py`
and must be registered in `config/settings.INSTALLED_APPS` and `config/urls.py`.

### 4. Provider factory / env-var wiring

The acceptance tests need to read `OPENAPI_URL` and `OPENAPI_SPEC` from the
environment and pass them to the provider under test.  Look at how `provider.go`
reads these env vars and replicate that in the test factory.

---

## How to run locally

```bash
# Terminal 1 — start the server
make server

# Terminal 2 — run acceptance tests
make testacc
```

Override server URL if needed:
```bash
OPENAPI_URL=http://192.168.1.10:8000 make testacc
```

Via Docker:
```bash
cd testacc/server
docker compose up          # starts on :8000
# then in another terminal:
make testacc
```

---

## Environment variables used by the provider

| Variable | Value in testacc | Purpose |
|---|---|---|
| `TF_ACC` | `1` | Enables Terraform acceptance tests |
| `OPENAPI_SPEC` | `http://localhost:8000/api/schema/` | OAS3 spec URL (served by Django) |
| `OPENAPI_URL` | `http://localhost:8000` | Base URL for API calls |
| `OPENAPI_TOKEN` | *(unset — server has no auth)* | Bearer token |

---

## Relevant files

| Path | Role |
|---|---|
| `testacc/server/` | Django test server |
| `.github/workflows/testacc.yml` | CI acceptance test pipeline |
| `GNUmakefile` | `server` and `testacc` targets |
| `internal/spec/testdata/widgets.yaml` | Original hand-written spec (reference only) |
| `internal/spec/discover.go:104` | `IDField` hardcoded to `"id"` |
| `internal/spec/loader.go:33` | Loader accepts `http://` URLs |
