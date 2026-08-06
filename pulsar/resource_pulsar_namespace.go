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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/streamnative/terraform-provider-pulsar/hashcode"
	"github.com/streamnative/terraform-provider-pulsar/types"
)

type namespaceReadMode uint8

const (
	namespaceReadRefresh namespaceReadMode = iota
	namespaceReadImport

	backlogQuotaManagedTypesStateAttr = "_backlog_quota_managed_types"
)

type backlogQuotaConfigPresence uint8

const (
	backlogQuotaConfigUnknown backlogQuotaConfigPresence = iota
	backlogQuotaConfigOmitted
	backlogQuotaConfigExplicit
)

func resourcePulsarNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarNamespaceCreate,
		ReadContext:   resourcePulsarNamespaceRead,
		UpdateContext: resourcePulsarNamespaceUpdate,
		DeleteContext: resourcePulsarNamespaceDelete,
		CustomizeDiff: resourcePulsarNamespaceCustomizeDiff,
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    pulsarNamespaceStateTypeV0(),
				Upgrade: resourcePulsarNamespaceStateUpgradeV0,
				Version: 0,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				importID := d.Id()
				ns, err := utils.GetNamespaceName(importID)
				if err != nil {
					return nil, fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
				}
				nsParts := strings.Split(ns.String(), "/")
				_ = d.Set("tenant", nsParts[0])
				_ = d.Set("namespace", nsParts[1])

				diags := resourcePulsarNamespaceReadWithMode(d, meta, namespaceReadImport)
				if diags.HasError() {
					return nil, fmt.Errorf("import %q: %s", importID, diags[0].Summary)
				}
				if d.Id() == "" {
					return nil, fmt.Errorf(
						"import %q: namespace not found in Pulsar; verify the namespace exists and the identifier is correct",
						importID,
					)
				}
				if err := d.Set(
					backlogQuotaManagedTypesStateAttr,
					[]interface{}{},
				); err != nil {
					return nil, fmt.Errorf("import %q: set backlog quota ownership state: %w", importID, err)
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
			},
			"dispatch_rate": {
				Type: schema.TypeSet,
				// Optional+Computed: the namespace-level dispatch rate is read back unconditionally, so
				// `terraform import` captures it and out-of-band changes are detected. Omitting the block
				// means "not managed here" rather than "must not exist", so a rate configured outside
				// Terraform is recorded in state without producing drift. See resourcePulsarNamespaceRead.
				Optional:    true,
				Computed:    true,
				Description: descriptions["namespace_dispatch_rate"],
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
				Type: schema.TypeSet,
				// Optional+Computed, see "dispatch_rate".
				Optional:    true,
				Computed:    true,
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
			"inactive_topic": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enable_delete_while_inactive": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"max_inactive_duration": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateInactiveTopicDuration,
						},
						"delete_mode": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validiateDeleteMode,
							Description:  "`delete_when_no_subscriptions` or `delete_when_subscriptions_caught_up`",
						},
					},
				},
				Set: inactiveTopicPoliciesToHash,
			},
			"backlog_quota": {
				Type: schema.TypeSet,
				// Optional+Computed, see "dispatch_rate". Only the quota types already tracked in state
				// are refreshed, so a quota type set outside Terraform is never adopted mid-life; see
				// setBacklogQuotaFiltered.
				Optional:    true,
				Computed:    true,
				Description: descriptions["backlog_quota"],
				Elem:        schemaBacklogQuotaSubset(),
				Set:         hashBacklogQuotaSubset(),
			},
			backlogQuotaManagedTypesStateAttr: {
				Type:     schema.TypeSet,
				Computed: true,
				Description: "Internal state tracking backlog quota types explicitly configured " +
					"when Terraform last applied a resource change.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"namespace_config": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: descriptions["namespace_config"],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"anti_affinity": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateNotBlank,
						},
						"is_allow_auto_update_schema": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"max_consumers_per_subscription": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
							Description: "Max consumers per subscription. 0 = unlimited, >0 = specific limit. " +
								"Omit to use broker defaults.",
						},
						"max_consumers_per_topic": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
							Description: "Max consumers per topic. 0 = unlimited, >0 = specific limit. " +
								"Omit to use broker defaults.",
						},
						"max_producers_per_topic": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
							Description: "Max producers per topic. 0 = unlimited, >0 = specific limit. " +
								"Omit to use broker defaults.",
						},
						"message_ttl_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
							Description: "Message TTL in seconds. 0 = never expire, >0 = expire after N seconds. " +
								"Omit to use broker defaults.",
						},
						"offload_threshold_size_in_mb": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
						},
						"replication_clusters": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"schema_compatibility_strategy": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateNotBlank,
							Description:  "Schema compatibility strategy. Managed only when explicitly set. Use Undefined to remove it.",
						},
						"schema_auto_update_compatibility_strategy": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateNotBlank,
							Description:  "Schema auto-update compatibility strategy. Managed only when explicitly set.",
						},
						"schema_validation_enforce": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"subscription_expiration_time_minutes": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validateGtEq0,
							Description: "Subscription expiration time in minutes. 0 = never expire, " +
								">0 = expire after N minutes. Omit to use broker defaults.",
						},
					},
				},
			},
			"persistence_policies": {
				Type: schema.TypeSet,
				// Optional+Computed, see "dispatch_rate".
				Optional:    true,
				Computed:    true,
				Description: descriptions["persistence_policies"],
				MaxItems:    1,
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
				Description: `Manages permissions within this namespace. **Warning:** Do not use this for roles that are ` +
					`already managed by the standalone pulsar_permission_grant resource, as it will cause conflicts.`,
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

// pulsarNamespaceStateTypeV0 is the frozen schema-version 0 state shape.
// It covers v0.11 state, where ownership metadata is absent, and v0.12.0-rc.3
// state, where it may be populated. Keep it independent from the current schema
// so future changes cannot break decoding of legacy flatmap states.
func pulsarNamespaceStateTypeV0() cty.Type {
	dispatchRateType := cty.Set(cty.Object(map[string]cty.Type{
		"dispatch_byte_throttling_rate": cty.Number,
		"dispatch_msg_throttling_rate":  cty.Number,
		"rate_period_seconds":           cty.Number,
	}))

	return cty.Object(map[string]cty.Type{
		backlogQuotaManagedTypesStateAttr: cty.Set(cty.String),
		"backlog_quota": cty.Set(cty.Object(map[string]cty.Type{
			"limit_bytes":   cty.String,
			"limit_seconds": cty.String,
			"policy":        cty.String,
			"type":          cty.String,
		})),
		"dispatch_rate":        dispatchRateType,
		"enable_deduplication": cty.Bool,
		"id":                   cty.String,
		"inactive_topic": cty.Set(cty.Object(map[string]cty.Type{
			"delete_mode":                  cty.String,
			"enable_delete_while_inactive": cty.Bool,
			"max_inactive_duration":        cty.String,
		})),
		"namespace": cty.String,
		"namespace_config": cty.List(cty.Object(map[string]cty.Type{
			"anti_affinity":                             cty.String,
			"is_allow_auto_update_schema":               cty.Bool,
			"max_consumers_per_subscription":            cty.Number,
			"max_consumers_per_topic":                   cty.Number,
			"max_producers_per_topic":                   cty.Number,
			"message_ttl_seconds":                       cty.Number,
			"offload_threshold_size_in_mb":              cty.Number,
			"replication_clusters":                      cty.Set(cty.String),
			"schema_auto_update_compatibility_strategy": cty.String,
			"schema_compatibility_strategy":             cty.String,
			"schema_validation_enforce":                 cty.Bool,
			"subscription_expiration_time_minutes":      cty.Number,
		})),
		"permission_grant": cty.Set(cty.Object(map[string]cty.Type{
			"actions": cty.Set(cty.String),
			"role":    cty.String,
		})),
		"persistence_policies": cty.Set(cty.Object(map[string]cty.Type{
			"bookkeeper_ack_quorum":               cty.Number,
			"bookkeeper_ensemble":                 cty.Number,
			"bookkeeper_write_quorum":             cty.Number,
			"managed_ledger_max_mark_delete_rate": cty.Number,
		})),
		"retention_policies": cty.Set(cty.Object(map[string]cty.Type{
			"retention_minutes":    cty.String,
			"retention_size_in_mb": cty.String,
		})),
		"subscription_dispatch_rate": dispatchRateType,
		"tenant":                     cty.String,
		"topic_auto_creation": cty.Set(cty.Object(map[string]cty.Type{
			"enable":     cty.Bool,
			"partitions": cty.Number,
			"type":       cty.String,
		})),
	})
}

func resourcePulsarNamespaceStateUpgradeV0(
	_ context.Context,
	rawState map[string]interface{},
	_ interface{},
) (map[string]interface{}, error) {
	if rawState == nil {
		return nil, fmt.Errorf("pulsar_namespace state upgrade from version 0: state is nil")
	}

	if managedTypes, exists := rawState[backlogQuotaManagedTypesStateAttr]; !exists || managedTypes == nil {
		rawState[backlogQuotaManagedTypesStateAttr] = []interface{}{}
	}

	return rawState, nil
}

func resourcePulsarNamespaceCustomizeDiff(
	_ context.Context,
	diff *schema.ResourceDiff,
	_ interface{},
) error {
	oldValue, newValue := diff.GetChange("backlog_quota")
	oldQuotas, err := backlogQuotaSet(oldValue)
	if err != nil {
		return err
	}
	plannedQuotas, err := backlogQuotaSet(newValue)
	if err != nil {
		return err
	}

	planned, changed, err := backlogQuotaPlannedSetForOwnership(
		diff.GetRawConfig(),
		diff.GetRawState(),
		oldValue,
		newValue,
	)
	if err != nil {
		return err
	}
	if changed {
		if err := diff.SetNew("backlog_quota", planned); err != nil {
			return err
		}
		plannedQuotas = planned
	}

	_, configPresence, err := rawConfigBacklogQuotaTypes(diff.GetRawConfig())
	if err != nil {
		return err
	}
	if configPresence != backlogQuotaConfigExplicit {
		return nil
	}
	if diff.Id() != "" && oldQuotas.Equal(plannedQuotas) && !namespaceDiffHasOtherChanges(diff) {
		// Do not turn an otherwise-empty post-import plan into metadata-only
		// churn. Ownership remains conservative until a real apply occurs.
		return nil
	}

	return diff.SetNewComputed(backlogQuotaManagedTypesStateAttr)
}

func namespaceDiffHasOtherChanges(diff *schema.ResourceDiff) bool {
	for _, key := range diff.GetChangedKeysPrefix("") {
		root := strings.SplitN(key, ".", 2)[0]
		if root != "backlog_quota" && root != backlogQuotaManagedTypesStateAttr && root != "id" {
			return true
		}
	}
	return false
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

func resourcePulsarNamespaceRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourcePulsarNamespaceReadWithMode(d, meta, namespaceReadRefresh)
}

// resourcePulsarNamespaceReadWithMode refreshes a namespace into state. Import mode force-hydrates
// the Optional+Computed policy blocks because an import starts without prior state. Normal refresh
// only reads blocks already tracked in state/config, preserving the v0.11 least-privilege behavior
// for namespaces that do not manage these policies.
func resourcePulsarNamespaceReadWithMode(
	d *schema.ResourceData,
	meta interface{},
	mode namespaceReadMode,
) diag.Diagnostics {
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
	if d.GetRawConfig().IsNull() {
		if err := initializeBacklogQuotaManagedTypesState(d); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: SetBacklogQuotaOwnershipState: %w", err))
		}
	}

	if _, ok := d.GetOk("namespace_config"); ok {
		var namespaceConfig = make(map[string]interface{})
		afgrp, err := client.GetNamespaceAntiAffinityGroup(ns.String())
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetNamespaceAntiAffinityGroup: %w", err))
		} else {
			namespaceConfig["anti_affinity"] = strings.Trim(strings.TrimSpace(afgrp), "\"")
		}

		isAllowAutoUpdateSchema, err := client.GetIsAllowAutoUpdateSchema(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetIsAllowAutoUpdateSchema: %w", err))
		} else {
			namespaceConfig["is_allow_auto_update_schema"] = isAllowAutoUpdateSchema
		}

		policies, err := client.GetPolicies(ns.String())
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetPolicies: %w", err))
		}
		namespaceConfig["max_consumers_per_subscription"] = policyNullableIntToStateValue(
			policies.MaxConsumersPerSubscription,
		)
		namespaceConfig["max_consumers_per_topic"] = policyNullableIntToStateValue(policies.MaxConsumersPerTopic)
		namespaceConfig["max_producers_per_topic"] = policyNullableIntToStateValue(policies.MaxProducersPerTopic)
		namespaceConfig["message_ttl_seconds"] = policyNullableIntToStateValue(policies.MessageTTLInSeconds)

		offloadTresholdSizeInMb, err := client.GetOffloadThreshold(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetOffloadThreshold: %w", err))
		} else {
			namespaceConfig["offload_threshold_size_in_mb"] = int(offloadTresholdSizeInMb)
		}

		replClustersRaw, err := client.GetNamespaceReplicationClusters(ns.String())
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetMaxProducersPerTopic: %w", err))
		} else {
			replClustersInterface := make([]interface{}, len(replClustersRaw))
			for i, cl := range replClustersRaw {
				replClustersInterface[i] = cl
			}
			replClusters := schema.NewSet(schema.HashString, replClustersInterface)
			namespaceConfig["replication_clusters"] = replClusters
		}

		schemaValidationEnforce, err := client.GetSchemaValidationEnforced(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSchemaValidationEnforced: %w", err))
		} else {
			namespaceConfig["schema_validation_enforce"] = schemaValidationEnforce
		}

		hasSchemaAutoUpdateCompatibilityStrategy := namespaceConfigHasSchemaAutoUpdateCompatibilityStrategy(d)
		if hasSchemaAutoUpdateCompatibilityStrategy {
			schemaAutoUpdateCompatibilityStrategy, err := client.GetSchemaAutoUpdateCompatibilityStrategy(*ns)
			if err != nil {
				if !strings.Contains(err.Error(), "Invalid auth strategy") && !strings.Contains(err.Error(), "404") {
					return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSchemaAutoUpdateCompatibilityStrategy: %w", err))
				}
			} else if value, ok := namespaceSchemaAutoUpdateCompatibilityStrategyStateValue(
				schemaAutoUpdateCompatibilityStrategy,
				hasSchemaAutoUpdateCompatibilityStrategy,
			); ok {
				namespaceConfig["schema_auto_update_compatibility_strategy"] = value
			}
		}

		hasSchemaCompatibilityStrategy := namespaceConfigHasSchemaCompatibilityStrategy(d)
		if hasSchemaCompatibilityStrategy {
			schemaCompatibilityStrategy, err := client.GetSchemaCompatibilityStrategy(*ns)
			if err != nil {
				if !strings.Contains(err.Error(), "Invalid auth strategy") && !strings.Contains(err.Error(), "404") {
					return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSchemaCompatibilityStrategy: %w", err))
				}
			} else if value, ok := namespaceSchemaCompatibilityStrategyStateValue(
				schemaCompatibilityStrategy,
				hasSchemaCompatibilityStrategy,
			); ok {
				namespaceConfig["schema_compatibility_strategy"] = value
			}
		}

		subscriptionExpirationTimeMinutes, err := client.GetSubscriptionExpirationTime(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSubscriptionExpirationTime: %w", err))
		} else {
			namespaceConfig["subscription_expiration_time_minutes"] = subscriptionExpirationTimeMinutes
		}

		_ = d.Set("namespace_config", []interface{}{
			namespaceConfig,
		})
	}

	if shouldReadNamespacePolicyBlock(d, "persistence_policies", mode) {
		persistence, err := client.GetPersistence(ns.String())
		if err != nil {
			if isAdminNotFoundError(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetPersistence: %w", err))
		}

		persistenceState := emptySet(persistencePoliciesToHash)
		if isPersistenceConfigured(persistence) {
			persistenceState = schema.NewSet(persistencePoliciesToHash, []interface{}{
				map[string]interface{}{
					"bookkeeper_ensemble":                 persistence.BookkeeperEnsemble,
					"bookkeeper_write_quorum":             persistence.BookkeeperWriteQuorum,
					"bookkeeper_ack_quorum":               persistence.BookkeeperAckQuorum,
					"managed_ledger_max_mark_delete_rate": persistence.ManagedLedgerMaxMarkDeleteRate,
				},
			})
		}
		if err := d.Set("persistence_policies", persistenceState); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: SetPersistenceState: %w", err))
		}
	}

	if retPoliciesCfg, ok := d.GetOk("retention_policies"); ok && retPoliciesCfg.(*schema.Set).Len() > 0 {
		ret, err := client.GetRetention(ns.String())
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetRetention: %w", err))
		}

		if ret != nil {
			_ = d.Set("retention_policies", schema.NewSet(retentionPoliciesToHash, []interface{}{
				map[string]interface{}{
					"retention_minutes":    fmt.Sprint(ret.RetentionTimeInMinutes),
					"retention_size_in_mb": fmt.Sprint(ret.RetentionSizeInMB),
				},
			}))
		} else {
			_ = d.Set("retention_policies", schema.NewSet(retentionPoliciesToHash, []interface{}{}))
		}
	}

	if inactiveTopicCfg, ok := d.GetOk("inactive_topic"); ok && inactiveTopicCfg.(*schema.Set).Len() > 0 {
		inactiveTopicPolicies, err := client.GetInactiveTopicPolicies(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetInactiveTopicPolicies: %w", err))
		}

		deleteMode := "delete_when_no_subscriptions"
		if inactiveTopicPolicies.InactiveTopicDeleteMode != nil {
			deleteMode = inactiveTopicPolicies.InactiveTopicDeleteMode.String()
		}

		_ = d.Set("inactive_topic", schema.NewSet(inactiveTopicPoliciesToHash, []interface{}{
			map[string]interface{}{
				"enable_delete_while_inactive": inactiveTopicPolicies.DeleteWhileInactive,
				"max_inactive_duration":        fmt.Sprintf("%ds", inactiveTopicPolicies.MaxInactiveDurationSeconds),
				"delete_mode":                  deleteMode,
			},
		}))
	}

	if shouldReadNamespacePolicyBlock(d, "backlog_quota", mode) {
		qt, err := client.GetBacklogQuotaMap(ns.String())
		if err != nil {
			if isAdminNotFoundError(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetBacklogQuotaMap: %w", err))
		}
		if err := setBacklogQuotaFiltered(d, qt); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: SetBacklogQuotaState: %w", err))
		}
	}

	if shouldReadNamespacePolicyBlock(d, "dispatch_rate", mode) {
		dr, err := client.GetDispatchRate(*ns)
		if err != nil {
			if isAdminNotFoundError(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetDispatchRate: %w", err))
		}

		dispatchRateState := emptySet(dispatchRateToHash)
		if isDispatchRateConfigured(dr) {
			dispatchRateState = schema.NewSet(dispatchRateToHash, []interface{}{
				map[string]interface{}{
					"dispatch_msg_throttling_rate":  dr.DispatchThrottlingRateInMsg,
					"rate_period_seconds":           dr.RatePeriodInSecond,
					"dispatch_byte_throttling_rate": int(dr.DispatchThrottlingRateInByte),
				},
			})
		}
		if err := d.Set("dispatch_rate", dispatchRateState); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: SetDispatchRateState: %w", err))
		}
	}

	if shouldReadNamespacePolicyBlock(d, "subscription_dispatch_rate", mode) {
		sdr, err := client.GetSubscriptionDispatchRate(*ns)
		if err != nil {
			if isAdminNotFoundError(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetSubscriptionDispatchRate: %w", err))
		}

		subscriptionDispatchRateState := emptySet(dispatchRateToHash)
		if isDispatchRateConfigured(sdr) {
			subscriptionDispatchRateState = schema.NewSet(dispatchRateToHash, []interface{}{
				map[string]interface{}{
					"dispatch_msg_throttling_rate":  sdr.DispatchThrottlingRateInMsg,
					"rate_period_seconds":           sdr.RatePeriodInSecond,
					"dispatch_byte_throttling_rate": int(sdr.DispatchThrottlingRateInByte),
				},
			})
		}
		if err := d.Set("subscription_dispatch_rate", subscriptionDispatchRateState); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: SetSubscriptionDispatchRateState: %w", err))
		}
	}

	if permissionGrantCfg, ok := d.GetOk("permission_grant"); ok && len(permissionGrantCfg.(*schema.Set).List()) > 0 {
		grants, err := client.GetNamespacePermissions(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetNamespacePermissions: %w", err))
		}

		setPermissionGrantFiltered(d, grants)
	}

	if topicAutoCreation, ok := d.GetOk("topic_auto_creation"); ok && topicAutoCreation.(*schema.Set).Len() > 0 {
		autoCreation, err := client.GetTopicAutoCreation(*ns)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE: GetTopicAutoCreation: %w", err))
		}

		// A nil response means the namespace override was removed out-of-band; report it as drift
		// rather than dereferencing a nil pointer.
		if autoCreation == nil {
			_ = d.Set("topic_auto_creation", emptySet(topicAutoCreationPoliciesToHash))
		} else {
			data := map[string]interface{}{
				"enable": autoCreation.Allow,
				"type":   autoCreation.Type.String(),
			}
			if autoCreation.Partitions != nil {
				data["partitions"] = *autoCreation.Partitions
			}

			_ = d.Set("topic_auto_creation", schema.NewSet(topicAutoCreationPoliciesToHash, []interface{}{data}))
		}
	}

	return nil
}

func resourcePulsarNamespaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	preservePriorStateOnError := !d.IsNewResource()
	if preservePriorStateOnError {
		// The SDK otherwise builds error state from the proposed diff. Preserve
		// prior state so failed quota writes/removals and ownership updates are
		// retried instead of being hidden by an unapplied planned value.
		d.Partial(true)
	}

	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)
	enableDeduplication, deduplicationDefined := d.GetOk("enable_deduplication")
	namespaceConfig := d.Get("namespace_config").([]interface{})
	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
	inactiveTopicConfig := d.Get("inactive_topic").(*schema.Set)
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

	if len(namespaceConfig) > 0 {
		nsCfg := unmarshalNamespaceConfigList(namespaceConfig)

		if len(nsCfg.AntiAffinity) > 0 {
			if err = client.SetNamespaceAntiAffinityGroup(nsName.String(), nsCfg.AntiAffinity); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceAntiAffinityGroup: %w", err))
			}
		}

		if err = client.SetIsAllowAutoUpdateSchema(*nsName, nsCfg.IsAllowAutoUpdateSchema); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetIsAllowAutoUpdateSchema: %w", err))
		}

		if nsCfg.MaxConsumersPerTopic >= 0 {
			if err = client.SetMaxConsumersPerTopic(*nsName, nsCfg.MaxConsumersPerTopic); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerTopic: %w", err))
			}
		} else {
			if err = client.RemoveMaxConsumersPerTopic(*nsName); err != nil && !isIgnorableNotFoundError(err) {
				errs = multierror.Append(errs, fmt.Errorf("RemoveMaxConsumersPerTopic: %w", err))
			}
		}

		if nsCfg.MaxConsumersPerSubscription >= 0 {
			if err = client.SetMaxConsumersPerSubscription(*nsName, nsCfg.MaxConsumersPerSubscription); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxConsumersPerSubscription: %w", err))
			}
		} else {
			if err = client.RemoveMaxConsumersPerSubscription(*nsName); err != nil && !isIgnorableNotFoundError(err) {
				errs = multierror.Append(errs, fmt.Errorf("RemoveMaxConsumersPerSubscription: %w", err))
			}
		}

		if nsCfg.MaxProducersPerTopic >= 0 {
			if err = client.SetMaxProducersPerTopic(*nsName, nsCfg.MaxProducersPerTopic); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetMaxProducersPerTopic: %w", err))
			}
		} else {
			if err = client.RemoveMaxProducersPerTopic(*nsName); err != nil && !isIgnorableNotFoundError(err) {
				errs = multierror.Append(errs, fmt.Errorf("RemoveMaxProducersPerTopic: %w", err))
			}
		}

		if nsCfg.MessageTTLInSeconds >= 0 {
			if err = client.SetNamespaceMessageTTL(nsName.String(), nsCfg.MessageTTLInSeconds); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceMessageTTL: %w", err))
			}
		} else {
			if err = client.RemoveNamespaceMessageTTL(nsName.String()); err != nil && !isIgnorableNotFoundError(err) {
				errs = multierror.Append(errs, fmt.Errorf("RemoveNamespaceMessageTTL: %w", err))
			}
		}

		if nsCfg.OffloadThresholdSizeInMb >= 0 {
			if err = client.SetOffloadThreshold(*nsName, int64(nsCfg.OffloadThresholdSizeInMb)); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetOffloadThreshold: %w", err))
			}
		}

		if len(nsCfg.ReplicationClusters) > 0 {
			if err = client.SetNamespaceReplicationClusters(nsName.String(), nsCfg.ReplicationClusters); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetNamespaceReplicationClusters: %w", err))
			}
		}

		if namespaceConfigHasSchemaCompatibilityStrategy(d) && len(nsCfg.SchemaCompatibilityStrategy) > 0 {
			strategy, err := parseSchemaCompatibilityStrategy(nsCfg.SchemaCompatibilityStrategy)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetSchemaCompatibilityStrategy: %w", err))
			} else if err = client.SetSchemaCompatibilityStrategy(*nsName, strategy); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetSchemaCompatibilityStrategy: %w", err))
			}
		}

		if namespaceConfigHasSchemaAutoUpdateCompatibilityStrategy(d) &&
			len(nsCfg.SchemaAutoUpdateCompatibilityStrategy) > 0 {
			strategy, err := utils.ParseSchemaAutoUpdateCompatibilityStrategy(nsCfg.SchemaAutoUpdateCompatibilityStrategy)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetSchemaAutoUpdateCompatibilityStrategy: %w", err))
			} else if err = client.SetSchemaAutoUpdateCompatibilityStrategy(*nsName, strategy); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetSchemaAutoUpdateCompatibilityStrategy: %w", err))
			}
		}

		if err = client.SetSchemaValidationEnforced(*nsName, nsCfg.SchemaValidationEnforce); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetSchemaValidationEnforced: %w", err))
		}

		if nsCfg.SubscriptionExpirationTimeMinutes >= 0 {
			if err = client.SetSubscriptionExpirationTime(*nsName, nsCfg.SubscriptionExpirationTimeMinutes); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetSubscriptionExpirationTime: %w", err))
			}
		} else { // remove the subscription expiration time
			if err = client.RemoveSubscriptionExpirationTime(*nsName); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("RemoveSubscriptionExpirationTime: %w", err))
			}
		}
	}

	if d.HasChange("retention_policies") || retentionPoliciesConfig.Len() > 0 {
		if retentionPoliciesConfig.Len() > 0 {
			retentionPolicies := unmarshalRetentionPolicies(retentionPoliciesConfig)
			if err = client.SetRetention(nsName.String(), *retentionPolicies); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetRetention: %w", err))
			}
		} else {
			oldRetentionConfig, _ := d.GetChange("retention_policies")
			if oldCfg, ok := oldRetentionConfig.(*schema.Set); ok && oldCfg != nil && oldCfg.Len() > 0 {
				if err = client.RemoveRetention(nsName.String()); err != nil && !isIgnorableNotFoundError(err) {
					errs = multierror.Append(errs, fmt.Errorf("RemoveRetention: %w", err))
				}
			}
		}
	}

	if d.HasChange("inactive_topic") || inactiveTopicConfig.Len() > 0 {
		if inactiveTopicConfig.Len() > 0 {
			inactiveTopicPolicies, err := unmarshalInactiveTopicPolicies(inactiveTopicConfig)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("unmarshalInactiveTopicPolicies: %w", err))
			} else if err = client.SetInactiveTopicPolicies(*nsName, *inactiveTopicPolicies); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("SetInactiveTopicPolicies: %w", err))
			}
		} else {
			oldInactiveTopicConfig, _ := d.GetChange("inactive_topic")
			if hasInactiveTopicPoliciesConfigured(oldInactiveTopicConfig) {
				if err = client.RemoveInactiveTopicPolicies(*nsName); err != nil && !isIgnorableNotFoundError(err) {
					errs = multierror.Append(errs, fmt.Errorf("RemoveInactiveTopicPolicies: %w", err))
				}
			}
		}
	}

	if d.HasChange("backlog_quota") && backlogQuotaConfig.Len() > 0 {
		oldBacklogQuotaConfig, _ := d.GetChange("backlog_quota")
		oldBacklogQuotaSet, _ := oldBacklogQuotaConfig.(*schema.Set)
		writesSucceeded := true
		configuredTypes, configPresence, err := rawConfigBacklogQuotaTypes(d.GetRawConfig())
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("configuredBacklogQuotaTypes: %w", err))
			writesSucceeded = false
		} else {
			backlogQuotas, err := unmarshalBacklogQuota(backlogQuotaConfig)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("unmarshalBacklogQuota: %w", err))
				writesSucceeded = false
			} else {
				for _, item := range backlogQuotas {
					if configPresence == backlogQuotaConfigOmitted {
						continue
					}
					if configPresence == backlogQuotaConfigExplicit {
						if _, configured := configuredTypes[item.backlogQuotaType]; !configured {
							continue
						}
					}

					err = client.SetBacklogQuota(nsName.String(), item.BacklogQuota, item.backlogQuotaType)
					if err != nil {
						errs = multierror.Append(errs, fmt.Errorf("SetBacklogQuota: %w", err))
						writesSucceeded = false
					}
				}
			}
		}

		// Apply additions/replacements before destructive removals. A failed
		// write leaves old quotas intact so retry does not lose both policies.
		if writesSucceeded && configPresence == backlogQuotaConfigExplicit {
			removedQuotaTypes, err := removedManagedBacklogQuotaTypes(
				oldBacklogQuotaSet,
				backlogQuotaConfig,
				d.GetRawState(),
			)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("removedManagedBacklogQuotaTypes: %w", err))
			} else {
				policyClient := getNamespacePolicyClientFromMeta(meta)
				for _, quotaType := range removedQuotaTypes {
					if err = policyClient.RemoveBacklogQuotaByType(ctx, nsName.String(), quotaType); err != nil &&
						!isIgnorableNotFoundError(err) {
						errs = multierror.Append(errs, fmt.Errorf("RemoveBacklogQuota(%s): %w", quotaType, err))
					}
				}
			}
		}
	}

	if d.HasChange("dispatch_rate") && dispatchRateConfig.Len() > 0 {
		dispatchRate := unmarshalDispatchRate(dispatchRateConfig)
		if err = client.SetDispatchRate(*nsName, *dispatchRate); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetDispatchRate: %w", err))
		}
	}

	if d.HasChange("subscription_dispatch_rate") && subscriptionDispatchRateConfig.Len() > 0 {
		subscriptionDispatchRate := unmarshalDispatchRate(subscriptionDispatchRateConfig)
		if err = client.SetSubscriptionDispatchRate(*nsName, *subscriptionDispatchRate); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("SetSubscriptionDispatchRate: %w", err))
		}
	}

	if d.HasChange("persistence_policies") && persistencePoliciesConfig.Len() > 0 {
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
	} else {
		// Only remove the override when Terraform previously owned it. Removing unconditionally would
		// wipe an override this resource never managed — e.g. one set by the console or pulsar-admin —
		// on any unrelated namespace update.
		oldTopicAutoCreation, _ := d.GetChange("topic_auto_creation")
		if oldCfg, ok := oldTopicAutoCreation.(*schema.Set); ok && oldCfg.Len() > 0 {
			if err = client.RemoveTopicAutoCreation(*nsName); err != nil && !isIgnorableNotFoundError(err) {
				errs = multierror.Append(errs, fmt.Errorf("RemoveTopicAutoCreation: %w", err))
			}
		}
	}

	if errs != nil {
		return diag.FromErr(fmt.Errorf("ERROR_UPDATE_NAMESPACE_CONFIG: %w", errs))
	}
	if err := setBacklogQuotaManagedTypesState(d); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_UPDATE_NAMESPACE_CONFIG: SetBacklogQuotaOwnershipState: %w", err))
	}

	d.SetId(nsName.String())
	diags := resourcePulsarNamespaceRead(ctx, d, meta)
	if diags.HasError() {
		return diags
	}
	if preservePriorStateOnError {
		d.Partial(false)
	}
	return diags
}

