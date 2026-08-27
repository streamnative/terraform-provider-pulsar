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
	"context"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestFunctionProducerConfigRawConfigPreservesOmittedFields(t *testing.T) {
	res := resourcePulsarFunction()
	remoteProducerConfig := utils.ProducerConfig{
		MaxPendingMessages:                 1000,
		MaxPendingMessagesAcrossPartitions: 50000,
		UseThreadLocalProducers:            true,
		BatchBuilder:                       "KEY_BASED",
		CompressionType:                    "ZSTD",
	}
	state := functionProducerState(t, remoteProducerConfig, "old")
	config := functionConfigWithBase(map[string]interface{}{
		resourceFunctionOutputKey:          "old",
		resourceFunctionPCMaxPendingMsgKey: 0,
	})
	legacyDiff, err := res.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), nil)
	if err != nil {
		t.Fatal(err)
	}
	prior, err := schema.StateValueFromInstanceState(state, res.CoreConfigSchema().ImpliedType())
	if err != nil {
		t.Fatal(err)
	}
	planned, err := schema.ApplyDiff(prior, legacyDiff, res.CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}
	rawConfig, err := schema.JSONMapToStateValue(config, res.CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}
	diff, err := schema.DiffFromValues(context.Background(), prior, planned, rawConfig, res)
	if err != nil {
		t.Fatal(err)
	}

	var sent *utils.ProducerConfig
	res.UpdateContext = func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
		functionConfig, err := marshalFunctionConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		sent = functionConfig.ProducerConfig
		return nil
	}
	_, diags := res.Apply(context.Background(), state, diff, nil)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if sent == nil {
		t.Fatal("expected explicit zero to send a producer config")
	}
	if sent.MaxPendingMessages != 0 ||
		sent.MaxPendingMessagesAcrossPartitions != remoteProducerConfig.MaxPendingMessagesAcrossPartitions ||
		sent.UseThreadLocalProducers != remoteProducerConfig.UseThreadLocalProducers ||
		sent.BatchBuilder != remoteProducerConfig.BatchBuilder ||
		sent.CompressionType != remoteProducerConfig.CompressionType {
		t.Fatalf("unexpected producer config: %#v", sent)
	}
}
