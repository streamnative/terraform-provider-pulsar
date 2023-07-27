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
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/streamnative/pulsar-admin-go/pkg/admin/config"
	"github.com/streamnative/pulsar-admin-go/pkg/rest"
	"github.com/streamnative/pulsar-admin-go/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func init() {
	initTestWebServiceURL()
}

func TestFunction(t *testing.T) {
	configBytes, err := ioutil.ReadFile("testdata/function/main.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheckWithAPIVersion(t, config.V3) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: string(configBytes),
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_function.function-1"
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					config, err := getPulsarFunctionByResourceID(rs.Primary.ID)
					if err != nil {
						return err
					}

					if config == nil {
						return fmt.Errorf("failed to create %s function", rs.Primary.ID)
					}

					assert.Equal(t, "function-1", config.Name)
					assert.Equal(t, "public", config.Tenant)
					assert.Equal(t, "default", config.Namespace)
					assert.Equal(t, ProcessingGuaranteesEffectivelyOnce, config.ProcessingGuarantees)
					assert.Equal(t, 666, config.TimeoutMs)
					assert.NotNil(t, config.Resources)

					return nil
				}),
			},
		},
	})
}

func testPulsarFunctionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_function" {
			continue
		}

		config, err := getPulsarFunctionByResourceID(rs.Primary.ID)
		if err != nil {
			return err
		}

		if config != nil {
			return fmt.Errorf("function still exists")
		}
	}

	return nil
}

func getPulsarFunctionByResourceID(id string) (*utils.FunctionConfig, error) {
	client := getClientFromMeta(testAccProvider.Meta()).Functions()

	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, errors.New("Primary ID should be tenant/namespace/name format")
	}

	resp, err := client.GetFunction(parts[0], parts[1], parts[2])
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			return nil, nil
		}
	}

	return &resp, nil
}
