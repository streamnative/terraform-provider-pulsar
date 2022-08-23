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

# `terraform-provider-pulsar`

A [Terraform][1] provider for managing [Apache Pulsar Entities][2].

<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 250 60.15">
  <path class="text" fill="#000" d="M77.35 7.86V4.63h-3v3.23h-1.46V.11h1.51v3.25h3V.11h1.51v7.75zm7 0h-1.2l-.11-.38a3.28 3.28 0 0 1-1.7.52c-1.06 0-1.52-.7-1.52-1.66 0-1.14.51-1.57 1.7-1.57h1.4v-.62c0-.62-.18-.84-1.11-.84a8.46 8.46 0 0 0-1.61.17L80 2.42a7.89 7.89 0 0 1 2-.26c1.83 0 2.37.62 2.37 2zm-1.43-2.11h-1.08c-.48 0-.61.13-.61.55s.13.56.59.56a2.37 2.37 0 0 0 1.1-.29zM87.43 8a7.12 7.12 0 0 1-2-.32l.2-1.07a6.77 6.77 0 0 0 1.73.24c.65 0 .74-.14.74-.56s-.07-.52-1-.73c-1.42-.33-1.59-.68-1.59-1.76s.49-1.65 2.16-1.65a8 8 0 0 1 1.75.2l-.14 1.11a10.66 10.66 0 0 0-1.6-.16c-.63 0-.74.14-.74.48s0 .48.82.68c1.63.41 1.78.62 1.78 1.77S89.19 8 87.43 8zm6.68-.11V4c0-.3-.13-.45-.47-.45a4.14 4.14 0 0 0-1.52.45v3.86h-1.46V0l1.46.22v2.47a5.31 5.31 0 0 1 2.13-.54c1 0 1.32.65 1.32 1.65v4.06zm2.68-6.38V.11h1.46v1.37zm0 6.38V2.27h1.46v5.59zm2.62-5.54c0-1.4.85-2.22 2.83-2.22a9.37 9.37 0 0 1 2.16.25l-.17 1.25a12.21 12.21 0 0 0-1.95-.2c-1 0-1.37.34-1.37 1.16V5.5c0 .81.33 1.16 1.37 1.16a12.21 12.21 0 0 0 1.95-.2l.17 1.25a9.37 9.37 0 0 1-2.16.25c-2 0-2.83-.81-2.83-2.22zM107.63 8c-2 0-2.53-1.06-2.53-2.2V4.36c0-1.15.54-2.2 2.53-2.2s2.53 1.06 2.53 2.2v1.41c.01 1.15-.53 2.23-2.53 2.23zm0-4.63c-.78 0-1.08.33-1.08 1v1.5c0 .63.3 1 1.08 1s1.08-.33 1.08-1V4.31c0-.63-.3-.96-1.08-.96zm6.64.09a11.57 11.57 0 0 0-1.54.81v3.6h-1.46v-5.6h1.23l.1.62a6.63 6.63 0 0 1 1.53-.73zM120.1 6a1.73 1.73 0 0 1-1.92 2 8.36 8.36 0 0 1-1.55-.16v2.26l-1.46.22v-8h1.16l.14.47a3.15 3.15 0 0 1 1.84-.59c1.17 0 1.79.67 1.79 1.94zm-3.48.63a6.72 6.72 0 0 0 1.29.15c.53 0 .73-.24.73-.75v-2c0-.46-.18-.71-.72-.71a2.11 2.11 0 0 0-1.3.51zM81.78 19.54h-8.89v-5.31H96.7v5.31h-8.9v26.53h-6z"/>
  <path class="text" fill="#000" d="M102.19 41.77a24.39 24.39 0 0 0 7.12-1.1l.91 4.4a25 25 0 0 1-8.56 1.48c-7.31 0-9.85-3.39-9.85-9V31.4c0-4.92 2.2-9.08 9.66-9.08s9.13 4.35 9.13 9.37v5h-13v1.2c.05 2.78 1.05 3.88 4.59 3.88zM97.65 32h7.41v-1.18c0-2.2-.67-3.73-3.54-3.73s-3.87 1.53-3.87 3.73zm28.54-4.33a45.65 45.65 0 0 0-6.19 3.39v15h-5.83V22.79h4.92l.38 2.58a26.09 26.09 0 0 1 6.12-3.06zm14.24 0a45.65 45.65 0 0 0-6.17 3.39v15h-5.83V22.79h4.92l.38 2.58a26.09 26.09 0 0 1 6.12-3.06zm19.51 18.4h-4.78l-.43-1.58a12.73 12.73 0 0 1-6.93 2.06c-4.25 0-6.07-2.92-6.07-6.93 0-4.73 2.06-6.55 6.79-6.55h5.59v-2.44c0-2.58-.72-3.49-4.45-3.49a32.53 32.53 0 0 0-6.45.72l-.72-4.45a30.38 30.38 0 0 1 8-1.1c7.31 0 9.47 2.58 9.47 8.41zm-5.83-8.8h-4.3c-1.91 0-2.44.53-2.44 2.29s.53 2.34 2.34 2.34a9.18 9.18 0 0 0 4.4-1.2zm23.75-19.79a17.11 17.11 0 0 0-3.35-.38c-2.29 0-2.63 1-2.63 2.77v2.92h5.93l-.33 4.64h-5.59v18.64h-5.83V27.43h-3.73v-4.64h3.73v-3.25c0-4.83 2.25-7.22 7.41-7.22a18.47 18.47 0 0 1 5 .67zm11.38 29.07c-8 0-10.13-4.4-10.13-9.18v-5.88c0-4.78 2.15-9.18 10.13-9.18s10.13 4.4 10.13 9.18v5.88c.01 4.78-2.15 9.18-10.13 9.18zm0-19.27c-3.11 0-4.3 1.39-4.3 4v6.26c0 2.63 1.2 4 4.3 4s4.3-1.39 4.3-4V31.3c0-2.63-1.19-4.02-4.3-4.02zm25.14.39a45.65 45.65 0 0 0-6.17 3.39v15h-5.83V22.79h4.92l.38 2.58a26.08 26.08 0 0 1 6.12-3.06zm16.02 18.4V29.82c0-1.24-.53-1.86-1.86-1.86a16.08 16.08 0 0 0-6.07 2v16.11h-5.83V22.79h4.45l.57 2a23.32 23.32 0 0 1 9.34-2.48 4.42 4.42 0 0 1 4.4 2.49 22.83 22.83 0 0 1 9.37-2.49c3.87 0 5.26 2.72 5.26 6.88v16.88h-5.83V29.82c0-1.24-.53-1.86-1.86-1.86a15.43 15.43 0 0 0-6.07 2v16.11z"/>
  <path class="rect-dark" fill="#4040B2" d="M36.4 20.2v18.93l16.4-9.46V10.72L36.4 20.2z"/>
  <path class="rect-light" fill="#5C4EE5" d="M18.2 10.72l16.4 9.48v18.93l-16.4-9.47V10.72z"/>
  <path class="rect-light" fill="#5C4EE5" d="M0 .15v18.94l16.4 9.47V9.62L0 .15zm18.2 50.53l16.4 9.47V41.21l-16.4-9.47v18.94z"/>
