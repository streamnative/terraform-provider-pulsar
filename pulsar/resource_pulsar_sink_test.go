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

package pulsar

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/pkg/errors"
	"github.com/streamnative/pulsarctl/pkg/cli"
	"github.com/streamnative/pulsarctl/pkg/pulsar"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func init() {
	initTestWebServiceURL()
}

func TestSink(t *testing.T) {
	configBytes, err := ioutil.ReadFile("testdata/sink/main.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSinkDestroy,
		Steps: []resource.TestStep{
			{
				Config: string(configBytes),
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_sink.sink-1"
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					client := testAccProvider.Meta().(pulsar.Client).Sinks()

					parts := strings.Split(rs.Primary.ID, "/")
					if len(parts) != 3 {
						return errors.New("resource id should be tenant/namespace/name format")
					}

					_, err := client.GetSink(parts[0], parts[1], parts[2])
					if err != nil {
						return err
					}

					return nil
				}),
			},
		},
	})
}

func testPulsarSinkDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(pulsar.Client).Sinks()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_sink" {
			continue
		}

		id := rs.Primary.ID
		parts := strings.Split(id, "/")
		if len(parts) != 3 {
			return errors.New("id should be tenant/namespace/name format")
		}

		resp, err := client.GetSink(parts[0], parts[1], parts[2])
		if err != nil {
			if cliErr, ok := err.(cli.Error); ok && cliErr.Code == 404 {
				return nil
			}

			return err
		}

		if resp.Name != "" {
			return fmt.Errorf("%s still exist", id)
		}
	}

	return nil
}
