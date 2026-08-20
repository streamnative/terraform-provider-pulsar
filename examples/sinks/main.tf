// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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

resource "pulsar_sink" "example" {
  tenant    = "public"
  namespace = "default"
  name      = "jdbc-sink"

  # Replace with an existing local archive or supported HTTP(S)/package URL.
  archive = "/path/to/pulsar-io-jdbc-postgres.nar"
  inputs  = ["persistent://public/default/input"]

  cleanup_subscription = false
  auto_ack             = true

  # Values in `configs` are stored and returned in plaintext, including in the
  # function metadata topic, `pulsar-admin sinks get` output and Terraform
  # state. Keep credentials out of it and reference them through the worker's
  # secrets provider instead:
  #
  #   configs = jsonencode({
  #     jdbcUrl   = "jdbc:postgresql://localhost:5432/pulsar"
  #     tableName = "events"
  #     userName  = "postgres"
  #   })
  #
  #   secrets = jsonencode({
  #     password = {
  #       path = "postgres-credentials"
  #       key  = "password"
  #     }
  #   })
  #
  # Note that `{path, key}` references are resolved only by a runtime whose
  # secrets provider understands them, such as the Kubernetes runtime's
  # KubernetesSecretsProviderConfigurator. Under the default
  # ClearTextSecretsProvider - which includes the standalone cluster started by
  # `make run-pulsar-in-docker` - apply succeeds but the sink receives no value
  # for the secret at runtime.
  configs = jsonencode({
    jdbcUrl   = "jdbc:postgresql://localhost:5432/pulsar"
    tableName = "events"
    userName  = "postgres"
    password  = "password"
  })
}
