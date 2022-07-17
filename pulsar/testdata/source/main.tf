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

provider "pulsar" {
  web_service_url = "http://localhost:8080"
  api_version = "3"
}

resource "pulsar_source" "source-1" {
  provider = pulsar

  name = "source-1"
  tenant = "public"
  namespace = "default"

  archive = "https://www.apache.org/dyn/mirrors/mirrors.cgi?action=download&filename=pulsar/pulsar-2.8.1/connectors/pulsar-io-file-2.8.1.nar"

  destination_topic_name = "source-1-topic"

  processing_guarantees = "EFFECTIVELY_ONCE"

  configs = "{\"inputDirectory\":\"opt\"}"

  cpu = 2
  disk_mb = 20480
  ram_mb = 2048
}