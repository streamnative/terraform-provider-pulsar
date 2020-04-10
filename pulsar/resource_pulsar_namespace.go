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
	"strings"

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
			"enable_deduplication": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"dispatch_rate": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: descriptions["dispatch_rate"],
				MaxItems:    1,
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
				MaxItems: 1,
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
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"anti_affinity": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(string)
								if len(strings.Trim(strings.TrimSpace(v), "\"")) == 0 {
									errs = append(errs, fmt.Errorf("%q must not be empty", key))
								}
								return
							},
						},
						"max_consumers_per_subscription": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(int)
								if v < 0 {
									errs = append(errs, fmt.Errorf("%q must be 0 or more, got: %d", key, v))
								}
								return
							},
						},
						"max_consumers_per_topic": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(int)
								if v < 0 {
									errs = append(errs, fmt.Errorf("%q must be 0 or more, got: %d", key, v))
								}
								return
							},
						},
						"max_producers_per_topic": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(int)
								if v < 0 {
									errs = append(errs, fmt.Errorf("%q must be 0 or more, got: %d", key, v))
								}
								return
							},
						},
						"replication_clusters": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
				Set: namespaceConfigToHash,
			},
			"persistence_policies": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bookkeeper_ensemble": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"bookkeeper_write_quorum": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"bookkeeper_ack_quorum": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"managed_ledger_max_mark_delete_rate": {
							Type:     schema.TypeFloat,
							Required: true,
						},
					},
				},
				Set: persistencePoliciesToHash,
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

	enableDeduplication, deduplicationDefined := d.GetOk("enable_deduplication")
	namespaceConfig := d.Get("namespace_config").(*schema.Set)
	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)
	persistencePoliciesConfig := d.Get("persistence_policies").(*schema.Set)

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_READING_NAMESPACE_NAME: %w", err)
	}

	var errs error

	if namespaceConfig.Len() > 0 {
		nsCfg := unmarshalNamespaceConfig(namespaceConfig)

		if len(nsCfg.AntiAffinity) > 0 {
			if err = client.SetNamespaceAntiAffinityGroup(nsName.String(), nsCfg.AntiAffinity); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceAntiAffinityGroup: %w", err))
			}
		}

		if len(nsCfg.ReplicationClusters) > 0 {
			if err = client.SetNamespaceReplicationClusters(nsName.String(), nsCfg.ReplicationClusters); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceReplicationClusters: %w", err))
			}
		}

		if nsCfg.MaxConsumersPerTopic >= 0 {
			if err = client.SetMaxConsumersPerTopic(*nsName, nsCfg.MaxConsumersPerTopic); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerTopic: %w", err))
			}
		}

		if nsCfg.MaxConsumersPerSubscription >= 0 {
			if err = client.SetMaxConsumersPerSubscription(*nsName, nsCfg.MaxConsumersPerSubscription); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerSubscription: %w", err))
			}
		}

		if nsCfg.MaxProducersPerTopic >= 0 {
			if err = client.SetMaxProducersPerTopic(*nsName, nsCfg.MaxProducersPerTopic); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxProducersPerTopic: %w", err))
			}
		}
	}

	if dispatchRateConfig.Len() > 0 {
		dispatchRate := unmarshalDispatchRate(dispatchRateConfig)

		if err = client.SetDispatchRate(*nsName, *dispatchRate); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetDispatchRate: %w", err))
		}
	}

	if retentionPoliciesConfig.Len() > 0 {
		retentionPolicies := unmarshalRetentionPolicies(retentionPoliciesConfig)

		if err = client.SetRetention(nsName.String(), *retentionPolicies); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetRetention: %w", err))
		}
	}

	if persistencePoliciesConfig.Len() > 0 {
		persistencePolicies := unmarshalPersistencePolicies(persistencePoliciesConfig)

		if err = client.SetPersistence(nsName.String(), *persistencePolicies); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetPersistence: %w", err))
		}
	}

	if deduplicationDefined {
		if err = client.SetDeduplicationStatus(nsName.String(), enableDeduplication.(bool)); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetDeduplicationStatus: %w", err))
		}
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

	if namespaceConfig, ok := d.GetOk("namespace_config"); ok && namespaceConfig.(*schema.Set).Len() > 0 {
		afgrp, err := client.GetNamespaceAntiAffinityGroup(ns.String())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetNamespaceAntiAffinityGroup: %w", err)
		}

		maxConsPerSub, err := client.GetMaxConsumersPerSubscription(*ns)
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxConsumersPerSubscription: %w", err)
		}

		maxConsPerTopic, err := client.GetMaxConsumersPerTopic(*ns)
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxConsumersPerTopic: %w", err)
		}

		maxProdPerTopic, err := client.GetMaxProducersPerTopic(*ns)
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxProducersPerTopic: %w", err)
		}

		replClustersRaw, err := client.GetNamespaceReplicationClusters(ns.String())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxProducersPerTopic: %w", err)
		}

		replClusters := make([]interface{}, len(replClustersRaw))
		for i, cl := range replClustersRaw {
			replClusters[i] = cl
		}

		_ = d.Set("namespace_config", schema.NewSet(namespaceConfigToHash, []interface{}{
			map[string]interface{}{
				"anti_affinity":                  strings.Trim(strings.TrimSpace(afgrp), "\""),
				"max_consumers_per_subscription": maxConsPerSub,
				"max_consumers_per_topic":        maxConsPerTopic,
				"max_producers_per_topic":        maxProdPerTopic,
				"replication_clusters":           replClusters,
			},
		}))
	}

	if persistencePoliciesConfig, ok := d.GetOk("persistence_policies"); ok && persistencePoliciesConfig.(*schema.Set).Len() > 0 {
		persistence, err := client.GetPersistence(ns.String())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetPersistence: %w", err)
		}

		_ = d.Set("persistence_policies", schema.NewSet(persistencePoliciesToHash, []interface{}{
			map[string]interface{}{
				"bookkeeper_ensemble":                 persistence.BookkeeperEnsemble,
				"bookkeeper_write_quorum":             persistence.BookkeeperWriteQuorum,
				"bookkeeper_ack_quorum":               persistence.BookkeeperAckQuorum,
				"managed_ledger_max_mark_delete_rate": persistence.ManagedLedgerMaxMarkDeleteRate,
			},
		}))
	}

	return nil
}

func resourcePulsarNamespaceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)
	enableDeduplication, deduplicationDefined := d.GetOk("enable_deduplication")
	namespaceConfig := d.Get("namespace_config").(*schema.Set)
	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)
	persistencePoliciesConfig := d.Get("persistence_policies").(*schema.Set)

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_READING_NAMESPACE_NAME: %w", err)
	}

	var errs error

	if namespaceConfig.Len() > 0 {
		nsCfg := unmarshalNamespaceConfig(namespaceConfig)

		if len(nsCfg.AntiAffinity) > 0 {
			if err = client.SetNamespaceAntiAffinityGroup(nsName.String(), nsCfg.AntiAffinity); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceAntiAffinityGroup: %w", err))
			}
		}

		if len(nsCfg.ReplicationClusters) > 0 {
			if err = client.SetNamespaceReplicationClusters(nsName.String(), nsCfg.ReplicationClusters); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceReplicationClusters: %w", err))
			}
		}

		if nsCfg.MaxConsumersPerTopic >= 0 {
			if err = client.SetMaxConsumersPerTopic(*nsName, nsCfg.MaxConsumersPerTopic); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerTopic: %w", err))
			}
		}

		if nsCfg.MaxConsumersPerSubscription >= 0 {
			if err = client.SetMaxConsumersPerSubscription(*nsName, nsCfg.MaxConsumersPerSubscription); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerSubscription: %w", err))
			}
		}

		if nsCfg.MaxProducersPerTopic >= 0 {
			if err = client.SetMaxProducersPerTopic(*nsName, nsCfg.MaxProducersPerTopic); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxProducersPerTopic: %w", err))
			}
		}
	}

	if retentionPoliciesConfig.Len() > 0 {
		retentionPolicies := unmarshalRetentionPolicies(retentionPoliciesConfig)
		if err = client.SetRetention(nsName.String(), *retentionPolicies); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetRetention: %w", err))
		}
	}

	if dispatchRateConfig.Len() > 0 {
		dispatchRate := unmarshalDispatchRate(dispatchRateConfig)
		if err = client.SetDispatchRate(*nsName, *dispatchRate); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetDispatchRate: %w", err))
		}
	}

	if persistencePoliciesConfig.Len() > 0 {
		persistencePolicies := unmarshalPersistencePolicies(persistencePoliciesConfig)
		if err = client.SetPersistence(nsName.String(), *persistencePolicies); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetPersistence: %w", err))
		}
	}

	if deduplicationDefined {
		if err = client.SetDeduplicationStatus(nsName.String(), enableDeduplication.(bool)); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetDeduplicationStatus: %w", err))
		}
	}

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
	_ = d.Set("enable_deduplication", nil)
	_ = d.Set("namespace_config", nil)
	_ = d.Set("retention_policies", nil)
	_ = d.Set("dispatch_rate", nil)
	_ = d.Set("persistence_policies", nil)

	return nil
}

func resourcePulsarNamespaceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	namespaceFullPath := fmt.Sprintf("%s/%s", tenant, namespace)

	nsList, err := client.GetNamespaces(tenant)
	if err != nil {
		return false, fmt.Errorf("ERROR_READING_NAMESPACES: %w", err)
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

func persistencePoliciesToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%d-", m["bookkeeper_ensemble"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["bookkeeper_write_quorum"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["bookkeeper_ack_quorum"].(int)))
	buf.WriteString(fmt.Sprintf("%f-", m["managed_ledger_max_mark_delete_rate"].(float64)))

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

func unmarshalPersistencePolicies(v *schema.Set) *utils.PersistencePolicies {
	var persPolicies utils.PersistencePolicies

	for _, policy := range v.List() {
		data := policy.(map[string]interface{})

		persPolicies.BookkeeperEnsemble = data["bookkeeper_ensemble"].(int)
		persPolicies.BookkeeperWriteQuorum = data["bookkeeper_write_quorum"].(int)
		persPolicies.BookkeeperAckQuorum = data["bookkeeper_ack_quorum"].(int)
		persPolicies.ManagedLedgerMaxMarkDeleteRate = data["managed_ledger_max_mark_delete_rate"].(float64)
	}

	return &persPolicies
}
