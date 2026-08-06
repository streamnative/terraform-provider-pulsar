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

# Contributing

Use Go version declared in [`go.mod`](../go.mod). CI currently validates with Terraform 1.2.7.

## Workflow

1. Fork repository and create a focused branch from `master`.
2. Match existing Go and Terraform style.
3. Add unit tests; add acceptance coverage for provider behavior changes.
4. Regenerate docs after schema or description changes.

## Checks

```shell
make tools # once
make test
make lint
go generate ./...
git diff --exit-code -- docs
```

Acceptance tests create real Pulsar resources:

```shell
make run-pulsar-in-docker
make testacc
make remove-pulsar-from-docker
```

New source files must include Apache License 2.0 header from [`.header`](../.header).
