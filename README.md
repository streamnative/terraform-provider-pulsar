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

# Terraform Provider for Pulsar

Authored by [StreamNative], this repository includes the source code and documentation for
a [Terraform](https://www.terraform.io) provider for managing [Apache Pulsar](https://pulsar.apache.org) entities such as clusters, tenants, namespaces, topics, sources, and sinks.

# Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) 0.12.0 or later
- [Go](https://golang.org/doc/install) 1.21 or later (to build the provider plugin)

# Installation

- From terraform registry

  This provider has been published in the [Terraform Registry](https://registry.terraform.io/providers/streamnative/pulsar/latest):

  ```hcl
  terraform {
    required_providers {
      pulsar = {
        version = "0.1.3"
        source = "registry.terraform.io/streamnative/pulsar"
      }
    }
  }
  ```

- From source code

  - Clone this repository and cd into the directory
  - Run `make build`, it will generate a binary file named `terraform-provider-pulsar`
  - Copy this `terraform-provider-pulsar` binary file to your terraform plugin directory based on your OS:

| Operating System | User plugins directory                                                                        |
| ---------------- | --------------------------------------------------------------------------------------------- |
| Windows(amd64)   | %APPDATA%\terraform.d\plugins\registry.terraform.io\streamnative\pulsar\0.1.0\windows_amd64\  |
| Linux(amd64)     | ~/.terraform.d/plugins/registry.terraform.io/streamnative/pulsar/0.1.0/linux_amd64/           |
| MacOS(amd64)     | ~/.terraform.d/plugins/registry.terraform.io/streamnative/pulsar/0.1.0/darwin_amd64/          |

- Run `make build-dev`, it will build the binary and copy it to the plugin directory automatically.

# Version Compatibility

The compatibility matrix from terraform-provider-pulsar to pulsar as shown below:

|        | Min Version of Pulsar |
|--------|-----------------------|
| v0.2.x | pulsar 2.10.4         |
| v0.1.x | pulsar 2.8.x          |

> Note: It's not strictly tested. Please report issues when you encounter any bugs.

# Using the Provider

## Configurations

### Example

Example provider with apache pulsar cluster, running locally with authentication disabled.

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
  token           = "my_auth_token"
}
```

```hcl
provider "pulsar" {
  web_service_url           = "https://localhost:8443"
  tls_trust_certs_file_path = "./ca.pem"
  tls_key_file_path         = "./key.pem"
  tls_cert_file_path        = "./cert.pem"
}
```

### Properties

| Property                        | Description                                                                                                            | Required |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | -------- |
| `web_service_url`               | URL of your Apache Pulsar Cluster                                                                                      | Yes      |
| `token`                         | Authentication Token for your Apache Pulsar Cluster, which is required only if your cluster has authentication enabled | No       |
| `tls_trust_certs_file_path`     | Path to a custom trusted TLS certificate file                                                                          | No       |
| `tls_key_file_path`             | Path to the key to use when using TLS client authentication                                                            | No       |
| `tls_cert_file_path`            | Path to the cert to use when using TLS client authentication                                                           | No       |
| `tls_allow_insecure_connection` | Boolean flag to accept untrusted TLS certificates                                                                      | No       |
| `api_version`                   | Used to request Apache Pulsar API service, default by 0, which represents use default version                          | No       |
| `audience`                      | The OAuth 2.0 resource server identifier for the Pulsar cluster                                                        | No       |
| `client_id`                     | The OAuth 2.0 client identifier                                                                                        | No       |
| `issuer_url`                    | The OAuth 2.0 URL of the authentication provider which allows the Pulsar client to obtain an access token              | No       |
| `scope`                         | The OAuth 2.0 scope to request                                                                                         | No       |
| `key_file_path`                 | The path of the private key file                                                                                       | No       |

## Resources

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
    broker_service_url = "pulsar://localhost:6050"
    peer_clusters      = ["skrulls", "krees"]
  }
}
```

#### Properties

| Property                 | Description                                                   | Required |
| ------------------------ | ------------------------------------------------------------- | -------- |
| `cluster`                | Name of the Cluster that you want to create                   | Yes      |
| `cluster_data`           | A Map of required fields for the cluster                      | Yes      |
| `web_service_url`        | Required in cluster data, pointing to your broker web service | Yes      |
| `web_service_url_tls`    | Pointing to your broker web service via tls                   | No       |
| `broker_service_url`     | Required in cluster data for broker discovery                 | Yes      |
| `broker_service_url_tls` | Required in cluster data for broker discovery via tls         | No       |
| `peer_clusters`          | Required in cluster data for adding peer clusters             | Yes      |

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

| Property           | Description                                     | Required |
| ------------------ | ----------------------------------------------- | -------- |
| `tenant`           | Name of the Tenant that you want to create      | Yes      |
| `allowed_clusters` | An Array of clusters, accessible by this tenant | No       |
| `admin_roles`      | Admin Roles to be assumed by this Tenant        | No       |

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
    broker_service_url = "pulsar://localhost:6050"
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
    message_ttl_seconds            = "86400"
    replication_clusters           = ["standalone"]
  }

  dispatch_rate {
    dispatch_msg_throttling_rate  = 50
    rate_period_seconds           = 50
    dispatch_byte_throttling_rate = 2048
  }

  subscription_dispatch_rate {
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

| Property                     | Description                                                                                                                                               | Required |
| ---------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| `tenant`                     | Name of the Tenant managing this namespace                                                                                                                | Yes      |
| `namespace`                  | name of the namespace                                                                                                                                     | Yes      |
| `enable_deduplication`       | Message deduplication state on a namespace                                                                                                                | No       |
| `namespace_config`           | Configuration for your namespaces like max allowed producers to produce messages                                                                          | No       |
| `dispatch_rate`              | [Apache Pulsar throttling config for topics](https://pulsar.apache.org/docs/en/admin-api-namespaces/#set-dispatch-throttling-for-topics)                  | No       |
| `subscription_dispatch_rate` | [Apache Pulsar throttling config for subscriptions](https://pulsar.apache.org/docs/en/admin-api-namespaces/#set-dispatch-throttling-for-subscription)     | No       |
| `retention_policies`         | Data retention policies                                                                                                                                   | No       |
| `backlog_quota`              | [Backlog Quota](https://pulsar.apache.org/docs/en/admin-api-namespaces/#set-backlog-quota-policies) for all topics                                        | No       |
| `persistence_policies`       | [Persistence policies](https://pulsar.apache.org/docs/en/admin-api-namespaces/#set-persistence-policies) for all topics under a given namespace           | No       |
| `permission_grant`           | [Permission grants](https://pulsar.apache.org/docs/en/admin-api-permissions/) on a namespace. This block can be repeated for each grant you'd like to add | No       |

namespace_config nested schema

| Property                               | Description                                      | Required |
|----------------------------------------|--------------------------------------------------|----------|
| `anti_affinity`                        | Anti-affinity group name                         | No       |
| `is_allow_auto_update_schema`          | Is schema auto-update allowed                    | No       |
| `max_consumers_per_subscription`       | Sets the max consumers per subscription          | No       |
| `max_consumers_per_topic`              | Sets the max consumers per topic                 | No       |
| `max_producers_per_topic`              | Sets the max producers per topic                 | No       |
| `message_ttl_seconds`                  | Sets the message TTL in seconds                  | No       |
| `offload_threshold_size_in_mb`         | Set topic offload threshold size in MB           | No       |
| `replication_clusters`                 | List of replication clusters for the namespace   | No       |
| `schema_compatibility_strategy`        | Set schema compatibility strategy                | No       |
| `schema_validation_enforce`            | Enable or disable schema validation              | No       |
| `subscription_expiration_time_minutes` | Sets the subscription expiration time in minutes | No       |

The `schema_compatibility_strategy` can take the following values:

- AutoUpdateDisabled
- Backward
- Forward
- Full
- AlwaysCompatible
- BackwardTransitive
- ForwardTransitive
- FullTransitive

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

  enable_deduplication = true

  dispatch_rate {
    msg_dispatch_rate = 0
    byte_dispatch_rate = 0
    dispatch_rate_period = 0
    relative_to_publish_rate = true
  }

  backlog_quota = {
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

  topic_config {
    delayed_delivery {
      enabled = true
      time = 1500
    }
    max_consumers = 10
    max_producers = 5
    message_ttl_seconds = 3600
    max_unacked_messages_per_consumer = 1000
    max_unacked_messages_per_subscription = 2000
    msg_publish_rate = 100
    byte_publish_rate = 1024
    inactive_topic {
      enable_delete_while_inactive = true
      max_inactive_duration = "60s"
      delete_mode = "delete_when_no_subscriptions"
    }
    compaction_threshold = 1073741824
  }


}
```

#### Properties

| Property             | Description                                                                                                                                                                                                             | Required |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| `tenant`             | Name of the Tenant managing this topic                                                                                                                                                                                  | Yes      |
| `namespace`          | Name of the Namespace for this topic                                                                                                                                                                                    | Yes      |
| `topic_type`         | Topic persistence (`persistent`, `non-persistent`)                                                                                                                                                                      | Yes      |
| `topic_name`         | Name of the topic                                                                                                                                                                                                       | Yes      |
| `partitions`         | Number of [partitions](https://pulsar.apache.org/docs/en/concepts-messaging/#partitioned-topics) (`0` for non-partitioned topic, `> 1` for partitioned topic)                                                           | Yes      |
| `permission_grant`   | [Permission grants](https://pulsar.apache.org/docs/en/admin-api-permissions/) on a topic. This block can be repeated for each grant you'd like to add. Permission grants are also inherited from the topic's namespace. | No       |
| `retention_policies` | Data retention policies                                                                                                                                                                                                 | No       |
| `enable_deduplication` | Enable message deduplication for the topic                                                                                                                                                                            | No       |
| `dispatch_rate` | Configure message dispatch rate for the topic                                                                                                                                                                              | No       |
| `backlog_quota` | Configure [Backlog Quota](https://pulsar.apache.org/api/admin/4.0.x/org/apache/pulsar/common/policies/data/BacklogQuota.html) for the topic                                                                                                | No       |
| `persistence_policies` | Set [Persistence Policies](https://pulsar.apache.org/api/admin/4.0.x/org/apache/pulsar/common/policies/data/PersistencePolicies.html) for the topic                                                                                            | No       |
| `topic_config` | Additional configuration for the topic                                                                                                                                                                                      | No       |

##### Nested Schema for `dispatch_rate`

| Property                 | Description                                                                               | Required |
| ------------------------ | ----------------------------------------------------------------------------------------- | -------- |
| `msg_dispatch_rate`      | Maximum number of messages per second                                                     | Yes      |
| `byte_dispatch_rate`     | Maximum number of bytes per second                                                        | Yes      |
| `dispatch_rate_period`   | The time period in seconds                                                                | Yes      |
| `relative_to_publish_rate` | Whether dispatch rate should be relative to publish rate                                | Yes      |

##### Nested Schema for `backlog_quota`

| Property                 | Description                                                                               | Required |
| ------------------------ | ----------------------------------------------------------------------------------------- | -------- |
| `limit_bytes`            | Maximum size of backlog in bytes                                                          | Yes      |
| `limit_seconds`          | Maximum time of backlog (in seconds). -1 means no time limit                              | Yes      |
| `policy`                 | Action to take when quota is exceeded (e.g., `consumer_backlog_eviction`)                 | Yes      |
| `type`                   | Backlog quota type (e.g., `destination_storage`)                                          | Yes      |

##### Nested Schema for `persistence_policies`

| Property                           | Description                                                                      | Required |
| ---------------------------------- | -------------------------------------------------------------------------------- | -------- |
| `bookkeeper_ensemble`              | Number of bookies to use for a topic                                             | Yes      |
| `bookkeeper_write_quorum`          | How many writes to make of each entry                                            | Yes      |
| `bookkeeper_ack_quorum`            | Number of acks (guaranteed copies) to wait for each entry                        | Yes      |
| `managed_ledger_max_mark_delete_rate` | Throttling rate of mark-delete operation (0 means no throttle)                | Yes      |

##### Nested Schema for `topic_config`

| Property                            | Description                                                                      | Optional |
| ----------------------------------- | -------------------------------------------------------------------------------- | -------- |
| `max_consumers`                     | Maximum number of consumers for the topic                                        | Yes      |
| `max_producers`                     | Maximum number of producers for the topic                                        | Yes      |
| `message_ttl_seconds`               | Time-to-live (in seconds) for messages                                           | Yes      |
| `max_unacked_messages_per_consumer` | Maximum number of unacknowledged messages per consumer                           | Yes      |
| `max_unacked_messages_per_subscription` | Maximum number of unacknowledged messages per subscription                   | Yes      |
| `msg_publish_rate`                  | Maximum message publish rate (per second)                                        | Yes      |
| `byte_publish_rate`                 | Maximum byte publish rate (per second)                                           | Yes      |
| `compaction_threshold`              | The threshold (in bytes) to trigger compaction                                   | Yes      |
| `delayed_delivery`                  | Configuration for delayed message delivery                                       | Yes      |
| `inactive_topic`                    | Configuration for inactive topic handling                                        | Yes      |

##### Nested Schema for `delayed_delivery`

| Property                 | Description                                                                               | Required |
| ------------------------ | ----------------------------------------------------------------------------------------- | -------- |
| `enabled`                | Whether to enable delayed delivery for messages                                           | Yes      |
| `time`                   | The tick time for delayed delivery messages                                               | Yes      |

##### Nested Schema for `inactive_topic`

| Property                      | Description                                                                           | Required |
| ----------------------------- | ------------------------------------------------------------------------------------- | -------- |
| `enable_delete_while_inactive` | Whether to delete the inactive topic                                                  | Yes      |
| `max_inactive_duration`       | Max duration of inactivity (e.g., "60s") after which topic will be deleted            | Yes      |
| `delete_mode`                 | Mode of deletion (`delete_when_no_subscriptions` or `delete_when_subscriptions_caught_up`) | Yes      |

### `pulsar_function`

A resource for creating and managing Apache Pulsar Functions.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_function" "function-1" {
  provider = pulsar

  name = "function1"
  tenant = "public"
  namespace = "default"
  parallelism = 1

  processing_guarantees = "ATLEAST_ONCE"

  jar = "/Downloads/apache-pulsar-2.10.1/examples/api-examples.jar"
  classname = "org.apache.pulsar.functions.api.examples.WordCountFunction"

  inputs = ["public/default/input1", "public/default/input2"]

  output = "public/default/test-out"

  subscription_name = "test-sub"
  subscription_position = "Latest"
  cleanup_subscription = true
  skip_to_latest = true
  forward_source_message_property = true
  retain_key_ordering = true
  auto_ack = true
  max_message_retries = 101
  dead_letter_topic = "public/default/dlt"
  log_topic = "public/default/lt"
  timeout_ms = 6666

  secrets = jsonencode(
  {
    "SECRET1": {
       "path": "sectest"
       "key": "hello"
    }
  })
  custom_runtime_options = jsonencode(
  {
      "env": {
          "MILLBRAE": "HILLCREST"
      }
  })
}
```

#### Properties

| Property                          | Description                                                                                                                                             | Required |
|-----------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| `name`                            | The name of the function.                                                                                                                               | True     |
| `tenant`                          | The tenant of the function.                                                                                                                             | True     |
| `namespace`                       | The namespace of the function.                                                                                                                          | True     |
| `jar`                             | The path to the jar file.                                                                                                                               | False    |
| `py`                              | The path to the python file.                                                                                                                            | False    |
| `go`                              | The path to the go file.                                                                                                                                | False    |
| `classname`                       | The class name of the function.                                                                                                                         | False    |
| `inputs`                          | The input topics of the function.                                                                                                                       | False    |
| `topics_pattern`                  | The input topics pattern of the function. The pattern is a regex expression. The function consumes from all topics matching the pattern.                | False    |
| `output`                          | The output topic of the function.                                                                                                                       | False    |
| `parallelism`                     | The parallelism of the function.                                                                                                                        | False    |
| `processing_guarantees`           | The processing guarantees (aka delivery semantics) applied to the function. Possible values are `ATMOST_ONCE`, `ATLEAST_ONCE`, and `EFFECTIVELY_ONCE`.  | False    |
| `subscription_name`               | The subscription name of the function.                                                                                                                  | False    |
| `subscription_position`           | The subscription position of the function. Possible values are `LATEST`, `EARLIEST`, and `CUSTOM`.                                                      | False    |
| `cleanup_subscription`            | Whether to clean up subscription when the function is deleted.                                                                                          | False    |
| `skip_to_latest`                  | Whether to skip to the latest position when the function is restarted after failure.                                                                    | False    |
| `forward_source_message_property` | Whether to forward source message property to the function output message.                                                                              | False    |
| `retain_ordering`                 | Whether to retain ordering when the function is restarted after failure.                                                                                | False    |
| `retain_key_ordering`             | Whether to retain key ordering when the function is restarted after failure.                                                                            | False    |
| `auto_ack`                        | User defined configs key/values (JSON string)                                                                                                           | False    |
| `max_message_retries`             | The maximum number of times that a message will be retried when the function is configured with `EFFECTIVELY_ONCE` processing guarantees.               | False    |
| `dead_letter_topic`               | The dead letter topic of the function.                                                                                                                  | False    |
| `log_topic`                       | The log topic of the function.                                                                                                                          | False    |
| `timeout_ms`                      | The timeout of the function in milliseconds.                                                                                                            | False    |
| `input_type_classname`            | The input type class name of the function.                                                                                                              | False    |
| `output_type_classname`           | The output type class name of the function.                                                                                                             | False    |
| `output_serde_classname`          | The output serde class name of the function.                                                                                                            | False    |
| `output_schema_type`              | The output schema type of the function.                                                                                                                 | False    |
| `custom_serde_inputs`             | The custom serde inputs of the function.                                                                                                                | False    |
| `custom_schema_inputs`            | The custom schema inputs of the function.                                                                                                               | False    |
| `custom_schema_outputs`           | The custom schema outputs of the function.                                                                                                              | False    |
| `custom_runtime_options`          | The custom runtime options of the function.                                                                                                             | False    |
| `secrets`                         | The secrets of the function.                                                                                                                            | False    |
| `cpu`                             | The CPU that needs to be allocated per function instance                                                                                                | False    |
| `ram_mb`                          | The RAM that need to be allocated per function instance                                                                                                 | False    |
| `disk_mb`                         | The disk that need to be allocated per function instance                                                                                                | False    |


### `pulsar_source`

A resource for creating and managing Apache Pulsar Sources.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_source" "source-1" {
  provider = pulsar

  name = "source-1"
  tenant = "public"
  namespace = "default"

  archive = "testdata/pulsar-io/pulsar-io-file-2.10.4.nar"

  destination_topic_name = "source-1-topic"

  processing_guarantees = "EFFECTIVELY_ONCE"

  configs = "{\"inputDirectory\":\"opt\"}"

  cpu = 2
  disk_mb = 20480
  ram_mb = 2048
}
```

#### Properties

| Property                    | Description                                                                                                                                                                                        | Required |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| `name`                      | The source's name                                                                                                                                                                                  | True     |
| `tenant`                    | The source's tenant                                                                                                                                                                                | True     |
| `namespace`                 | The source's namespace                                                                                                                                                                             | True     |
| `destination_topic_name`    | The Pulsar topic to which data is sent                                                                                                                                                             | True     |
| `archive`                   | The path to the NAR archive for the Source. It also supports url-path [http/https/file (file protocol assumes that file already exists on worker host)] from which worker can download the package | True     |
| `classname`                 | The source's class name if archive is file-url-path (file://)                                                                                                                                      | False    |
| `configs`                   | User defined configs key/values (JSON string)                                                                                                                                                      | False    |
| `deserialization_classname` | The SerDe classname for the source                                                                                                                                                                 | False    |
| `processing_guarantees`     | Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE)                                                                                       | False    |
| `parallelism`               | The source's parallelism factor                                                                                                                                                                    | False    |
| `cpu`                       | The CPU that needs to be allocated per source instance (applicable only to Docker runtime)                                                                                                         | False    |
| `ram_mb`                    | The RAM that need to be allocated per source instance (applicable only to the process and Docker runtimes)                                                                                         | False    |
| `disk_mb`                   | The disk that need to be allocated per source instance (applicable only to Docker runtime)                                                                                                         | False    |
| `runtime_flags`             | User defined configs key/values (JSON string)                                                                                                                                                      | False    |

### `pulsar_sink`

A resource for creating and managing Apache Pulsar Sinks.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
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

  archive = "testdata/pulsar-io/pulsar-io-jdbc-postgres-2.10.4.nar"
  configs = "{\"jdbcUrl\":\"jdbc:clickhouse://localhost:8123/pulsar_clickhouse_jdbc_sink\",\"password\":\"password\",\"tableName\":\"pulsar_clickhouse_jdbc_sink\",\"userName\":\"clickhouse\"}"
}
```

#### Properties

| Property                 | Description                                                                                                                                                                                   | Required |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| `tenant`                 | The sink's tenant                                                                                                                                                                             | True     |
| `namespace`              | The sink's namespace                                                                                                                                                                          | True     |
| `name`                   | The sink's name                                                                                                                                                                               | True     |
| `inputs`                 | The sink's input topics                                                                                                                                                                       | False    |
| `topics_pattern`         | TopicsPattern to consume from list of topics under a namespace that match the pattern                                                                                                         | False    |
| `input_specs`            | The map of input topics specs                                                                                                                                                                 | False    |
| `configs`                | User defined configs key/values (JSON string)                                                                                                                                                 | False    |
| `archive`                | Path to the archive file for the sink. It also supports url-path [http/https/file (file protocol assumes that file already exists on worker host)] from which worker can download the package | True     |
| `subscription_name`      | Pulsar source subscription name if user wants a specific subscription-name for input-topic consumer                                                                                           | False    |
| `subscription_position`  | Pulsar source subscription position if user wants to consume messages from the specified location (Latest, Earliest)                                                                          | False    |
| `cleanup_subscription`   | Whether the subscriptions the functions created/used should be deleted when the functions was deleted                                                                                         | True     |
| `processing_guarantees`  | Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE)                                                                                  | False    |
| `retain_ordering`        | Sink consumes and sinks messages in order                                                                                                                                                     | False    |
| `auto_ack`               | Whether or not the framework will automatically acknowledge messages                                                                                                                          | True     |
| `timeout_ms`             | The message timeout in milliseconds                                                                                                                                                           | False    |
| `parallelism`            | The sink's parallelism factor                                                                                                                                                                 | False    |
| `cpu`                    | The CPU that needs to be allocated per sink instance (applicable only to Docker runtime)                                                                                                      | False    |
| `ram_mb`                 | The RAM that need to be allocated per sink instance (applicable only to the process and Docker runtimes)                                                                                      | False    |
| `disk_mb`                | The disk that need to be allocated per sink instance (applicable only to Docker runtime)                                                                                                      | False    |
| `custom_schema_inputs`   | The map of input topics to Schema types or class names (as a JSON string)                                                                                                                     | False    |
| `custom_serde_inputs`    | The map of input topics to SerDe class names (as a JSON string)                                                                                                                               | False    |
| `custom_runtime_options` | A string that encodes options to customize the runtime                                                                                                                                        | False    |

### `pulsar_subscription`

A resource for creating and managing Apache Pulsar Subscriptions.

#### Example

```hcl
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_subscription" "example" {
  topic_name        = "persistent://public/default/example-topic"
  subscription_name = "example-subscription"
  position          = "latest"
}
```

#### Properties

| Property             | Description                                                                                                                                | Required |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| `topic_name`         | The topic in the format of `{topic_type}://{tenant}/{namespace}/{topic_name}`                                                              | Yes      |
| `subscription_name`  | The subscription name                                                                                                                      | Yes      |
| `position`           | The initial position (`earliest` or `latest`). Default is "latest"                                                                          | No       |

## Importing existing resources

All resources could be imported using the [standard terraform way](https://www.terraform.io/docs/import/usage.html).

### Example

```shell
terraform import pulsar_cluster.standalone standalone
```

# Testing the Provider

- Change directory to the project </path/to/provider/terraform-provider-pulsar>
- In order to test the provider, you can run `make test`
- In order to run the full suite of Acceptance tests, run `make testacc`

> Note: Acceptance tests create real resources, and often cost money to run.

# Contributing

Contributions are warmly welcomed and greatly appreciated! Please read the [contribution guidelines](.github/CONTRIBUTING.md) for more details.

Issues on GitHub are intended to be related to bugs or feature requests with the provider codebase. For a list of community resources to ask questions about Terraform, see https://www.terraform.io/docs/extend/community/index.html.

# Licence

This library is licensed under the terms of the [Apache License 2.0](LICENSE) and may include packages written by third parties which carry their own copyright notices and license terms.

# About StreamNative

Founded in 2019 by the original creators of Apache Pulsar, [StreamNative](https://streamnative.io) is one of the leading contributors to the open-source Apache Pulsar project. We have helped engineering teams worldwide make the move to Pulsar with [StreamNative Cloud](https://streamnative.io/product), a fully managed service to help teams accelerate time-to-production.

