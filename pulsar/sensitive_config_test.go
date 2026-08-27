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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericConfigSensitivityIsOptIn(t *testing.T) {
	functionSchema := resourcePulsarFunction().Schema
	assert.False(t, functionSchema[resourceFunctionUserConfig].Sensitive)

	for _, key := range []string{resourceFunctionSinkConfigKey, resourceFunctionSourceConfigKey} {
		configResource, ok := functionSchema[key].Elem.(*schema.Resource)
		require.True(t, ok)
		assert.False(t, configResource.Schema[resourceFunctionRuntimeConfigConfigsKey].Sensitive)
	}

	assert.False(t, resourcePulsarSink().Schema[resourceSinkConfigsKey].Sensitive)
	assert.False(t, resourcePulsarSource().Schema[resourceSourceConfigsKey].Sensitive)
	assert.NotNil(t, resourcePulsarSink().Schema[resourceSinkConfigsKey].ValidateFunc)
	assert.NotNil(t, resourcePulsarSource().Schema[resourceSourceConfigsKey].ValidateFunc)
}

func TestJSONValidationDoesNotEchoInput(t *testing.T) {
	secretJSON := `{"password":"secret"`

	_, errors := jsonValidateFunc(secretJSON, "configs")

	require.Len(t, errors, 1)
	assert.EqualError(t, errors[0], "configs must contain valid JSON")
	assert.NotContains(t, errors[0].Error(), "password")
	assert.NotContains(t, errors[0].Error(), "secret")
}

func TestJSONValidationAcceptsValidJSON(t *testing.T) {
	_, errors := jsonValidateFunc(`{"password":"secret"}`, "configs")
	assert.Empty(t, errors)

	_, errors = jsonValidateFunc("", "configs")
	assert.Empty(t, errors)
}
