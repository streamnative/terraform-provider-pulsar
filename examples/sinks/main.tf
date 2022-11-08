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
      version = "0.1.0"
      source = "registry.terraform.io/apache/pulsar"
    }
  }
}

provider "pulsar" {
  web_service_url = "http://localhost:8080"
  api_version = "3"
}

// Note: sink resource requires v3 api.
resource "pulsar_sink" "sink-1" {
  provider = pulsar

  name = "sink-1"
  tenant = "public"
  namespace = "default"
  inputs = ["sink-1-topic"]
  subscription_position = "Latest"
  cleanup_subscription = false
  parallelism = 1
  auto_ack = true

  processing_guarantees = "EFFECTIVELY_ONCE"

  cpu = 1
  ram_mb = 2048
  disk_mb = 102400

  archive = "https://www.apache.org/dyn/mirrors/mirrors.cgi?action=download&filename=pulsar/pulsar-2.8.1/connectors/pulsar-io-jdbc-postgres-2.8.1.nar"
  configs = "{\"jdbcUrl\":\"jdbc:postgresql://localhost:5432/pulsar_postgres_jdbc_sink\",\"password\":\"password\",\"tableName\":\"pulsar_postgres_jdbc_sink\",\"userName\":\"postgres\"}"
}
