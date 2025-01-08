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

# How to contribute

If you would like to contribute code to this project, fork the repository and send a pull request.

## Prerequisite

In this project, use `go mod` as the package management tool and make sure your Go version is higher then `Go 1.18`.

## Fork

Before contributing, you need to
fork [terraform-provider-pulsar](https://github.com/streamnative/terraform-provider-pulsar) to your GitHub account.

## Contribution flow

```bash
$ git remote add tfpulsar https://github.com/streamnative/terraform-provider-pulsar.git
# sync with the remote master
$ git checkout master
$ git fetch tfpulsar
$ git rebase tfpulsar/master
$ git push origin master
# create a PR branch
$ git checkout -b your_branch   
# do something
$ git add [your change files]
$ git commit -sm "xxx"
$ git push origin your_branch
```

## Configure Jetbrains - GoLand

The `terraform-provider-pulsar` uses `go mod` to manage dependencies, so make sure your IDE enables `Go Modules(vgo)`.

To configure annotation processing in GoLand, follow the steps below.

1. To open the **Go Modules Settings** window, in GoLand, click **Preferences** > **Go** > **Go Modules(vgo)**.

2. Select the **Enable Go Modules(vgo) integration** checkbox.

3. Click **Apply** and **OK**.

## Code style

We use Go Community Style Guide in `terraform-provider-pulsar`.
For more information, see [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
Also please refer to [Terraform Community Guidelines](https://www.hashicorp.com/community-guidelines)
Always use gofmt and linter (linter config is provided), before submitting any code changes.

To make your pull request easy to review, maintain and develop, follow this style.

## Test Scripts

To make sure that the changes/features/bug-fixes submitted by you does not break anything, always include test scripts
for your work. If you're implementing a feature/pulsar entity, please submit an Acceptance Test as well.
You can learn more about Terraform Acceptance Tests [here](https://www.terraform.io/docs/extend/testing/index.html)

## Create a new file

The project uses the open source protocol of Apache License 2.0. If you need to create a new file when developing new
features,
add the license at the beginning of each file. The location of the header file: [header file](../.header).

## Update dependencies

The `terraform-provider-pulsar` uses [Go module](https://go.dev/wiki/Modules) to manage dependencies.
To add or update a dependency, use the `go mod edit` command to change the dependencies