func hasInactiveTopicPoliciesConfigured(data interface{}) bool {
	cfg, ok := data.(*schema.Set)
	return ok && cfg != nil && cfg.Len() > 0
}

func shouldReadNamespacePolicyBlock(d *schema.ResourceData, attr string, mode namespaceReadMode) bool {
	if mode == namespaceReadImport {
		return true
	}

	value, ok := d.GetOk(attr)
	if !ok {
		return false
	}
	configured, ok := value.(*schema.Set)
	return ok && configured.Len() > 0
}

// emptySet returns an empty set for the given hash function, used to clear an Optional+Computed
// block when the server reports the policy as unset.
func emptySet(f schema.SchemaSetFunc) *schema.Set {
	return schema.NewSet(f, []interface{}{})
}

// isDispatchRateConfigured reports whether a dispatch rate returned by the admin API represents an
// explicitly configured policy rather than an unset/zero-value default.
//
// Pulsar answers 404 for an unconfigured namespace dispatch rate, which the caller handles; this
// guards the other shape seen in the wild, where the endpoint answers 200 with an empty or null body
// and the client decodes it into a zero value. A configured rate always carries a rate period of at
// least one second, while -1 msg/byte is a legitimate "unlimited" value that must be preserved.
func isDispatchRateConfigured(rate utils.DispatchRate) bool {
	return rate.RatePeriodInSecond != 0
}

