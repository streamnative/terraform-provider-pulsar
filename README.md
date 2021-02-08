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

<img alt="" src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

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

- [Terraform](https://www.terraform.io/downloads.html) 0.10+
- [Go](https://golang.org/doc/install) 1.13 (to build the provider plugin)

Installation
------------

* Clone this repository and cd into the directory
* Run `make build`, it will out a file named `terraform-provider-pulsar`
* Copy this `terraform-provider-pulsar` bin file to your [terraform plugin directory][third-party-plugins]
* Typically this plugin directory is `~/.terraform.d/plugins/`
* On Linux based 64-bit devices, this directory can be `~/.terraform.d/plugins/linux_amd64`


Testing the Apache Pulsar Terraform Provider
------------

* change directory to the project </path/to/provider/terraform-provider-pulsar>
* Make sure you have the required tools installed. Run `make tools`
* In order to test the provider, you can run `make test`

* In order to run the full suite of Acceptance tests, run `make testacc`

`Note: Acceptance tests create real resources, and often cost money to run.`



Provider Configuration
------------

### Example

Example provider with apache pulsar cluster, running locally with authentication disabled.

```hcl
terraform {
  required_providers {
    pulsar = {
      versions = ["1.0.0"]
      source = "registry.terraform.io/apache/pulsar"
    }
  }
}

provider "pulsar" {
  web_service_url = "http://localhost:8080"
  token = "my_auth_token"
}
```

| Property                      | Description                                                                                                           | Required    |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------- | ---------- |
| `web_service_url`             | URL of your Apache Pulsar Cluster                             | `Yes` |
| `token`           | Authentication Token for your Apache Pulsar Cluster, which is required only if your cluster has authentication enabled| `No`       |
| `tls_trust_certs_file_path` | Path to a custom trusted TLS certificate file | `No`       |
| `tls_allow_insecure_connection` | Boolean flag to accept untrusted TLS certificates | `No`       |

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
    policy = "consumer_backlog_eviction"
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
