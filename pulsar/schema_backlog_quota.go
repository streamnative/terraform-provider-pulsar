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
	"strconv"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func schemaBacklogQuotaSubset() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"limit_bytes": {
				Type:     schema.TypeString,
				Required: true,
			},
			"limit_seconds": {
				Type:     schema.TypeString,
				Required: true,
			},
			"policy": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func hashBacklogQuotaSubset() schema.SchemaSetFunc {
	return schema.HashResource(schemaBacklogQuotaSubset())
}

type backlogQuota struct {
	utils.BacklogQuota
	backlogQuotaType utils.BacklogQuotaType
}

func unmarshalBacklogQuota(v *schema.Set) ([]*backlogQuota, error) {
	bklQuotas := make([]*backlogQuota, 0)

	for _, quota := range v.List() {
		var bklQuota backlogQuota

		data := quota.(map[string]interface{})
		policyStr := data["policy"].(string)
		limitSizeStr := data["limit_bytes"].(string)
		limitTimeStr := data["limit_seconds"].(string)
		typeStr := data["type"].(string)

		limitSize, err := strconv.Atoi(limitSizeStr)
		if err != nil {
			return bklQuotas, fmt.Errorf("ERROR_PARSE_BACKLOG_QUOTA_LIMIT_BYTES: %w", err)
		}

		limitTime, err := strconv.Atoi(limitTimeStr)
		if err != nil {
			return bklQuotas, fmt.Errorf("ERROR_PARSE_BACKLOG_QUOTA_LIMIT_SECONDS: %w", err)
		}

		policy, err := utils.ParseRetentionPolicy(policyStr)
		if err != nil {
			return bklQuotas, fmt.Errorf("ERROR_INVALID_BACKLOG_QUOTA_POLICY: %w", err)
		}

		backlogQuotaType, err := utils.ParseBacklogQuotaType(typeStr)
		if err != nil {
			return bklQuotas, fmt.Errorf("ERROR_INVALID_BACKLOG_QUOTA_TYPE: %w", err)
		}

		// zero values are fine, even if the ASCII to Int fails
		bklQuota.LimitSize = int64(limitSize)
		bklQuota.LimitTime = int64(limitTime)
		bklQuota.Policy = policy
		bklQuota.backlogQuotaType = backlogQuotaType

		bklQuotas = append(bklQuotas, &bklQuota)
	}

	return bklQuotas, nil
}