// isPersistenceConfigured reports whether a persistence policy is explicitly configured. The admin
// API returns a nil pointer for an unset policy (empty body) and, on some brokers, a non-nil
// zero-value struct for a literal JSON null; a real BookKeeper ensemble size is always >= 1, so an
// all-zero struct can only be a default/unset sentinel.
func isPersistenceConfigured(p *utils.PersistencePolicies) bool {
	return p != nil && p.BookkeeperEnsemble != 0
}

// setBacklogQuotaFiltered writes the namespace backlog quota map into state.
//
// backlog_quota is non-authoritative, like permission_grant: only quota types already tracked in
// state are refreshed, so a type configured out-of-band is not adopted or deleted. A fresh import
// has no tracked types and adopts every type returned by the broker.
func setBacklogQuotaFiltered(
	d *schema.ResourceData,
	quotas map[utils.BacklogQuotaType]utils.BacklogQuota,
) error {
	managedTypes := make(map[string]bool)
	for _, quota := range d.Get("backlog_quota").(*schema.Set).List() {
		managedTypes[quota.(map[string]interface{})["type"].(string)] = true
	}
	adoptAll := len(managedTypes) == 0

	backlogQuotas := []interface{}{}
	for backlogQuotaType, data := range quotas {
		if !adoptAll && !managedTypes[string(backlogQuotaType)] {
			continue
		}
		backlogQuotas = append(backlogQuotas, map[string]interface{}{
			"limit_bytes":   strconv.FormatInt(data.LimitSize, 10),
			"limit_seconds": strconv.FormatInt(data.LimitTime, 10),
			"policy":        string(data.Policy),
			"type":          string(backlogQuotaType),
		})
	}

	return d.Set("backlog_quota", schema.NewSet(hashBacklogQuotaSubset(), backlogQuotas))
}