</svg>

## Contents

* [Requirements](#requirements)
* [Installation](#installation)
* [Resources](#resources)
  * [`pulsar_tenant`](#pulsar_tenant)
  * [`pulsar_cluster`](#pulsar_cluster)
  * [`pulsar_namespace`](#pulsar_namespace)
  * [`pulsar_topic`](#pulsar_topic)

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 0.12.0 or later
- [Go](https://golang.org/doc/install) 1.16 or later (to build the provider plugin)

Installation
------------

- Manual build

  * Clone this repository and cd into the directory
  * Run `make build`, it will out a file named `terraform-provider-pulsar`
  * Copy this `terraform-provider-pulsar` bin file to your terraform plugin directory:

      Operating system  | User plugins directory
      ------------------|-----------------------
      Windows(amd64)    | %APPDATA%\terraform.d\plugins\registry.terraform.io\streamnative\pulsar\0.1.0\windows_amd64\
      Linux(amd64)      | ~/.terraform.d/plugins/registry.terraform.io/streamnative/pulsar\0.1.0\linux_amd64\
      MacOS(amd64)      | ~/.terraform.d/plugins/registry.terraform.io/streamnative/pulsar\0.1.0\darwin_amd64\

- From Terraform registry

  This plugin has been released to [Terraform registry](https://registry.terraform.io/providers/streamnative/pulsar/latest):
  
  ```terraform
  terraform {
    required_providers {
      pulsar = {
        version = "0.1.0"
        source = "registry.terraform.io/streamnative/pulsar"
      }
    }
  }
  ```


Testing the Apache Pulsar Terraform Provider
------------

* Change directory to the project </path/to/provider/terraform-provider-pulsar>
* In order to test the provider, you can run `make test`
* In order to run the full suite of Acceptance tests, run `make testacc`

`Note: Acceptance tests create real resources, and often cost money to run.`


Provider Configuration
------------

### Example

Example provider with apache pulsar cluster, running locally with authentication disabled.

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
  token = "my_auth_token"
}
```

| Property                      | Description                                                                                                           | Required    |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------- | ---------- |
| `web_service_url`             | URL of your Apache Pulsar Cluster                             | Yes |
| `token`           | Authentication Token for your Apache Pulsar Cluster, which is required only if your cluster has authentication enabled| No       |
| `tls_trust_certs_file_path` | Path to a custom trusted TLS certificate file | No       |
| `tls_allow_insecure_connection` | Boolean flag to accept untrusted TLS certificates | No       |
| `api_version`| Used to request Apache Pulsar API service, default by 0, which represents use default version | No |      
Resources
------------

### `pulsar_tenant`

A resource for managing Pulsar Tenants, can update admin roles and allowed clusters for a tenant.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_tenant" "my_tenant" {
  tenant           = "thanos"
  allowed_clusters = ["pulsar-cluster-1"]
  admin_roles      = ["godmode"]
}
```

#### Properties

| Property                      | Description                                         | Required                   |
| ----------------------------- | --------------------------------------------------- |----------------------------|
| `tenant`                      | Name of the Tenant that you want to create          | Yes
| `allowed_clusters`            | An Array of clusters, accessible by this tenant     | No
| `admin_roles`                 | Admin Roles to be assumed by this Tenant            | No

### `pulsar_cluster`

A resource for managing Apache Pulsar Clusters, can update various properties for a given cluster.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_cluster" "my_cluster" {
  cluster = "eternals"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["skrulls", "krees"]
  }
}
```

#### Properties

| Property                      | Description                                                       | Required                   |
| ----------------------------- | ----------------------------------------------------------------- |----------------------------|
| `cluster`                     | Name of the Cluster that you want to create                       | Yes
| `cluster_data`                | A Map of required fields for the cluster                          | Yes
| `web_service_url`             | Required in cluster data, pointing to your broker web service     | Yes
| `web_service_url_tls`         | Pointing to your broker web service via tls                       | No
| `broker_service_url`          | Required in cluster data for broker discovery                     | Yes
| `broker_service_url_tls`      | Required in cluster data for broker discovery via tls             | No
| `peer_clusters`               | Required in cluster data for adding peer clusters                 | Yes

### `pulsar_namespace`

A resource for creating and managing Apache Pulsar Namespaces, can update various properties for a given namespace.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "skrulls"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }
}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "thanos"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "eternals"

  enable_deduplication = true

  // If defined partially, plan would show difference
  // however, none of the mising optionals would be changed
  namespace_config {
    anti_affinity                  = "anti-aff"
    max_consumers_per_subscription = "50"
    max_consumers_per_topic        = "50"
    max_producers_per_topic        = "50"
    replication_clusters           = ["standalone"]
  }

  dispatch_rate {
    dispatch_msg_throttling_rate  = 50
    rate_period_seconds           = 50
    dispatch_byte_throttling_rate = 2048
  }

  retention_policies {
    retention_minutes    = "1600"
    retention_size_in_mb = "10000"
  }

  backlog_quota {
    limit_bytes  = "10000000000"
    limit_seconds = "-1"
    policy = "consumer_backlog_eviction"
    type = "destination_storage"
  }

  persistence_policies {
    bookkeeper_ensemble                   = 1   // Number of bookies to use for a topic, default: 0
    bookkeeper_write_quorum               = 1   // How many writes to make of each entry, default: 0
    bookkeeper_ack_quorum                 = 1   // Number of acks (guaranteed copies) to wait for each entry, default: 0
    managed_ledger_max_mark_delete_rate   = 0.0 // Throttling rate of mark-delete operation (0 means no throttle), default: 0.0
  }

  permission_grant {
    role    = "some-role"
    actions = ["produce", "consume", "functions"]
  }
}
```

#### Properties

| Property                        | Description                                                       | Required                   |
| ------------------------------- | ----------------------------------------------------------------- |----------------------------|
| `tenant`                        | Name of the Tenant managing this namespace                        | Yes
| `namespace`                     | name of the namespace                                             | Yes
| `enable_deduplication`          | Message deduplication state on a namespace                        | No |
| `namespace_config`              | Configuration for your namespaces like max allowed producers to produce messages | No |
| `dispatch_rate`                 | Apache Pulsar throttling config                                   | No |
| `retention_policies`            | Data retention policies                                           | No |
| `schema_validation_enforce`     | Enable or disable schema validation                               | No |
| `schema_compatibility_strategy` | Set schema compatibility strategy                                 | No |
| `backlog_quota`                 | [Backlog Quota](https://pulsar.apache.org/docs/en/admin-api-namespaces/#set-backlog-quota-policies) for all topics | No |
| `persistence_policies`          | [Persistence policies](https://pulsar.apache.org/docs/en/admin-api-namespaces/#set-persistence-policies) for all topics under a given namespace       | No |
| `permission_grant`              | [Permission grants](https://pulsar.apache.org/docs/en/admin-api-permissions/) on a namespace. This block can be repeated for each grant you'd like to add | No |

##### `schema_compatibility_strategy`

* AutoUpdateDisabled
* Backward
* Forward
* Full
* AlwaysCompatible
* BackwardTransitive
* ForwardTransitive
* FullTransitive

### `pulsar_topic`

A resource for creating and managing Apache Pulsar Topics, can update partitions for a given partition topic.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_topic" "sample-topic-1" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "partition-topic"
  partitions = 4                     # partitions > 0 means this is a partition topic

  permission_grant {
    role    = "some-role"
    actions = ["produce", "consume", "functions"]
  }

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }
}

resource "pulsar_topic" "sample-topic-2" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "non-partition-topic"
  partitions = 0                     # partitions = 0 means this is a non-partition topic

  permission_grant {
    role    = "some-role"
    actions = ["produce", "consume", "functions"]
  }

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }
}
```

#### Properties

| Property                      | Description                                                       | Required                   |
| ----------------------------- | ----------------------------------------------------------------- |----------------------------|
| `tenant`                      | Name of the Tenant managing this topic                            | Yes
| `namespace`                   | Name of the Namespace for this topic                              | Yes
| `topic_type`                  | Topic persistence (`persistent`, `non-persistent`)                | Yes
| `topic_name`                  | Name of the topic                                                 | Yes
| `partitions`                  | Number of [partitions](https://pulsar.apache.org/docs/en/concepts-messaging/#partitioned-topics) (`0` for non-partitioned topic, `> 1` for partitioned topic) | Yes
| `permission_grant`            | [Permission grants](https://pulsar.apache.org/docs/en/admin-api-permissions/) on a topic. This block can be repeated for each grant you'd like to add. Permission grants are also inherited from the topic's namespace. | No |
| `retention_policies`          | Data retention policies                                           | No |

### `pulsar_source`

A resource for creating and managing Apache Pulsar Sources.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
  api_version = "3"
}

resource "pulsar_source" "source-1" {
  provider = pulsar

  name = "source-1"
  tenant = "public"
  namespace = "default"

  archive = "testdata/pulsar-io/pulsar-io-file-2.8.1.nar"

  destination_topic_name = "source-1-topic"

  processing_guarantees = "EFFECTIVELY_ONCE"

  configs = "{\"inputDirectory\":\"opt\"}"

  cpu = 2
  disk_mb = 20480
  ram_mb = 2048
}
```

#### Properties

| Property | Description | Required|  
| ----------------------------- | ----------------------------------------------------------------- |----------------------------|
| `name` | The source's name | True
| `tenant` | The source's tenant | True  
| `namespace` | The source's namespace | True
| `destination_topic_name` | The Pulsar topic to which data is sent | True
| `archive` | The path to the NAR archive for the Source. It also supports url-path [http/https/file (file protocol assumes that file already exists on worker host)] from which worker can download the package | True  
| `classname` | The source's class name if archive is file-url-path (file://) | False
| `configs` | User defined configs key/values (JSON string) | False
| `deserialization_classname` | The SerDe classname for the source | False  
| `processing_guarantees` | Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE) | False  
| `parallelism` | The source's parallelism factor | False  
| `cpu` | The CPU that needs to be allocated per source instance (applicable only to Docker runtime) | False
| `ram_mb` | The RAM that need to be allocated per source instance (applicable only to the process and Docker runtimes) | False
| `disk_mb` | The disk that need to be allocated per source instance (applicable only to Docker runtime) | False
| `runtime_flags` | User defined configs key/values (JSON string) | False

### `pulsar_sink`

A resource for creating and managing Apache Pulsar Sinks.


#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
  api_version = "3"
}

resource "pulsar_sink" "sample-sink-1" {
  provider = pulsar

  name = "sample-sink-1"
  tenant = "public"
  namespace = "default"
  inputs = ["sink-1-topic"]
  subscription_position = "Latest"
  cleanup_subscription = false
  parallelism = 1
  auto_ack = true

  processing_guarantees = "EFFECTIVELY_ONCE"

  archive = "testdata/pulsar-io/pulsar-io-jdbc-postgres-2.8.1.nar"
  configs = "{\"jdbcUrl\":\"jdbc:clickhouse://localhost:8123/pulsar_clickhouse_jdbc_sink\",\"password\":\"password\",\"tableName\":\"pulsar_clickhouse_jdbc_sink\",\"userName\":\"clickhouse\"}"
}
```


#### Properties

| Property                      | Description                                                       | Required                   |
| ----------------------------- | ----------------------------------------------------------------- |----------------------------|
| `tenant` | The sink's tenant | True
| `namespace` | The sink's namespace | True
| `name` | The sink's name | True
| `inputs` | The sink's input topics | False
| `topics_pattern` | TopicsPattern to consume from list of topics under a namespace that match the pattern | False
| `input_specs` | The map of input topics specs | False
| `configs` | User defined configs key/values (JSON string) | False
| `archive` | Path to the archive file for the sink. It also supports url-path [http/https/file (file protocol assumes that file already exists on worker host)] from which worker can download the package | True
| `subscription_name` | Pulsar source subscription name if user wants a specific subscription-name for input-topic consumer | False
| `subscription_position` | Pulsar source subscription position if user wants to consume messages from the specified location (Latest, Earliest) | False
| `cleanup_subscription` | Whether the subscriptions the functions created/used should be deleted when the functions was deleted | True
| `processing_guarantees` | Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE) | False
| `retain_ordering` | Sink consumes and sinks messages in order | False
| `auto_ack` | Whether or not the framework will automatically acknowledge messages | True
| `timeout_ms` | The message timeout in milliseconds | False
| `parallelism` | The sink's parallelism factor | False
| `cpu` | The CPU that needs to be allocated per sink instance (applicable only to Docker runtime) | False
| `ram_mb` | The RAM that need to be allocated per sink instance (applicable only to the process and Docker runtimes) | False
| `disk_mb` | The disk that need to be allocated per sink instance (applicable only to Docker runtime) | False
| `custom_schema_inputs` | The map of input topics to Schema types or class names (as a JSON string) | False
| `custom_serde_inputs` | The map of input topics to SerDe class names (as a JSON string) | False
| `custom_runtime_options` | A string that encodes options to customize the runtime | False


Importing existing resources
------------

All resources could be imported using the [standard terraform way](https://www.terraform.io/docs/import/usage.html).

#### Example

```
terraform import pulsar_cluster.standalone standalone
```

Contributing
---------------------------

Terraform is the work of thousands of contributors. We appreciate your help!

To contribute, please read the contribution guidelines: [Contributing to Terraform - Apache Pulsar Provider](.github/CONTRIBUTING.md)

Issues on GitHub are intended to be related to bugs or feature requests with provider codebase.
See https://www.terraform.io/docs/extend/community/index.html for a list of community resources to ask questions about Terraform.



[1]: https://www.terraform.io
[2]: https://pulsar.apache.org
[third-party-plugins]: https://www.terraform.io/docs/configuration/providers.html#third-party-plugins
[install-go]: https://golang.org/doc/install#install
