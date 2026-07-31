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
	"github.com/stretchr/testify/require"
)

func TestPulsarNamespacePolicyBlocksAreOptionalComputedSets(t *testing.T) {
	t.Parallel()

	resourceSchema := resourcePulsarNamespace().Schema
	for _, attr := range []string{
		"dispatch_rate",
		"subscription_dispatch_rate",
		"persistence_policies",
		"backlog_quota",
	} {
		attr := attr
		t.Run(attr, func(t *testing.T) {
			t.Parallel()
			policySchema := resourceSchema[attr]
			require.Equal(t, schema.TypeSet, policySchema.Type)
			require.True(t, policySchema.Optional)
			require.True(t, policySchema.Computed)
			require.False(t, policySchema.Required)
			require.False(t, policySchema.ForceNew)
		})
	}
}

func TestPulsarNamespaceBacklogQuotaOwnershipStateIsComputedOnly(t *testing.T) {
	t.Parallel()

	ownershipSchema := resourcePulsarNamespace().Schema[backlogQuotaManagedTypesStateAttr]
	require.Equal(t, schema.TypeSet, ownershipSchema.Type)
	require.True(t, ownershipSchema.Computed)
	require.False(t, ownershipSchema.Optional)
	require.False(t, ownershipSchema.Required)
	require.False(t, ownershipSchema.ForceNew)
	require.Equal(t, schema.TypeString, ownershipSchema.Elem.(*schema.Schema).Type)
}

func TestNamespaceAndTopicDispatchRateDescriptionsAreDistinct(t *testing.T) {
	t.Parallel()

	namespaceDescription := resourcePulsarNamespace().Schema["dispatch_rate"].Description
	topicDescription := resourcePulsarTopic().Schema["dispatch_rate"].Description

	require.Contains(t, namespaceDescription, "namespace")
	require.Contains(t, namespaceDescription, "terraform import")
	require.Equal(t, "Topic-level data transfer rate for the given topic", topicDescription)
}