func backlogQuotaPlannedSetForOwnership(
	rawConfig cty.Value,
	rawState cty.Value,
	oldValue interface{},
	newValue interface{},
) (*schema.Set, bool, error) {
	oldQuotas, err := backlogQuotaSet(oldValue)
	if err != nil {
		return nil, false, err
	}
	newQuotas, err := backlogQuotaSet(newValue)
	if err != nil {
		return nil, false, err
	}

	configuredTypes, configPresence, err := rawConfigBacklogQuotaTypes(rawConfig)
	if err != nil {
		return nil, false, err
	}
	if configPresence != backlogQuotaConfigExplicit {
		if oldQuotas.Equal(newQuotas) {
			return nil, false, nil
		}

		// Unknown configuration and an omitted Optional+Computed block must never
		// turn refreshed state into a destructive removal.
		return schema.CopySet(oldQuotas), true, nil
	}

	managedTypes, ownershipKnown, err := rawStateBacklogQuotaManagedTypes(rawState)
	if err != nil {
		return nil, false, err
	}
	if !ownershipKnown {
		// v0.11 and v0.12.0-rc.2 states predate ownership metadata. State
		// membership alone cannot prove ownership, so preserve every old type
		// that current configuration does not explicitly replace.
		managedTypes = make(map[utils.BacklogQuotaType]struct{})
	}

	planned := schema.CopySet(newQuotas)
	plannedTypes, err := backlogQuotaTypes(planned)
	if err != nil {
		return nil, false, err
	}

	for _, quota := range oldQuotas.List() {
		quotaType, err := backlogQuotaType(quota)
		if err != nil {
			return nil, false, err
		}
		if _, configured := configuredTypes[quotaType]; configured {
			continue
		}
		if _, managed := managedTypes[quotaType]; managed {
			continue
		}
		if _, exists := plannedTypes[quotaType]; exists {
			continue
		}

		planned.Add(quota)
		plannedTypes[quotaType] = struct{}{}
	}

	if planned.Equal(newQuotas) {
		return nil, false, nil
	}
	return planned, true, nil
}

