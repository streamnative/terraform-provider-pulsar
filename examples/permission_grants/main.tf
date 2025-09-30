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
      source  = "registry.terraform.io/streamnative/pulsar"
    }
  }
}

provider "pulsar" {}

resource "pulsar_cluster" "test_cluster" {
  cluster = "marvel"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "pulsar://localhost:6050"
    peer_clusters      = ["standalone"]
  }
}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "avengers"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test_namespace" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "heroes"
}

# Grant produce permissions to application producer role
resource "pulsar_permission_grant" "app_producer" {
  namespace = "${pulsar_tenant.test_tenant.tenant}/${pulsar_namespace.test_namespace.namespace}"
  role      = "app-producer"
  actions   = ["produce"]
}

# Grant consume permissions to application consumer role  
resource "pulsar_permission_grant" "app_consumer" {
  namespace = "${pulsar_tenant.test_tenant.tenant}/${pulsar_namespace.test_namespace.namespace}"
  role      = "app-consumer" 
  actions   = ["consume"]
}

# Grant full permissions to admin role
resource "pulsar_permission_grant" "admin_access" {
  namespace = "${pulsar_tenant.test_tenant.tenant}/${pulsar_namespace.test_namespace.namespace}"
  role      = "admin-user"
  actions   = ["produce", "consume", "functions"]
}

# Create topic for topic-level permissions
resource "pulsar_topic" "test_topic" {
  tenant     = pulsar_tenant.test_tenant.tenant
  namespace  = pulsar_namespace.test_namespace.namespace
  topic_type = "persistent"
  topic_name = "important-topic"
}

# Topic permission grant example
resource "pulsar_permission_grant" "topic_producer" {
  topic   = "persistent://avengers/heroes/important-topic"
  role    = "topic-producer"
  actions = ["produce"]
}

resource "pulsar_permission_grant" "topic_consumer" {
  topic   = "persistent://avengers/heroes/important-topic"
  role    = "topic-consumer"
  actions = ["consume"]
}

resource "pulsar_permission_grant" "topic_admin" {
  topic   = "persistent://avengers/heroes/important-topic"
  role    = "topic-admin"
  actions = ["produce", "consume", "functions", "sources", "sinks", "packages"]
}