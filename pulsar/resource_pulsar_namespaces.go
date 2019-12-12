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
				Optional:    true,
				Description: descriptions["tenant"],
				Default:     "",
			},
			"dispatch_rate": {
				Type:     schema.TypeSet,
				Optional: true,
				//Default:     *utils.NewDispatchRate(),
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
			"split_namespaces": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bundle": {
							Type:     schema.TypeString,
							Required: true,
						},
						"unload_split_bundles": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: splitNamespacesToHash,
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

	var nsCfg types.NamespaceConfig

	namespaceConfig := d.Get("namespace_config").(*schema.Set)
	retentionPolicies := d.Get("retention_policies").(*schema.Set)
	splitNSConfig := d.Get("split_namespaces").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)

	for _, dr := range dispatchRateConfig.List() {
		data := dr.(map[string]interface{})

		nsCfg.DispatchRate.DispatchThrottlingRateInMsg = data["dispatch_msg_throttling_rate"].(int)
		nsCfg.DispatchRate.RatePeriodInSecond = data["rate_period_seconds"].(int)
		nsCfg.DispatchRate.DispatchThrottlingRateInByte = int64(data["dispatch_byte_throttling_rate"].(int))
	}

	for _, rp := range retentionPolicies.List() {
		data := rp.(map[string]interface{})

		retMin, err := strconv.Atoi(data["retention_minutes"].(string))
		if err != nil {
			return fmt.Errorf("ERROR_DATA_TYPE_CONVERSION: %w", err)
		}
		retSize, err := strconv.Atoi(data["retention_size_in_mb"].(string))
		if err != nil {
			return fmt.Errorf("ERROR_DATA_TYPE_CONVERSION: %w", err)
		}

		nsCfg.RetentionPolicies.RetentionTimeInMinutes = retMin
		nsCfg.RetentionPolicies.RetentionSizeInMB = int64(retSize)
	}

	for _, cfg := range splitNSConfig.List() {
		data := cfg.(map[string]interface{})

		usb := data["unload_split_bundles"].(string)

		if usb == "1" {
			nsCfg.SplitNamespaces.Bundle = data["bundle"].(string)
			nsCfg.SplitNamespaces.UnloadSplitBundles = true
			break
		}

		nsCfg.SplitNamespaces.Bundle = data["bundle"].(string)
		nsCfg.SplitNamespaces.UnloadSplitBundles = false

	}

	for _, cfg := range namespaceConfig.List() {
		data := cfg.(map[string]interface{})

		nsCfg.AntiAffinity = data["anti_affinity"].(string)
		nsCfg.MaxConsumersPerSubscription = data["max_consumers_per_subscription"].(int)
		nsCfg.MaxConsumersPerTopic = data["max_consumers_per_topic"].(int)
		nsCfg.MaxProducersPerTopic = data["max_producers_per_topic"].(int)
		nsCfg.ReplicationClusters = handleHCLArrayV2(data["replication_clusters"].([]interface{}))

	}

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return err
	}

	var errs error

	if err = client.SetNamespaceAntiAffinityGroup(nsName.String(), nsCfg.AntiAffinity); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SetNamespaceAntiAffinityGroup"))
	}
	if err = client.SetDispatchRate(*nsName, nsCfg.DispatchRate); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SetDispatchRate"))

	}
	if err = client.SetNamespaceReplicationClusters(nsName.String(), nsCfg.ReplicationClusters); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SetNamespaceReplicationClusters"))

	}
	if err = client.SetRetention(nsName.String(), nsCfg.RetentionPolicies); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SetRetention"))

	}
	if err = client.SplitNamespaceBundle(nsName.String(), nsCfg.SplitNamespaces.Bundle, nsCfg.SplitNamespaces.UnloadSplitBundles); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SplitNamespaceBundle"))
	}
	if err = client.SetMaxConsumersPerTopic(*nsName, nsCfg.MaxConsumersPerTopic); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerTopic"))
	}
	if err = client.SetMaxConsumersPerSubscription(*nsName, nsCfg.MaxConsumersPerSubscription); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerSubscription"))
	}
	if err = client.SetMaxProducersPerTopic(*nsName, nsCfg.MaxProducersPerTopic); err != nil {
		err = multierror.Append(errs, fmt.Errorf("SetMaxProducersPerTopic"))
	}

	if errs != nil {
		return fmt.Errorf("ERROR_CREATE_NAMESPACE_CONFIG: %w", errs)
	}

	_ = d.Set("namespace", namespace)
	_ = d.Set("tenant", tenant)

	return resourcePulsarNamespaceRead(d, meta)
}

func resourcePulsarNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	_, err := client.GetNamespaces(tenant)
	if err != nil {
		return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
	}

	nsFullPAth := fmt.Sprintf("%s/%s", tenant, namespace)
	d.SetId(nsFullPAth)

	_ = d.Set("namespace", namespace)
	_ = d.Set("tenant", tenant)

	return nil
}

func resourcePulsarNamespaceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)
	namespaceConfig := d.Get("namespace_config").(*schema.Set)
	retentionPolicies := d.Get("retention_policies").(*schema.Set)
	splitNSConfig := d.Get("split_namespaces").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)

	var nsCfg types.NamespaceConfig

	for _, dr := range dispatchRateConfig.List() {
		data := dr.(map[string]interface{})

		nsCfg.DispatchRate.DispatchThrottlingRateInMsg = data["dispatch_msg_throttling_rate"].(int)
		nsCfg.DispatchRate.RatePeriodInSecond = data["rate_period_seconds"].(int)
		nsCfg.DispatchRate.DispatchThrottlingRateInByte = int64(data["dispatch_byte_throttling_rate"].(int))
	}

	for _, rp := range retentionPolicies.List() {
		data := rp.(map[string]interface{})

		retMin, err := strconv.Atoi(data["retention_minutes"].(string))
		if err != nil {
			return fmt.Errorf("ERROR_DATA_TYPE_CONVERSION: %w", err)
		}
		retSize, err := strconv.Atoi(data["retention_size_in_mb"].(string))
		if err != nil {
			return fmt.Errorf("ERROR_DATA_TYPE_CONVERSION: %w", err)
		}

		nsCfg.RetentionPolicies.RetentionTimeInMinutes = retMin
		nsCfg.RetentionPolicies.RetentionSizeInMB = int64(retSize)
	}

	for _, cfg := range splitNSConfig.List() {
		data := cfg.(map[string]interface{})

		usb := data["unload_split_bundles"].(string)

		if usb == "1" {
			nsCfg.SplitNamespaces.Bundle = data["bundle"].(string)
			nsCfg.SplitNamespaces.UnloadSplitBundles = true
			break
		}

		nsCfg.SplitNamespaces.Bundle = data["bundle"].(string)
		nsCfg.SplitNamespaces.UnloadSplitBundles = false

	}

	for _, cfg := range namespaceConfig.List() {
		data := cfg.(map[string]interface{})

		nsCfg.AntiAffinity = data["anti_affinity"].(string)
		nsCfg.MaxConsumersPerSubscription = data["max_consumers_per_subscription"].(int)
		nsCfg.MaxConsumersPerTopic = data["max_consumers_per_topic"].(int)
		nsCfg.MaxProducersPerTopic = data["max_producers_per_topic"].(int)
		nsCfg.ReplicationClusters = handleHCLArrayV2(data["replication_clusters"].([]interface{}))

	}

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return err
	}

	var errs error

	errs = multierror.Append(errs, client.SetNamespaceAntiAffinityGroup(nsName.String(), nsCfg.AntiAffinity))
	errs = multierror.Append(errs, client.SetDispatchRate(*nsName, nsCfg.DispatchRate))
	errs = multierror.Append(errs, client.SetNamespaceReplicationClusters(nsName.String(), nsCfg.ReplicationClusters))
	errs = multierror.Append(errs, client.SetRetention(nsName.String(), nsCfg.RetentionPolicies))
	errs = multierror.Append(errs, client.SplitNamespaceBundle(nsName.String(), nsCfg.SplitNamespaces.Bundle,
		nsCfg.SplitNamespaces.UnloadSplitBundles))
	errs = multierror.Append(errs, client.SetMaxConsumersPerTopic(*nsName, nsCfg.MaxConsumersPerTopic))
	errs = multierror.Append(errs, client.SetMaxConsumersPerSubscription(*nsName, nsCfg.MaxConsumersPerSubscription))
	errs = multierror.Append(errs, client.SetMaxProducersPerTopic(*nsName, nsCfg.MaxProducersPerTopic))

	if errs != nil {
		return fmt.Errorf("ERROR_UPDATE_NAMESPACE_CONFIG: %w", errs)
	}
	//return resourcePulsarNamespaceRead(d, meta)
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

	return nil
}

func resourcePulsarNamespaceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	namespaceFullPath := fmt.Sprintf("%s/%s", tenant, namespace)

	nsList, err := client.GetNamespaces(tenant)
	if err != nil {
		return false, err
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

func splitNamespacesToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["bundle"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["unload_split_bundles"].(string)))

	return hashcode.String(buf.String())
}

func namespaceConfigToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["anti_affinity"].(string)))
	//buf.WriteString(fmt.Sprintf("%v-", m["dispatch_rate"].(*schema.Set)))
	buf.WriteString(fmt.Sprintf("%d-", m["max_consumers_per_subscription"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["max_consumers_per_topic"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["max_producers_per_topic"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["replication_clusters"].([]interface{})))
	//buf.WriteString(fmt.Sprintf("%v-", m["retention_policies"].(*schema.Set)))
	//buf.WriteString(fmt.Sprintf("%v-", m["split_namespaces"].(*schema.Set)))

	return hashcode.String(buf.String())
}