func backlogQuotaSet(value interface{}) (*schema.Set, error) {
	if value == nil {
		return emptySet(hashBacklogQuotaSubset()), nil
	}

	quotas, ok := value.(*schema.Set)
	if !ok {
		return nil, fmt.Errorf("unexpected backlog quota set value %T", value)
	}
	if quotas == nil {
		return emptySet(hashBacklogQuotaSubset()), nil
	}
	return quotas, nil
}

func rawConfigBacklogQuotaTypes(
	rawConfig cty.Value,
) (map[utils.BacklogQuotaType]struct{}, backlogQuotaConfigPresence, error) {
	types := make(map[utils.BacklogQuotaType]struct{})
	if !rawConfig.IsKnown() || rawConfig.IsNull() {
		return types, backlogQuotaConfigUnknown, nil
	}
	if !rawConfig.Type().IsObjectType() || !rawConfig.Type().HasAttribute("backlog_quota") {
		return types, backlogQuotaConfigUnknown, nil
	}

	configured := rawConfig.GetAttr("backlog_quota")
	if !configured.IsKnown() {
		return types, backlogQuotaConfigUnknown, nil
	}
	if configured.IsNull() {
		return types, backlogQuotaConfigOmitted, nil
	}

	parsed, known, err := ctyBacklogQuotaTypes(configured)
	if err != nil {
		return nil, backlogQuotaConfigUnknown, err
	}
	if !known {
		return types, backlogQuotaConfigUnknown, nil
	}
	return parsed, backlogQuotaConfigExplicit, nil
}

func rawStateBacklogQuotaManagedTypes(
	rawState cty.Value,
) (map[utils.BacklogQuotaType]struct{}, bool, error) {
	types := make(map[utils.BacklogQuotaType]struct{})
	if !rawState.IsKnown() || rawState.IsNull() {
		return types, false, nil
	}
	if !rawState.Type().IsObjectType() || !rawState.Type().HasAttribute(backlogQuotaManagedTypesStateAttr) {
		return types, false, nil
	}

	managed := rawState.GetAttr(backlogQuotaManagedTypesStateAttr)
	if !managed.IsKnown() || managed.IsNull() {
		return types, false, nil
	}
	if !managed.Type().IsSetType() && !managed.Type().IsListType() && !managed.Type().IsTupleType() {
		return nil, false, fmt.Errorf(
			"unexpected %s state type %s",
			backlogQuotaManagedTypesStateAttr,
			managed.Type().FriendlyName(),
		)
	}

	iterator := managed.ElementIterator()
	for iterator.Next() {
		_, value := iterator.Element()
		if !value.IsKnown() || value.IsNull() {
			return types, false, nil
		}
		if value.Type() != cty.String {
			return nil, false, fmt.Errorf(
				"unexpected %s element type %s",
				backlogQuotaManagedTypesStateAttr,
				value.Type().FriendlyName(),
			)
		}

		quotaType, err := utils.ParseBacklogQuotaType(value.AsString())
		if err != nil {
			return nil, false, err
		}
		types[quotaType] = struct{}{}
	}
	return types, true, nil
}

