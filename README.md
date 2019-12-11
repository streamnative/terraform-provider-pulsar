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

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

## Contents

* [Requirements](#requirements)
* [Installation](#installation)
* [Resources](#resources)
  * [`pulsar_tenant`](#pulsar_tenant)

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
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}
```

| Property            | Description                                                                                                           | Default    |
| ------------------- | --------------------------------------------------------------------------------------------------------------------- | ---------- |
| `web_service_url` | URL of your Apache Pulsar Cluster                             | `Required` |
| `token`           | Authentication Token for your Apache Pulsar Cluster, which is required only if your cluster has authentication enabled                              | `""`       |

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
  admin_roles = ["godmode"]
}
```

#### Properties

| Property                      | Description                                         | Required                   |
| ----------------------------- | --------------------------------------------------- |----------------------------|
| `tenant`                      | Name of the Tenant that you want to create          | Yes
| `allowed_clusters`            | An Array of clusters, accessible by this tenant     | No
| `admin_roles`                 | Admin Roles to be assumed by this Tenant            | No

[1]: https://www.terraform.io
[2]: https://pulsar.apache.org
[third-party-plugins]: https://www.terraform.io/docs/configuration/providers.html#third-party-plugins
[install-go]: https://golang.org/doc/install#install
