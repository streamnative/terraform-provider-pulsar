# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build           # Compile terraform-provider-pulsar binary
make build-dev       # Build and install to ~/.terraform.d/plugins for local testing
make test            # Run unit tests with go test ./...
make testacc         # Run acceptance tests (requires running Pulsar cluster, TF_ACC=1)
make lint            # Run golangci-lint and tfproviderlint
make fmt             # Format code with gofmt
make tools           # Install required linting tools
```

### Running a Single Test
```bash
go test -v -run TestAccPulsarNamespace_basic ./pulsar/
```

### Local Pulsar for Testing
```bash
make run-pulsar-in-docker          # Start Pulsar in Docker for acceptance tests
make remove-pulsar-from-docker     # Stop and remove the container
```

For clusters without topic-level policies:
```bash
make run-pulsar-in-docker-no-topic-policies
make testacc-no-topic-policies
```

## Architecture

This is a Terraform provider for Apache Pulsar using the HashiCorp terraform-plugin-sdk/v2.

### Key Components

- **`main.go`**: Entry point, serves the provider via `plugin.Serve`
- **`pulsar/provider.go`**: Provider schema and configuration; creates `PulsarClientBundle` with both v2 and v3 API clients
- **`pulsar/resource_pulsar_*.go`**: Individual resource implementations (CRUD operations)
- **`pkg/admin/`**: Pulsar admin client wrapper with OAuth2 and token authentication support
- **`pkg/authentication/`**: Authentication type definitions

### Provider Resources

| Resource | File |
|----------|------|
| pulsar_cluster | `resource_pulsar_cluster.go` |
| pulsar_tenant | `resource_pulsar_tenant.go` |
| pulsar_namespace | `resource_pulsar_namespace.go` |
| pulsar_topic | `resource_pulsar_topic.go` |
| pulsar_schema | `resource_pulsar_schema.go` |
| pulsar_source | `resource_pulsar_source.go` |
| pulsar_sink | `resource_pulsar_sink.go` |
| pulsar_function | `resource_pulsar_function.go` |
| pulsar_subscription | `resource_pulsar_subscription.go` |
| pulsar_permission_grant | `resource_pulsar_permission_grant.go` |
| pulsar_package | `resource_pulsar_package.go` |

### Client Structure

The provider maintains two Pulsar admin clients (`PulsarClientBundle`):
- `Client`: Default API version (v2) for most operations
- `V3Client`: API v3 for features requiring newer API

Both clients are created during provider configuration and passed to resources via the schema's meta interface.

### Test Patterns

- Unit tests: `*_unit_test.go` files
- Acceptance tests: `*_test.go` files with `TestAcc` prefix, require `TF_ACC=1`
- Tests use the `TestAccPulsar<Resource>_<Scenario>` naming convention

## Code Style

- Run `make fmtcheck` before commits (enforced by make targets)
- Resource files follow `resource_pulsar_<noun>.go` naming
- Schema attributes use Terraform snake_case conventions
- Use `goimports` for import ordering

## Environment Variables

Provider configuration can use these environment variables:
- `WEB_SERVICE_URL` / `PUSLAR_WEB_SERVICE_URL`: Pulsar web service URL
- `PULSAR_TOKEN` / `PULSAR_AUTH_TOKEN`: Authentication token
- `PULSAR_API_VERSION`: API version (default: 0 for auto)
- `PULSAR_TLS_*`: TLS configuration paths
- `PULSAR_KEY_FILE` / `PULSAR_KEY_FILE_PATH`: OAuth2 private key file

## Documentation Generation

```bash
go generate ./...   # Regenerates docs using tfplugindocs
```

Documentation templates are in `templates/`, generated docs go to `docs/`.