func ctyBacklogQuotaTypes(
	quotas cty.Value,
) (map[utils.BacklogQuotaType]struct{}, bool, error) {
	types := make(map[utils.BacklogQuotaType]struct{})
	if !quotas.Type().IsSetType() && !quotas.Type().IsListType() && !quotas.Type().IsTupleType() {
		return nil, false, fmt.Errorf("unexpected backlog_quota config type %s", quotas.Type().FriendlyName())
	}

	iterator := quotas.ElementIterator()
	for iterator.Next() {
		_, value := iterator.Element()
		if !value.IsKnown() || value.IsNull() {
			return types, false, nil
		}
		if !value.Type().IsObjectType() || !value.Type().HasAttribute("type") {
			return nil, false, fmt.Errorf("unexpected backlog_quota element type %s", value.Type().FriendlyName())
		}

		quotaTypeValue := value.GetAttr("type")
		if !quotaTypeValue.IsKnown() || quotaTypeValue.IsNull() {
			return types, false, nil
		}
		if quotaTypeValue.Type() != cty.String {
			return nil, false, fmt.Errorf("unexpected backlog_quota.type value type %s", quotaTypeValue.Type().FriendlyName())
		}

		quotaType, err := utils.ParseBacklogQuotaType(quotaTypeValue.AsString())
		if err != nil {
			return nil, false, err
		}
		types[quotaType] = struct{}{}
	}
	return types, true, nil
}

func setBacklogQuotaManagedTypesState(d *schema.ResourceData) error {
	// A successful resource apply is the persistence point for explicit HCL
	// ownership. The backlog quota itself need not change: configuration
	// declaration is sufficient ownership evidence, while imported-only types
	// never appear in raw config and remain unmanaged.
	configuredTypes, configPresence, err := rawConfigBacklogQuotaTypes(d.GetRawConfig())
	if err != nil {
		return err
	}

	switch configPresence {
	case backlogQuotaConfigExplicit:
		return d.Set(backlogQuotaManagedTypesStateAttr, backlogQuotaTypeStateValues(configuredTypes))
	case backlogQuotaConfigOmitted:
		if d.IsNewResource() {
			return d.Set(backlogQuotaManagedTypesStateAttr, []interface{}{})
		}
		return nil
	case backlogQuotaConfigUnknown:
		if !d.IsNewResource() {
			return nil
		}

		// Creation cannot contain import-hydrated quota types. Falling back to
		// the planned set therefore records only values this provider wrote.
		configuredTypes, err := backlogQuotaTypes(d.Get("backlog_quota").(*schema.Set))
		if err != nil {
			return err
		}
		return d.Set(backlogQuotaManagedTypesStateAttr, backlogQuotaTypeStateValues(configuredTypes))
	default:
		return nil
	}
}

func initializeBacklogQuotaManagedTypesState(d *schema.ResourceData) error {
	_, ownershipKnown, err := rawStateBacklogQuotaManagedTypes(d.GetRawState())
	if err != nil {
		return err
	}
	if ownershipKnown {
		return nil
	}

	// Legacy v0.11 and v0.12.0-rc.2 state has no provenance. Initialize it
	// conservatively: no existing quota type is considered Terraform-managed.
	return d.Set(backlogQuotaManagedTypesStateAttr, []interface{}{})
}

func backlogQuotaTypeStateValues(types map[utils.BacklogQuotaType]struct{}) []interface{} {
	values := make([]interface{}, 0, len(types))
	for quotaType := range types {
		values = append(values, quotaType.String())
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].(string) < values[j].(string)
	})
	return values
}

func removedManagedBacklogQuotaTypes(
	oldQuotas *schema.Set,
	newQuotas *schema.Set,
	rawState cty.Value,
) ([]utils.BacklogQuotaType, error) {
	managedTypes, ownershipKnown, err := rawStateBacklogQuotaManagedTypes(rawState)
	if err != nil {
		return nil, err
	}
	if !ownershipKnown {
		return []utils.BacklogQuotaType{}, nil
	}

	removed, err := removedBacklogQuotaTypes(oldQuotas, newQuotas)
	if err != nil {
		return nil, err
	}
	managedRemoved := make([]utils.BacklogQuotaType, 0, len(removed))
	for _, quotaType := range removed {
		if _, managed := managedTypes[quotaType]; managed {
			managedRemoved = append(managedRemoved, quotaType)
		}
	}
	return managedRemoved, nil
}

func removedBacklogQuotaTypes(oldQuotas, newQuotas *schema.Set) ([]utils.BacklogQuotaType, error) {
	oldTypes, err := backlogQuotaTypes(oldQuotas)
	if err != nil {
		return nil, err
	}
	newTypes, err := backlogQuotaTypes(newQuotas)
	if err != nil {
		return nil, err
	}

	removed := make([]utils.BacklogQuotaType, 0)
	for quotaType := range oldTypes {
		if _, ok := newTypes[quotaType]; !ok {
			removed = append(removed, quotaType)
		}
	}
	sort.Slice(removed, func(i, j int) bool {
		return removed[i].String() < removed[j].String()
	})
	return removed, nil
}

func backlogQuotaType(quota interface{}) (utils.BacklogQuotaType, error) {
	data, ok := quota.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected backlog quota value %T", quota)
	}
	quotaTypeValue, ok := data["type"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected backlog quota type value %T", data["type"])
	}
	return utils.ParseBacklogQuotaType(quotaTypeValue)
}

func backlogQuotaTypes(quotas *schema.Set) (map[utils.BacklogQuotaType]struct{}, error) {
	types := make(map[utils.BacklogQuotaType]struct{})
	if quotas == nil {
		return types, nil
	}

	for _, quota := range quotas.List() {
		quotaType, err := backlogQuotaType(quota)
		if err != nil {
			return nil, err
		}
		types[quotaType] = struct{}{}
	}
	return types, nil
}

func policyNullableIntToStateValue(value *int) int {
	if value == nil {
		return -1
	}

	return *value
}

func isIgnorableNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var adminErr rest.Error
	if errors.As(err, &adminErr) {
		return adminErr.Code == 404
	}

	return strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found")
}

// Supported Pulsar versions report an unset namespace policy as success: 2.10 uses 204 for
// persistence/dispatch and an empty backlog map, while 3.x/4.x use 200 with an empty or null body.
// After GetNamespaces has confirmed the namespace exists, a typed 404 from a policy endpoint means
// it disappeared during the refresh race, so the resource ID must be cleared.
func isAdminNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var adminErr rest.Error
	return errors.As(err, &adminErr) && adminErr.Code == 404
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
	_ = d.Set("inactive_topic", nil)
	_ = d.Set("backlog_quota", nil)
	_ = d.Set(backlogQuotaManagedTypesStateAttr, nil)
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

func unmarshalInactiveTopicPolicies(v *schema.Set) (*utils.InactiveTopicPolicies, error) {
	policies := v.List()
	if len(policies) == 0 {
		return nil, fmt.Errorf("inactive topic policies configuration is empty")
	}

	data := policies[0].(map[string]interface{})

	enableDeleteWhileInactive := data["enable_delete_while_inactive"].(bool)
	maxInactiveDurationStr := data["max_inactive_duration"].(string)
	deleteModeStr := data["delete_mode"].(string)

	maxInactiveDurationSeconds, err := parseInactiveTopicDurationSeconds(maxInactiveDurationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid max_inactive_duration %q: %w", maxInactiveDurationStr, err)
	}

	deleteMode, err := utils.ParseInactiveTopicDeleteMode(deleteModeStr)
	if err != nil {
		return nil, err
	}

	inactiveTopicPolicies := utils.NewInactiveTopicPolicies(
		&deleteMode,
		maxInactiveDurationSeconds,
		enableDeleteWhileInactive,
	)
	return &inactiveTopicPolicies, nil
}

