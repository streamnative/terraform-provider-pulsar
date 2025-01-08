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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/streamnative/terraform-provider-pulsar/hashcode"
	"github.com/streamnative/terraform-provider-pulsar/types"
)

func resourcePulsarNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarNamespaceCreate,
		ReadContext:   resourcePulsarNamespaceRead,
		UpdateContext: resourcePulsarNamespaceUpdate,
		DeleteContext: resourcePulsarNamespaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				ns, err := utils.GetNamespaceName(d.Id())
				if err != nil {
					return nil, fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
				}
				nsParts := strings.Split(ns.String(), "/")
				_ = d.Set("tenant", nsParts[0])
				_ = d.Set("namespace", nsParts[1])

				diags := resourcePulsarNamespaceRead(ctx, d, meta)
				if diags.HasError() {
					return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
				}
				return []*schema.ResourceData{d}, nil
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
				Computed: true,
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
			"subscription_dispatch_rate": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: descriptions["subscription_dispatch_rate"],
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
				Elem:     schemaBacklogQuotaSubset(),
				Set:      hashBacklogQuotaSubset(),
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
							ValidateFunc: validateGtEq0,
						},
						"max_consumers_per_topic": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateGtEq0,
						},
						"max_producers_per_topic": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateGtEq0,
						},
						"message_ttl_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateGtEq0,
						},
						"replication_clusters": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								// If the field is not set in config, suppress the diff
								_, exists := d.GetOk("namespace_config.0.replication_clusters")
								return !exists
							},
						},
						"schema_validation_enforce": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"schema_compatibility_strategy": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateNotBlank,
							Default:      "Full",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								// If the field is not set in config, suppress the diff
								_, exists := d.GetOk("namespace_config.0.schema_compatibility_strategy")
								return !exists
							},
						},
						"is_allow_auto_update_schema": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"offload_threshold_size_in_mb": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateGtEq0,
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
			"permission_grant": {
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 0,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
						"actions": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateAuthAction,
							},
						},
					},
				},
			},
			"topic_auto_creation": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enable": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"type": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validatePartitionedTopicType,
							Default:      "non-partitioned",
						},
						"partitions": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				Set: topicAutoCreationPoliciesToHash,
			},
		},
	}
}

func resourcePulsarNamespaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)

	ns, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	if err := client.CreateNamespace(ns.String()); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_NAMESPACE: %w", err))
	}

	if err := resourcePulsarNamespaceUpdate(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_NAMESPACE_CONFIG: %v", err))
	}

	return nil
}

func resourcePulsarNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	ns, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	if nss, err := client.GetNamespaces(tenant); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetNamespaces: %w", err))
	} else if !contains(nss, ns.String()) {
		d.SetId("")
		return nil
	}

	d.SetId(ns.String())

	_ = d.Set("namespace", namespace)
	_ = d.Set("tenant", tenant)

	if namespaceConfig, ok := d.GetOk("namespace_config"); ok && namespaceConfig.(*schema.Set).Len() > 0 {
		configData := namespaceConfig.(*schema.Set).List()[0].(map[string]interface{})
		data := make(map[string]interface{})

		if _, ok := configData["anti_affinity"]; ok {
			afgrp, err := client.GetNamespaceAntiAffinityGroup(ns.String())
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetNamespaceAntiAffinityGroup: %w", err))
			}
			data["anti_affinity"] = strings.Trim(strings.TrimSpace(afgrp), "\"")
		}

		if _, ok := configData["max_consumers_per_subscription"]; ok {
			maxConsPerSub, err := client.GetMaxConsumersPerSubscription(*ns)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxConsumersPerSubscription: %w", err))
			}
			data["max_consumers_per_subscription"] = maxConsPerSub
		}

		if _, ok := configData["max_consumers_per_topic"]; ok {
			maxConsPerTopic, err := client.GetMaxConsumersPerTopic(*ns)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxConsumersPerTopic: %w", err))
			}
			data["max_consumers_per_topic"] = maxConsPerTopic
		}

		if _, ok := configData["max_producers_per_topic"]; ok {
			maxProdPerTopic, err := client.GetMaxProducersPerTopic(*ns)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxProducersPerTopic: %w", err))
			}
			data["max_producers_per_topic"] = maxProdPerTopic
		}

		if _, ok := configData["message_ttl_seconds"]; ok {
			messageTTL, err := client.GetNamespaceMessageTTL(ns.String())
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetNamespaceMessageTTL: %w", err))
			}
			data["message_ttl_seconds"] = messageTTL
		}

		if _, ok := configData["schema_validation_enforce"]; ok {
			schemaValidationEnforce, err := client.GetSchemaValidationEnforced(*ns)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSchemaValidationEnforced: %w", err))
			}
			data["schema_validation_enforce"] = schemaValidationEnforce
		}

		if _, ok := configData["schema_compatibility_strategy"]; ok {
			schemaCompatibilityStrategy, err := client.GetSchemaAutoUpdateCompatibilityStrategy(*ns)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSchemaAutoUpdateCompatibilityStrategy: %w", err))
			}
			data["schema_compatibility_strategy"] = schemaCompatibilityStrategy.String()
		}

		if _, ok := configData["replication_clusters"]; ok {
			replClustersRaw, err := client.GetNamespaceReplicationClusters(ns.String())
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxProducersPerTopic: %w", err))
			}

			// Only set replication_clusters if it was explicitly configured
			if len(replClustersRaw) > 0 {
				replClusters := make([]interface{}, len(replClustersRaw))
				for i, cl := range replClustersRaw {
					replClusters[i] = cl
				}
				data["replication_clusters"] = replClusters
			}
		}

		if _, ok := configData["is_allow_auto_update_schema"]; ok {
			isAllowAutoUpdateSchema, err := client.GetIsAllowAutoUpdateSchema(*ns)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetIsAllowAutoUpdateSchema: %w", err))
			}
			data["is_allow_auto_update_schema"] = isAllowAutoUpdateSchema
		}

		if _, ok := configData["offload_threshold_size_in_mb"]; ok {
			offloadTresholdSizeInMb, err := client.GetOffloadThreshold(*ns)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetOffloadThreshold: %w", err))
			}
			data["offload_threshold_size_in_mb"] = int(offloadTresholdSizeInMb)
		}

		_ = d.Set("namespace_config", schema.NewSet(namespaceConfigToHash, []interface{}{data}))
	}

	if persPoliciesCfg, ok := d.GetOk("persistence_policies"); ok && persPoliciesCfg.(*schema.Set).Len() > 0 {
		persistence, err := client.GetPersistence(ns.String())
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetPersistence: %w", err))
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
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetRetention: %w", err))
		}

		_ = d.Set("retention_policies", schema.NewSet(retentionPoliciesToHash, []interface{}{
			map[string]interface{}{
				"retention_minutes":    fmt.Sprint(ret.RetentionTimeInMinutes),
				"retention_size_in_mb": fmt.Sprint(ret.RetentionSizeInMB),
			},
		}))
	}

	if backlogQuotaCfg, ok := d.GetOk("backlog_quota"); ok && backlogQuotaCfg.(*schema.Set).Len() > 0 {
		qt, err := client.GetBacklogQuotaMap(ns.String())
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetBacklogQuotaMap: %w", err))
		}

		var backlogQuotas []interface{}
		for backlogQuotaType, data := range qt {
			backlogQuotas = append(backlogQuotas, map[string]interface{}{
				"limit_bytes":   strconv.FormatInt(data.LimitSize, 10),
				"limit_seconds": strconv.FormatInt(data.LimitTime, 10),
				"policy":        string(data.Policy),
				"type":          string(backlogQuotaType),
			})
		}

		_ = d.Set("backlog_quota", schema.NewSet(hashBacklogQuotaSubset(), backlogQuotas))
	}

	if dispatchRateCfg, ok := d.GetOk("dispatch_rate"); ok && dispatchRateCfg.(*schema.Set).Len() > 0 {
		dr, err := client.GetDispatchRate(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetDispatchRate: %w", err))
		}

		_ = d.Set("dispatch_rate", schema.NewSet(dispatchRateToHash, []interface{}{
			map[string]interface{}{
				"dispatch_msg_throttling_rate":  dr.DispatchThrottlingRateInMsg,
				"rate_period_seconds":           dr.RatePeriodInSecond,
				"dispatch_byte_throttling_rate": int(dr.DispatchThrottlingRateInByte),
			},
		}))
	}

	if subscriptionDispatchRateCfg, ok := d.GetOk("subscription_dispatch_rate"); ok && subscriptionDispatchRateCfg.(*schema.Set).Len() > 0 { //nolint:lll
		sdr, err := client.GetSubscriptionDispatchRate(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSubscriptionDispatchRate: %w", err))
		}

		_ = d.Set("subscription_dispatch_rate", schema.NewSet(dispatchRateToHash, []interface{}{
			map[string]interface{}{
				"dispatch_msg_throttling_rate":  sdr.DispatchThrottlingRateInMsg,
				"rate_period_seconds":           sdr.RatePeriodInSecond,
				"dispatch_byte_throttling_rate": int(sdr.DispatchThrottlingRateInByte),
			},
		}))
	}

	if permissionGrantCfg, ok := d.GetOk("permission_grant"); ok && len(permissionGrantCfg.(*schema.Set).List()) > 0 {
		grants, err := client.GetNamespacePermissions(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetNamespacePermissions: %w", err))
		}

		setPermissionGrant(d, grants)
	}

	if topicAutoCreation, ok := d.GetOk("topic_auto_creation"); ok && topicAutoCreation.(*schema.Set).Len() > 0 {
		autoCreation, err := client.GetTopicAutoCreation(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetTopicAutoCreation: %w", err))
		}

		data := map[string]interface{}{
			"enable": autoCreation.Allow,
			"type":   autoCreation.Type.String(),
		}
		if autoCreation.Partitions != nil {
			data["partitions"] = *autoCreation.Partitions
		}

		_ = d.Set("topic_auto_creation", schema.NewSet(topicAutoCreationPoliciesToHash, []interface{}{data}))
	}

	return nil
}

func resourcePulsarNamespaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)
	enableDeduplication, deduplicationDefined := d.GetOk("enable_deduplication")
	namespaceConfig := d.Get("namespace_config").(*schema.Set)
	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
	backlogQuotaConfig := d.Get("backlog_quota").(*schema.Set)
	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)
	subscriptionDispatchRateConfig := d.Get("subscription_dispatch_rate").(*schema.Set)
	persistencePoliciesConfig := d.Get("persistence_policies").(*schema.Set)
	permissionGrantConfig := d.Get("permission_grant").(*schema.Set)
	topicAutoCreation := d.Get("topic_auto_creation").(*schema.Set)

	nsName, err := utils.GetNameSpaceName(tenant, namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
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

		if nsCfg.MessageTTLInSeconds >= 0 {
			if err = client.SetNamespaceMessageTTL(nsName.String(), nsCfg.MessageTTLInSeconds); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceMessageTTL: %w", err))
			}
		}

		if nsCfg.OffloadThresholdSizeInMb >= 0 {
			if err = client.SetOffloadThreshold(*nsName, int64(nsCfg.OffloadThresholdSizeInMb)); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetOffloadThreshold: %w", err))
			}
		}

		if err = client.SetSchemaValidationEnforced(*nsName, nsCfg.SchemaValidationEnforce); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetSchemaValidationEnforced: %w", err))
		}

		if len(nsCfg.SchemaCompatibilityStrategy) > 0 {
			strategy, err := utils.ParseSchemaAutoUpdateCompatibilityStrategy(nsCfg.SchemaCompatibilityStrategy)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetSchemaCompatibilityStrategy: %w", err))
			} else if err = client.SetSchemaAutoUpdateCompatibilityStrategy(*nsName, strategy); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetSchemaCompatibilityStrategy: %w", err))
			}
		}
		if err = client.SetIsAllowAutoUpdateSchema(*nsName, nsCfg.IsAllowAutoUpdateSchema); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetIsAllowAutoUpdateSchema: %w", err))
		}
	}

	if retentionPoliciesConfig.Len() > 0 {
		retentionPolicies := unmarshalRetentionPolicies(retentionPoliciesConfig)
		if err = client.SetRetention(nsName.String(), *retentionPolicies); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetRetention: %w", err))
		}
	}

	if backlogQuotaConfig.Len() > 0 {
		backlogQuotas, err := unmarshalBacklogQuota(backlogQuotaConfig)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("unmarshalBacklogQuota: %w", err))
		} else {
			for _, item := range backlogQuotas {
				err = client.SetBacklogQuota(nsName.String(), item.BacklogQuota, item.backlogQuotaType)
				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("SetBacklogQuota: %w", err))
				}
			}
		}
	}

	if dispatchRateConfig.Len() > 0 {
		dispatchRate := unmarshalDispatchRate(dispatchRateConfig)
		if err = client.SetDispatchRate(*nsName, *dispatchRate); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetDispatchRate: %w", err))
		}
	}

	if subscriptionDispatchRateConfig.Len() > 0 {
		subscriptionDispatchRate := unmarshalDispatchRate(subscriptionDispatchRateConfig)
		if err = client.SetSubscriptionDispatchRate(*nsName, *subscriptionDispatchRate); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetSubscriptionDispatchRate: %w", err))
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

	if d.HasChange("permission_grant") {
		permissionGrants, err := unmarshalPermissionGrants(permissionGrantConfig)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("unmarshalPermissionGrants: %w", err))
		} else {
			for _, grant := range permissionGrants {
				if err = client.GrantNamespacePermission(*nsName, grant.Role, grant.Actions); err != nil {
					errs = multierror.Append(errs, fmt.Errorf("GrantNamespacePermission: %w", err))
				}
			}

			// Revoke permissions for roles removed from the set
			oldPermissionGrants, _ := d.GetChange("permission_grant")
			for _, oldGrant := range oldPermissionGrants.(*schema.Set).List() {
				oldRole := oldGrant.(map[string]interface{})["role"].(string)
				found := false
				for _, newGrant := range permissionGrants {
					if newGrant.Role == oldRole {
						found = true
						break
					}
				}
				if !found {
					if err = client.RevokeNamespacePermission(*nsName, oldRole); err != nil {
						errs = multierror.Append(errs, fmt.Errorf("RevokeNamespacePermission: %w", err))
					}
				}
			}
		}
	}

	if topicAutoCreation.Len() > 0 {
		topicAutoCreationPolicy, err := unmarshalTopicAutoCreation(topicAutoCreation)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetTopicAutoCreation: %w", err))
		} else {
			if err = client.SetTopicAutoCreation(*nsName, *topicAutoCreationPolicy); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetTopicAutoCreation: %w", err))
			}
		}
	} else { // remove the topicAutoCreation
		if err = client.RemoveTopicAutoCreation(*nsName); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("RemoveTopicAutoCreation: %w", err))
		}
	}

	if errs != nil {
		return diag.FromErr(fmt.Errorf("ERROR_UPDATE_NAMESPACE_CONFIG: %w", errs))
	}

	d.SetId(nsName.String())
	return resourcePulsarNamespaceRead(ctx, d, meta)
}

func resourcePulsarNamespaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)

	ns := fmt.Sprintf("%s/%s", tenant, namespace)

	if err := client.DeleteNamespace(ns); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_DELETE_NAMESPACE: %w", err))
	}

	_ = d.Set("namespace", "")
	_ = d.Set("tenant", "")
	_ = d.Set("enable_deduplication", nil)
	_ = d.Set("namespace_config", nil)
	_ = d.Set("retention_policies", nil)
	_ = d.Set("backlog_quota", nil)
	_ = d.Set("dispatch_rate", nil)
	_ = d.Set("subscription_dispatch_rate", nil)
	_ = d.Set("persistence_policies", nil)
	_ = d.Set("permission_grant", nil)
	_ = d.Set("topic_auto_creation", nil)

	return nil
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
	buf.WriteString(fmt.Sprintf("%d-", m["message_ttl_seconds"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["replication_clusters"].([]interface{})))
	buf.WriteString(fmt.Sprintf("%t-", m["schema_validation_enforce"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["schema_compatibility_strategy"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["offload_threshold_size_in_mb"].(int)))

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

func topicAutoCreationPoliciesToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%t-", m["enable"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))
	if m["partitions"] != nil {
		buf.WriteString(fmt.Sprintf("%d-", m["partitions"].(int)))
	}

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
		nsConfig.MessageTTLInSeconds = data["message_ttl_seconds"].(int)
		nsConfig.AntiAffinity = data["anti_affinity"].(string)
		nsConfig.SchemaValidationEnforce = data["schema_validation_enforce"].(bool)
		nsConfig.SchemaCompatibilityStrategy = data["schema_compatibility_strategy"].(string)
		nsConfig.IsAllowAutoUpdateSchema = data["is_allow_auto_update_schema"].(bool)
		nsConfig.OffloadThresholdSizeInMb = data["offload_threshold_size_in_mb"].(int)
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

func unmarshalTopicAutoCreation(v *schema.Set) (*utils.TopicAutoCreationConfig, error) {
	var topicAutoCreation utils.TopicAutoCreationConfig

	for _, policy := range v.List() {
		data := policy.(map[string]interface{})

		topicAutoCreation.Allow = data["enable"].(bool)
		topicAutoCreation.Type = utils.TopicType(data["type"].(string))
		if topicAutoCreation.Type == utils.Partitioned {
			partitions := data["partitions"].(int)
			if partitions <= 0 {
				return nil, fmt.Errorf("ERROR_PARSE_TOPIC_AUTO_CREATION: partitions must be greater than 0")
			}
			topicAutoCreation.Partitions = &partitions
		} else if topicAutoCreation.Type != utils.NonPartitioned {
			return nil, fmt.Errorf("ERROR_PARSE_TOPIC_AUTO_CREATION: unknown topic type %s", topicAutoCreation.Type)
		}
	}

	return &topicAutoCreation, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
