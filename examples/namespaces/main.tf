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
      version = "0.1.3"
      source = "registry.terraform.io/streamnative/pulsar"
    }
  }
}

provider "pulsar" {}

resource "pulsar_cluster" "test_cluster" {
  cluster = "skrulls"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters = [
    "standalone"]
  }
}

resource "pulsar_tenant" "test_tenant" {
  tenant = "thanos"
  allowed_clusters = [
    pulsar_cluster.test_cluster.cluster,
  "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "eternals"

  namespace_config {
    anti_affinity                         = "anti-aff"
    max_consumers_per_subscription        = "50"
    max_consumers_per_topic               = "50"
    max_producers_per_topic               = "50"
    message_ttl_seconds                   = "86400"
    replication_clusters                  = ["standalone"]
    subscription_expiration_time_minutes  = 90
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
    policy = "producer_request_hold"
  }

  topic_auto_creation {
    enable = true
    type = "partitioned"
    partitions = 3
  }
}