func unmarshalNamespaceConfigList(v []interface{}) *types.NamespaceConfig {
	var nsConfig types.NamespaceConfig

	for _, ns := range v {
		data := ns.(map[string]interface{})

		nsConfig.AntiAffinity = data["anti_affinity"].(string)
		nsConfig.IsAllowAutoUpdateSchema = data["is_allow_auto_update_schema"].(bool)
		nsConfig.MaxProducersPerTopic = data["max_producers_per_topic"].(int)
		nsConfig.MaxConsumersPerTopic = data["max_consumers_per_topic"].(int)
		nsConfig.MaxConsumersPerSubscription = data["max_consumers_per_subscription"].(int)
		nsConfig.MessageTTLInSeconds = data["message_ttl_seconds"].(int)
		nsConfig.OffloadThresholdSizeInMb = data["offload_threshold_size_in_mb"].(int)
		rplClusters := data["replication_clusters"].(*schema.Set).List()
		nsConfig.ReplicationClusters = handleHCLArrayV2(rplClusters)
		if v, ok := data["schema_compatibility_strategy"]; ok && v != nil {
			nsConfig.SchemaCompatibilityStrategy = v.(string)
		}
		if v, ok := data["schema_auto_update_compatibility_strategy"]; ok && v != nil {
			nsConfig.SchemaAutoUpdateCompatibilityStrategy = v.(string)
		}
		nsConfig.SchemaValidationEnforce = data["schema_validation_enforce"].(bool)
		nsConfig.SubscriptionExpirationTimeMinutes = data["subscription_expiration_time_minutes"].(int)
	}

	return &nsConfig
}

func namespaceConfigHasSchemaCompatibilityStrategy(d *schema.ResourceData) bool {
	return namespaceConfigHasStringField(d, "schema_compatibility_strategy")
}

func namespaceConfigHasSchemaAutoUpdateCompatibilityStrategy(d *schema.ResourceData) bool {
	return namespaceConfigHasStringField(d, "schema_auto_update_compatibility_strategy")
}

func namespaceConfigHasStringField(d *schema.ResourceData, field string) bool {
	rawConfig := d.GetRawConfig()
	if !rawConfig.IsNull() {
		return rawValueHasNamespaceConfigStringField(rawConfig, field)
	}

	if managed, known := namespaceConfigStringFieldFromResourceData(d, field); known {
		return managed
	}

	return false
}

func namespaceConfigStringFieldFromResourceData(d *schema.ResourceData, field string) (bool, bool) {
	state := d.State()
	if state == nil {
		return false, false
	}

	if _, ok := state.Attributes["namespace_config.#"]; !ok {
		return false, false
	}

	namespaceConfig, ok := d.Get("namespace_config").([]interface{})
	if !ok {
		return false, true
	}

	return namespaceConfigListHasStringField(namespaceConfig, field), true
}

func namespaceConfigListHasStringField(namespaceConfig []interface{}, field string) bool {
	if len(namespaceConfig) == 0 || namespaceConfig[0] == nil {
		return false
	}

	configBlock, ok := namespaceConfig[0].(map[string]interface{})
	if !ok {
		return false
	}

	fieldValue, ok := configBlock[field]
	if !ok || fieldValue == nil {
		return false
	}

	value, ok := fieldValue.(string)
	if !ok {
		return false
	}

	return value != ""
}

func namespaceSchemaCompatibilityStrategyStateValue(
	strategy utils.SchemaCompatibilityStrategy,
	hasExplicitValue bool,
) (string, bool) {
	if strategy == "" || strategy == utils.SchemaCompatibilityStrategyUndefined {
		if !hasExplicitValue {
			return "", false
		}

		return schemaCompatibilityStrategyToTerraformValue(utils.SchemaCompatibilityStrategyUndefined), true
	}

	terraformValue := schemaCompatibilityStrategyToTerraformValue(strategy)
	if terraformValue == "" {
		return "", false
	}

	return terraformValue, true
}

func namespaceSchemaAutoUpdateCompatibilityStrategyStateValue(
	strategy utils.SchemaAutoUpdateCompatibilityStrategy,
	hasExplicitValue bool,
) (string, bool) {
	if !hasExplicitValue || strategy == "" {
		return "", false
	}

	return strategy.String(), true
}

func rawConfigOrStateHasNamespaceConfigStringField(rawConfig cty.Value, rawState cty.Value, field string) bool {
	if !rawConfig.IsNull() {
		return rawValueHasNamespaceConfigStringField(rawConfig, field)
	}

	return rawValueHasNamespaceConfigStringField(rawState, field)
}

func rawValueHasNamespaceConfigStringField(rawValue cty.Value, field string) bool {
	if !rawValue.IsKnown() || rawValue.IsNull() {
		return false
	}

	if !rawValue.Type().IsObjectType() || !rawValue.Type().HasAttribute("namespace_config") {
		return false
	}

	namespaceConfig := rawValue.GetAttr("namespace_config")
	if !namespaceConfig.Type().IsListType() && !namespaceConfig.Type().IsTupleType() {
		return false
	}

	if !namespaceConfig.IsKnown() || namespaceConfig.IsNull() || namespaceConfig.LengthInt() == 0 {
		return false
	}

	configBlock := namespaceConfig.Index(cty.NumberIntVal(0))
	if !configBlock.IsKnown() || configBlock.IsNull() {
		return false
	}

	if !configBlock.Type().IsObjectType() || !configBlock.Type().HasAttribute(field) {
		return false
	}

	fieldValue := configBlock.GetAttr(field)
	if !fieldValue.IsKnown() || fieldValue.IsNull() || fieldValue.Type() != cty.String {
		return false
	}

	return fieldValue.AsString() != ""
}

func parseSchemaCompatibilityStrategy(strategy string) (utils.SchemaCompatibilityStrategy, error) {
	if parsed, err := utils.ParseSchemaCompatibilityStrategy(strategy); err == nil {
		return parsed, nil
	}

	normalized := camelToUpperSnake(strategy)
	if normalized == strategy {
		return "", fmt.Errorf("invalid schema compatibility strategy %s", strategy)
	}

	return utils.ParseSchemaCompatibilityStrategy(normalized)
}

func schemaCompatibilityStrategyToTerraformValue(strategy utils.SchemaCompatibilityStrategy) string {
	if strategy == "" {
		return ""
	}

	return screamingSnakeToCamel(strategy.String())
}

func camelToUpperSnake(s string) string {
	if s == "" {
		return s
	}

	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteRune('_')
		}
		b.WriteRune(unicode.ToUpper(r))
	}

	return b.String()
}

func screamingSnakeToCamel(s string) string {
	if s == "" {
		return s
	}

	parts := strings.Split(strings.ToLower(s), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}

		switch {
		case len(part) == 1:
			parts[i] = strings.ToUpper(part)
		default:
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	return strings.Join(parts, "")
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
