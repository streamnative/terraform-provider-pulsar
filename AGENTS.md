# Repository Guidelines

## Project Structure & Module Organization
Provider logic lives in `pulsar/`, where every Terraform resource keeps its schema, CRUD helpers, and matching `_test.go` files (acceptance tests sit beside the resource they validate). Shared Pulsar clients and auth helpers live in `pkg/admin` and `pkg/authentication`. CLI entrypoints and provider wiring live in `main.go` and `pulsar/provider.go`. Supporting assets include `docs/` (registry docs), `examples/` (ready-to-run HCL), `templates/` (doc generation), and `hack/` (utility scripts such as `pulsar-docker.sh`). Keep new files in the closest existing module to simplify doc generation and release automation.

## Build, Test, and Development Commands
- `make build`: compiles `terraform-provider-pulsar` for the current platform after ensuring Go formatting.
- `make build-dev`: builds, then stages the binary under `~/.terraform.d/plugins/registry.terraform.io/streamnative/pulsar/<version>/<os_arch>` for local Terraform runs.
- `make test`: runs `go test ./...` with formatting checks.
- `make testacc` / `make testacc-no-topic-policies`: executes acceptance tests with `TF_ACC=1`, optionally disabling topic-policy features via `PULSAR_TEST_CONFIG`.
- `make run-pulsar-in-docker`: spins up a compatible Pulsar stack for local acceptance runs; `make remove-pulsar-from-docker` tears it down.

## Coding Style & Naming Conventions
Code must stay `gofmt`-clean; `scripts/gofmtcheck.sh` enforces this and make targets run it automatically. Use idiomatic Go (tabs for indentation, CamelCase exported symbols) and keep resource helpers in files named `resource_pulsar_<noun>.go`. Prefer `goimports` ordering and run `make lint` (golangci-lint plus `tfproviderlint`) before submitting. Schema attributes follow Terraform snake_case conventions; keep provider docs synchronized by updating `templates/` and regenerating docs when schemas change.

## Testing Guidelines
Unit tests belong near the code under `pulsar/` and should use `Test<Resource>_<Behavior>` naming. Acceptance tests follow Terraform patterns (`TestAccPulsar<Resource>_<Scenario>`) and require a running Pulsar cluster plus `TF_ACC=1`. Use `make run-pulsar-in-docker` to bootstrap a local cluster; export tokens via env vars instead of hardcoding. Capture any new fixtures under `pulsar/testdata` and keep them minimal.

## Commit & Pull Request Guidelines
Recent history favors informative prefixes with optional scopes, e.g., `Feature: standalone topic permission grant (#162)` or `Docs: make doc generation compatible...`. Follow that style: a concise imperative summary, optional category, and a trailing issue/PR reference. Each PR should describe motivation, include reproduction or testing notes (`make test`, `make testacc`), and link to the relevant GitHub issue. Add screenshots only when UI artifacts (docs, diagrams) change. Confirm lint/test status before requesting review and mention any configuration steps contributors must perform.
