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
      source = "registry.terraform.io/apache/pulsar"
    }
  }
}

provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

// Note: function resource requires v3 api.
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
          "HELLO": "WORLD"
      }
  })
}