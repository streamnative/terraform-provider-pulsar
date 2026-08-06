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

resource "pulsar_topic" "example" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "permission-example"
  partitions = 0
}

resource "pulsar_permission_grant" "namespace" {
  namespace = "public/default"
  role      = "app-producer"
  actions   = ["produce"]
}

resource "pulsar_permission_grant" "topic" {
  topic = "${pulsar_topic.example.topic_type}://${pulsar_topic.example.tenant}/${pulsar_topic.example.namespace}/${pulsar_topic.example.topic_name}"
  role  = "app-consumer"

  actions = ["consume"]
}
