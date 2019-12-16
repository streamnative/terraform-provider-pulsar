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
	"bytes"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"

	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
	"github.com/streamnative/terraform-provider-pulsar/types"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
)

func resourcePulsarNamespace() *schema.Resource {

	return &schema.Resource{
		Create: resourcePulsarNamespaceCreate,
		Read:   resourcePulsarNamespaceRead,
		Update: resourcePulsarNamespaceUpdate,
		Delete: resourcePulsarNamespaceDelete,
		Exists: resourcePulsarNamespaceExists,

		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["namespace"],
			},
			"tenant": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["tenant"],
			},
			"dispatch_rate": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: descriptions["dispatch_rate"],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dispatch_msg_throttling_rate": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"rate_period_seconds": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"dispatch_byte_throttling_rate": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: dispatchRateToHash,
			},
			"retention_policies": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"retention_minutes": {
							Type:     schema.TypeString,
							Required: true,
						},
						"retention_size_in_mb": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: retentionPoliciesToHash,
			},
			"namespace_config": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: descriptions["namespace_config"],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"anti_affinity": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"max_consumers_per_subscription": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"max_consumers_per_topic": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"max_producers_per_topic": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"replication_clusters": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
				Set: namespaceConfigToHash,
			},
		},
	}
}

func resourcePulsarNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	ok, err := resourcePulsarNamespaceExists(d, meta)
	if err != nil {
		return err
	}

	if ok {
		return resourcePulsarNamespaceRead(d, meta)
	}

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)

	ns := fmt.Sprintf("%s/%s", tenant, namespace)

	if err := client.CreateNamespace(ns); err != nil {
		return fmt.Errorf("ERROR_CREATE_NAMESPACE: %w", err)
	}

	namespaceConfig := d.Get("namespace_config").(*schema.Set)
	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)

	dispatchRate := unmarshalDispatchRate(dispatchRateConfig)
	retentionPolicies := unmarshalRetentionPolicies(retentionPoliciesConfig)
	nsCfg := unmarshalNamespaceConfig(namespaceConfig)

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_READING_NAMESPACE_NAME: %w", err)
	}

	var errs error

	if err = client.SetNamespaceAntiAffinityGroup(nsName.String(), nsCfg.AntiAffinity); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("SetNamespaceAntiAffinityGroup: %w", err))
	}

	if err = client.SetDispatchRate(*nsName, *dispatchRate); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("SetDispatchRate: %w", err))
	}

	if err = client.SetNamespaceReplicationClusters(nsName.String(), nsCfg.ReplicationClusters); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("SetNamespaceReplicationClusters: %w", err))
	}

	if err = client.SetRetention(nsName.String(), *retentionPolicies); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("SetRetention: %w", err))
	}

	if err = client.SetMaxConsumersPerTopic(*nsName, nsCfg.MaxConsumersPerTopic); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerTopic: %w", err))
	}

	if err = client.SetMaxConsumersPerSubscription(*nsName, nsCfg.MaxConsumersPerSubscription); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerSubscript1ion: %w", err))
	}

	if err = client.SetMaxProducersPerTopic(*nsName, nsCfg.MaxProducersPerTopic); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("SetMaxProducersPerTopic: %w", err))
	}

	if errs != nil {
		return fmt.Errorf("ERROR_CREATE_NAMESPACE_CONFIG: %w", errs)
	}

	_ = d.Set("namespace", namespace)
	_ = d.Set("tenant", tenant)

	return resourcePulsarNamespaceRead(d, meta)
}

func resourcePulsarNamespaceRead(d *schema.ResourceData, meta interface{}) error {

	// pulsar client is not required as we just need to find the full namespace name,
	// which can be achieved from utils package from the pulsarctl module
	_ = meta

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	ns, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
	}

	if ns == nil {
		return fmt.Errorf("ERROR_READ_NAMESPACE: %s", "namespace name is nil")
	}

	d.SetId(ns.String())

	_ = d.Set("namespace", namespace)
	_ = d.Set("tenant", tenant)

	return nil
}

func resourcePulsarNamespaceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)
	namespaceConfig := d.Get("namespace_config").(*schema.Set)
	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)

	nsCfg := unmarshalNamespaceConfig(namespaceConfig)
	dispatchRate := unmarshalDispatchRate(dispatchRateConfig)
	retentionPolicies := unmarshalRetentionPolicies(retentionPoliciesConfig)

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_READING_NAMESPACE_NAME: %w", err)
	}

	var errs error

	errs = multierror.Append(errs, client.SetNamespaceAntiAffinityGroup(nsName.String(), nsCfg.AntiAffinity))
	errs = multierror.Append(errs, client.SetDispatchRate(*nsName, *dispatchRate))
	errs = multierror.Append(errs, client.SetNamespaceReplicationClusters(nsName.String(), nsCfg.ReplicationClusters))
	errs = multierror.Append(errs, client.SetRetention(nsName.String(), *retentionPolicies))
	errs = multierror.Append(errs, client.SetMaxConsumersPerTopic(*nsName, nsCfg.MaxConsumersPerTopic))
	errs = multierror.Append(errs, client.SetMaxConsumersPerSubscription(*nsName, nsCfg.MaxConsumersPerSubscription))
	errs = multierror.Append(errs, client.SetMaxProducersPerTopic(*nsName, nsCfg.MaxProducersPerTopic))

	if errs != nil {
		return fmt.Errorf("ERROR_UPDATE_NAMESPACE_CONFIG: %w", errs)
	}

	d.SetId(nsName.String())
	return resourcePulsarNamespaceRead(d, meta)
}

func resourcePulsarNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)

	ns := fmt.Sprintf("%s/%s", tenant, namespace)

	if err := client.DeleteNamespace(ns); err != nil {
		return fmt.Errorf("ERROR_DELETE_NAMESPACE: %w", err)
	}

	_ = d.Set("namespace", "")
	_ = d.Set("tenant", "")
	_ = d.Set("namespace_config", nil)
	_ = d.Set("retention_policies", nil)
	_ = d.Set("split_namespaces", nil)
	_ = d.Set("dispatch_rate", nil)

	return nil
}

func resourcePulsarNamespaceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	namespaceFullPath := fmt.Sprintf("%s/%s", tenant, namespace)

	nsList, err := client.GetNamespaces(tenant)
	if err != nil {
		return false, fmt.Errorf("ERROR_READING_NAMESPACE_NAME: %w", err)
	}

	for _, ns := range nsList {
		if ns == namespaceFullPath {
			return true, nil
		}
	}

	return false, nil
}

func dispatchRateToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%d-", m["dispatch_msg_throttling_rate"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["rate_period_seconds"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["dispatch_byte_throttling_rate"].(int)))

	return hashcode.String(buf.String())
}

func retentionPoliciesToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["retention_minutes"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["retention_size_in_mb"].(string)))

	return hashcode.String(buf.String())
}

func namespaceConfigToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["anti_affinity"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["max_consumers_per_subscription"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["max_consumers_per_topic"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["max_producers_per_topic"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["replication_clusters"].([]interface{})))

	return hashcode.String(buf.String())
}

func unmarshalDispatchRate(v *schema.Set) *utils.DispatchRate {
	var dispatchRate utils.DispatchRate

	for _, dr := range v.List() {
		data := dr.(map[string]interface{})

		dispatchRate.DispatchThrottlingRateInByte = int64(data["dispatch_byte_throttling_rate"].(int))
		dispatchRate.DispatchThrottlingRateInMsg = data["dispatch_msg_throttling_rate"].(int)
		dispatchRate.RatePeriodInSecond = data["rate_period_seconds"].(int)
	}

	return &dispatchRate
}

func unmarshalRetentionPolicies(v *schema.Set) *utils.RetentionPolicies {
	var rtnPolicies utils.RetentionPolicies

	for _, policy := range v.List() {
		data := policy.(map[string]interface{})

		retentionMinutes, _ := strconv.Atoi(data["retention_minutes"].(string))
		retentionMB, _ := strconv.Atoi(data["retention_size_in_mb"].(string))

		// zero values are fine, even if the ASCII to Int fails
		rtnPolicies.RetentionTimeInMinutes = retentionMinutes
		rtnPolicies.RetentionSizeInMB = int64(retentionMB)
	}

	return &rtnPolicies
}

func unmarshalNamespaceConfig(v *schema.Set) *types.NamespaceConfig {
	var nsConfig types.NamespaceConfig

	for _, ns := range v.List() {
		data := ns.(map[string]interface{})
		rplClusters := data["replication_clusters"].([]interface{})

		nsConfig.ReplicationClusters = handleHCLArrayV2(rplClusters)
		nsConfig.MaxProducersPerTopic = data["max_producers_per_topic"].(int)
		nsConfig.MaxConsumersPerTopic = data["max_consumers_per_topic"].(int)
		nsConfig.MaxConsumersPerSubscription = data["max_consumers_per_subscription"].(int)
		nsConfig.AntiAffinity = data["anti_affinity"].(string)
	}

	return &nsConfig
}
