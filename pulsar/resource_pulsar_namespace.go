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
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				ns, err := utils.GetNamespaceName(d.Id())
				if err != nil {
					return nil, fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
				}
				nsParts := strings.Split(ns.String(), "/")
				_ = d.Set("tenant", nsParts[0])
				_ = d.Set("namespace", nsParts[1])

				err = resourcePulsarNamespaceRead(d, meta)
				return []*schema.ResourceData{d}, err
			},
		},
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
			"backlog_quota": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"limit_bytes": {
							Type:     schema.TypeString,
							Required: true,
						},
						"policy": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: backlogQuotaToHash,
			},
			"namespace_config": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: descriptions["namespace_config"],
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"anti_affinity": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateNotBlank,
						},
						"max_consumers_per_subscription": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
						},
						"max_consumers_per_topic": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
						},
						"max_producers_per_topic": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
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

	ns, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
	}

	if err := client.CreateNamespace(ns.String()); err != nil {
		return fmt.Errorf("ERROR_CREATE_NAMESPACE: %w", err)
	}

	if err = resourcePulsarNamespaceUpdate(d, meta); err != nil {
		return fmt.Errorf("ERROR_CREATE_NAMESPACE_CONFIG: %w", err)
	}

	return nil
}

func resourcePulsarNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	ns, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
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

	if persPoliciesCfg, ok := d.GetOk("persistence_policies"); ok && persPoliciesCfg.(*schema.Set).Len() > 0 {
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

	if retPoliciesCfg, ok := d.GetOk("retention_policies"); ok && retPoliciesCfg.(*schema.Set).Len() > 0 {
		ret, err := client.GetRetention(ns.String())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetRetention: %w", err)
		}

		_ = d.Set("persistence_policies", schema.NewSet(retentionPoliciesToHash, []interface{}{
			map[string]interface{}{
				"retention_minutes":    string(ret.RetentionTimeInMinutes),
				"retention_size_in_mb": string(ret.RetentionSizeInMB),
			},
		}))
	}

	if backlogQuotaCfg, ok := d.GetOk("backlog_quota"); ok && backlogQuotaCfg.(*schema.Set).Len() > 0 {
		qt, err := client.GetBacklogQuotaMap(ns.String())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetBacklogQuotaMap: %w", err)
		}

		_ = d.Set("backlog_quota", schema.NewSet(backlogQuotaToHash, []interface{}{
			map[string]interface{}{
				"limit_bytes": strconv.FormatInt(qt[utils.DestinationStorage].Limit, 10),
				"policy":      string(qt[utils.DestinationStorage].Policy),
			},
		}))
	}

	if dispatchRateCfg, ok := d.GetOk("dispatch_rate"); ok && dispatchRateCfg.(*schema.Set).Len() > 0 {
		dr, err := client.GetDispatchRate(*ns)
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: GetDispatchRate: %w", err)
		}

		_ = d.Set("persistence_policies", schema.NewSet(dispatchRateToHash, []interface{}{
			map[string]interface{}{
				"dispatch_msg_throttling_rate":  dr.DispatchThrottlingRateInMsg,
				"rate_period_seconds":           dr.RatePeriodInSecond,
				"dispatch_byte_throttling_rate": int(dr.DispatchThrottlingRateInByte),
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
	backlogQuotaConfig := d.Get("backlog_quota").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)
	persistencePoliciesConfig := d.Get("persistence_policies").(*schema.Set)

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
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

	if backlogQuotaConfig.Len() > 0 {
		backlogQuota, err := unmarshalBacklogQuota(backlogQuotaConfig)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("unmarshalBacklogQuota: %w", err))
		} else if err = client.SetBacklogQuota(nsName.String(), *backlogQuota); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetBacklogQuota: %w", err))
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
	_ = d.Set("backlog_quota", nil)
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
		return false, fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
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

func backlogQuotaToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["limit_bytes"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["policy"].(string)))

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

func unmarshalBacklogQuota(v *schema.Set) (*utils.BacklogQuota, error) {
	var bklQuota utils.BacklogQuota

	for _, quota := range v.List() {
		data := quota.(map[string]interface{})
		policyStr := data["policy"].(string)
		limitStr := data["limit_bytes"].(string)

		var policy utils.RetentionPolicy
		switch policyStr {
		case "producer_request_hold":
			policy = utils.ProducerRequestHold
		case "producer_exception":
			policy = utils.ProducerException
		case "consumer_backlog_eviction":
			policy = utils.ConsumerBacklogEviction
		default:
			return &bklQuota, fmt.Errorf("ERROR_INVALID_BACKLOG_QUOTA_POLICY: %v", policyStr)
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return &bklQuota, fmt.Errorf("ERROR_PARSE_BACKLOG_QUOTA_LIMIT: %w", err)
		}

		// zero values are fine, even if the ASCII to Int fails
		bklQuota.Limit = int64(limit)
		bklQuota.Policy = policy
	}

	return &bklQuota, nil
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
