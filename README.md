<!--
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
-->

# Terraform Provider for Apache Pulsar

Terraform provider for managing Pulsar clusters, tenants, namespaces, topics, schemas, functions, connectors, subscriptions, permissions, and packages.

## Requirements

- Terraform 1.2.7 or newer
- Go 1.24.4 or newer for source builds

## Usage

```hcl
terraform {
  required_providers {
    pulsar = {
      source = "streamnative/pulsar"
    }
  }
}

provider "pulsar" {
  web_service_url = "http://localhost:8080"
}
```

Use `token` or `PULSAR_TOKEN`/`PULSAR_AUTH_TOKEN` when authentication is enabled. See [provider configuration](docs/index.md) for TLS and OAuth options.

## Documentation

Generated resource docs are canonical:

- [Cluster](docs/resources/cluster.md)
- [Tenant](docs/resources/tenant.md)
- [Namespace](docs/resources/namespace.md)
- [Topic](docs/resources/topic.md)
- [Schema](docs/resources/schema.md)
- [Function](docs/resources/function.md)
- [Source](docs/resources/source.md)
- [Sink](docs/resources/sink.md)
- [Subscription](docs/resources/subscription.md)
- [Permission grant](docs/resources/permission_grant.md)
- [Package](docs/resources/package.md)

Configuration samples live under [`examples/`](examples/).

## Development

```shell
make build
make test
go generate ./...
```

For local Terraform runs, add `version = "0.0.1"` to `required_providers.pulsar`, run `make build-dev VERSION=0.0.1`, then `terraform init -upgrade`. Acceptance tests require a Pulsar cluster:

```shell
make run-pulsar-in-docker
make testacc
make remove-pulsar-from-docker
```

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for contribution checks.

## License

[Apache License 2.0](LICENSE)
