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
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	rt "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/streamnative/terraform-provider-pulsar/hashcode"
)

// isIgnorableTopicPolicyError checks if an error indicates that a topic policy was not found
// or is disabled at the cluster level. This includes HTTP 404 errors, or specific
// HTTP 405 errors when topic-level policies are disabled.
func isIgnorableTopicPolicyError(err error) bool {
	if err == nil {
		return false
	}
	var cliErr rest.Error
	if errors.As(err, &cliErr) { // Use errors.As to get the underlying rest.Error
		if cliErr.Code == 404 {
			return true
		}
		// Check for the specific 405 error reason: "Topic level policies is disabled"
		if cliErr.Code == 405 && strings.Contains(cliErr.Reason, "Topic level policies is disabled") {
			return true
		}
	}
	return false
}

func resourcePulsarTopic() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarTopicCreate,
		ReadContext:   resourcePulsarTopicRead,
		UpdateContext: resourcePulsarTopicUpdate,
		DeleteContext: resourcePulsarTopicDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourcePulsarTopicImport,
		},
		Schema: map[string]*schema.Schema{
			"tenant": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["tenant"],
			},
			"namespace": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["namespace"],
			},
			"topic_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  descriptions["topic_type"],
				ValidateFunc: validateTopicType,
			},
			"topic_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["topic_name"],
			},
			"partitions": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  descriptions["partitions"],
				ValidateFunc: validateGtEq0,
			},
			"permission_grant": {
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 0,
				Description: `Manages permissions within this topic. **Warning:** Do not use this for roles that are ` +
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
						"msg_dispatch_rate": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"byte_dispatch_rate": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"dispatch_rate_period": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"relative_to_publish_rate": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				Set: topicDispatchRateToHash,
			},
			"retention_policies": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"retention_time_minutes": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"retention_size_mb": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"backlog_quota": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     schemaBacklogQuotaSubset(),
				Set:      hashBacklogQuotaSubset(),
			},
			"topic_config": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: descriptions["topic_config"],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"compaction_threshold": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"delayed_delivery": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Required: true,
									},
									"time": {
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
							Set: delayedDeliveryPoliciesToHash,
						},
						"inactive_topic": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enable_delete_while_inactive": {
										Type:     schema.TypeBool,
										Required: true,
									},
									"max_inactive_duration": {
										Type:     schema.TypeString,
										Required: true,
									},
									"delete_mode": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validiateDeleteMode,
									},
								},
							},
							Set: inactiveTopicPoliciesToHash,
						},
						"max_consumers": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateGtEq0,
						},
						"max_producers": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateGtEq0,
						},
						"message_ttl_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateGtEq0,
						},
						"max_unacked_messages_per_consumer": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateGtEq0,
						},
						"max_unacked_messages_per_subscription": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateGtEq0,
						},
						"msg_publish_rate": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateGtEq0,
						},
						"byte_publish_rate": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validateGtEq0,
						},
					},
				},
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
			"topic_properties": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Custom properties for the topic",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
	}
}

func resourcePulsarTopicImport(ctx context.Context, d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	topic, err := utils.GetTopicName(d.Id())
	if err != nil {
		return nil, fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err)
	}

	_ = d.Set("tenant", topic.GetTenant())
	_ = d.Set("namespace", topic.GetNamespace())
	_ = d.Set("topic_type", topic.GetDomain())
	_ = d.Set("topic_name", topic.GetLocalName())

	// When importing, we don't set topic_config explicitly here
	// Instead, we let resourcePulsarTopicRead detect non-default values
	// This ensures that we only import configurations that are non-default

	diags := resourcePulsarTopicRead(ctx, d, meta)
	if diags.HasError() {
		return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
	}
	return []*schema.ResourceData{d}, nil
}

func resourcePulsarTopicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Topics()
	var diags diag.Diagnostics

	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Create(*topicName, partitions)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC: %w", err))
	}

	// Rollback topic creation if any error occurs after this point
	defer func() {
		if diags.HasError() && d.Id() == "" {
			log.Printf("[INFO] Attempting to roll back creation of topic %s due to subsequent error.", topicName.String())
			forceDelete := true
			isTopicPartitioned := partitions > 0
			if err := client.Delete(*topicName, forceDelete, !isTopicPartitioned); err != nil {
				log.Printf("[WARN] Failed to delete topic %s during rollback: %v", topicName.String(), err)
			} else {
				log.Printf("[INFO] Successfully rolled back topic %s.", topicName.String())
			}
		}
	}()

	diags = internalPulsarTopicPoliciesCreate(d, meta, topicName)
	if !diags.HasError() {
		d.SetId(topicName.String())
		err = rt.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *rt.RetryError {
			// Sleep 1 seconds between checks so we don't overload the API
			time.Sleep(time.Second * 1)

			dia := resourcePulsarTopicRead(ctx, d, meta)
			if dia.HasError() {
				return rt.NonRetryableError(fmt.Errorf("ERROR_RETRY_READ_TOPIC: %s", dia[0].Summary))
			}
			return nil
		})

		if err != nil {
			diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_RETRY_READ_TOPIC: %w", err))...)
			return diags
		}
	}

	return diags
}

func internalPulsarTopicPoliciesCreate(d *schema.ResourceData, meta interface{},
	topicName *utils.TopicName) diag.Diagnostics {
	var diags diag.Diagnostics
	var err error

	err = retry(func() error {
		return updatePermissionGrant(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_PERMISSION_GRANT: %w", err))...)
		return diags
	}

	err = retry(func() error {
		return updateRetentionPolicies(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_RETENTION_POLICIES: %w", err))...)
		return diags
	}

	err = retry(func() error {
		return updateDeduplicationStatus(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_DEDUPLICATION_STATUS: %w", err))...)
		return diags
	}

	err = retry(func() error {
		return updateBacklogQuota(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_BACKLOG_QUOTA: %w", err))...)
		return diags
	}

	err = retry(func() error {
		return updateDispatchRate(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_DISPATCH_RATE: %w", err))...)
		return diags
	}

	err = retry(func() error {
		return updatePersistencePolicies(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_PERSISTENCE_POLICIES: %w", err))...)
		return diags
	}

	err = retry(func() error {
		return updateTopicConfig(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_CONFIG: %w", err))...)
		return diags
	}

	err = retry(func() error {
		return updateTopicProperties(d, meta, topicName)
	})
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_PROPERTIES: %w", err))...)
		return diags
	}

	return diags
}

func resourcePulsarTopicRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Topics()

	topicName, found, err := getTopic(d, meta)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			d.SetId("")
			return nil
		}
		return diag.Errorf("%v", err)
	}
	if !found {
		d.SetId("")
		return nil
	}

	d.SetId(topicName.String())

	tm, err := client.GetMetadata(*topicName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetMetadata: %w", err))
	}

	_ = d.Set("tenant", topicName.GetTenant())
	_ = d.Set("namespace", topicName.GetNamespace())
	_ = d.Set("topic_type", topicName.GetDomain())
	_ = d.Set("topic_name", topicName.GetLocalName())
	_ = d.Set("partitions", tm.Partitions)

	if permissionGrantCfg, ok := d.GetOk("permission_grant"); ok && permissionGrantCfg.(*schema.Set).Len() > 0 {
		grants, err := client.GetPermissions(*topicName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetPermissions: %w", err))
		}

		setPermissionGrantFiltered(d, grants)
	}

	if retPoliciesCfg, ok := d.GetOk("retention_policies"); ok && retPoliciesCfg.(*schema.Set).Len() > 0 {
		if topicName.IsPersistent() {

			ret, err := client.GetRetention(*topicName, true)
			if err != nil {
				if !isIgnorableTopicPolicyError(err) {
					return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetRetention: %w", err))
				}
			} else {
				_ = d.Set("retention_policies", []interface{}{
					map[string]interface{}{
						"retention_time_minutes": ret.RetentionTimeInMinutes,
						"retention_size_mb":      int(ret.RetentionSizeInMB),
					},
				})
			}
		} else {
			return diag.FromErr(errors.New("ERROR_READ_TOPIC: unsupported get retention policies for non-persistent topic"))
		}
	}

	if _, ok := d.GetOk("enable_deduplication"); ok {
		deduplicationStatus, err := client.GetDeduplicationStatus(*topicName)
		if err != nil {
			if !isIgnorableTopicPolicyError(err) {
				return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetDeduplicationStatus: %w", err))
			}
		} else {
			_ = d.Set("enable_deduplication", deduplicationStatus)
		}
	}

	if backlogQuotaCfg, ok := d.GetOk("backlog_quota"); ok && backlogQuotaCfg.(*schema.Set).Len() > 0 {
		if topicName.IsPersistent() {
			qt, err := client.GetBacklogQuotaMap(*topicName, true)
			if err != nil {
				if !isIgnorableTopicPolicyError(err) {
					return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetBacklogQuotaMap: %w", err))
				}
			} else {
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
		} else {
			return diag.FromErr(errors.New("ERROR_READ_TOPIC: unsupported get backlog quota for non-persistent topic"))
		}
	}

	if dispatchRateCfg, ok := d.GetOk("dispatch_rate"); ok && dispatchRateCfg.(*schema.Set).Len() > 0 {
		dr, err := client.GetDispatchRate(*topicName)
		if err != nil {
			if !isIgnorableTopicPolicyError(err) {
				return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetDispatchRate: %w", err))
			}
		} else if dr != nil {
			_ = d.Set("dispatch_rate", schema.NewSet(topicDispatchRateToHash, []interface{}{
				map[string]interface{}{
					"msg_dispatch_rate":        int(dr.DispatchThrottlingRateInMsg),
					"byte_dispatch_rate":       int(dr.DispatchThrottlingRateInByte),
					"dispatch_rate_period":     int(dr.RatePeriodInSecond),
					"relative_to_publish_rate": dr.RelativeToPublishRate,
				},
			}))
		}
	}

	if persPoliciesCfg, ok := d.GetOk("persistence_policies"); ok && persPoliciesCfg.(*schema.Set).Len() > 0 {
		if topicName.IsPersistent() {
			persistence, err := client.GetPersistence(*topicName)
			if err != nil {
				if !isIgnorableTopicPolicyError(err) {
					return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetPersistence: %w", err))
				}
			} else if persistence != nil {
				_ = d.Set("persistence_policies", schema.NewSet(persistencePoliciesToHash, []interface{}{
					map[string]interface{}{
						"bookkeeper_ensemble":                 int(persistence.BookkeeperEnsemble),
						"bookkeeper_write_quorum":             int(persistence.BookkeeperWriteQuorum),
						"bookkeeper_ack_quorum":               int(persistence.BookkeeperAckQuorum),
						"managed_ledger_max_mark_delete_rate": persistence.ManagedLedgerMaxMarkDeleteRate,
					},
				}))
			}
		} else {
			return diag.FromErr(errors.New("ERROR_READ_TOPIC: unsupported get persistence policies for non-persistent topic"))
		}
	}

	// Only set topic_config if explicitly requested via resource schema or if we find non-default values during import
	var topicConfigMap = make(map[string]interface{})

	compactionThreshold, err := client.GetCompactionThreshold(*topicName, true)
	if err == nil {
		topicConfigMap["compaction_threshold"] = int(compactionThreshold)
	} else if !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetCompactionThreshold: %w", err))
	}

	time.Sleep(time.Millisecond * 100)
	delayedDelivery, err := client.GetDelayedDelivery(*topicName)
	//nolint:gocritic
	if err == nil && delayedDelivery != nil {
		topicConfigMap["delayed_delivery"] = schema.NewSet(delayedDeliveryPoliciesToHash, []interface{}{
			map[string]interface{}{
				"enabled": delayedDelivery.Active,
				"time":    int(delayedDelivery.TickTime),
			},
		})
	} else if err != nil && !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetDelayedDelivery: %w", err))
	} else {
		topicConfigMap["delayed_delivery"] = schema.NewSet(delayedDeliveryPoliciesToHash, []interface{}{
			map[string]interface{}{
				"enabled": false,
				"time":    0,
			},
		})
	}

	time.Sleep(time.Millisecond * 100)
	inactiveTopicPolicies, err := client.GetInactiveTopicPolicies(*topicName, true)
	//nolint:gocritic
	if err == nil {
		topicConfigMap["inactive_topic"] = schema.NewSet(inactiveTopicPoliciesToHash, []interface{}{
			map[string]interface{}{
				"enable_delete_while_inactive": inactiveTopicPolicies.DeleteWhileInactive,
				"max_inactive_duration":        fmt.Sprintf("%ds", inactiveTopicPolicies.MaxInactiveDurationSeconds),
				"delete_mode":                  inactiveTopicPolicies.InactiveTopicDeleteMode.String(),
			},
		})
	} else if !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetInactiveTopicPolicies: %w", err))
	} else {
		topicConfigMap["inactive_topic"] = schema.NewSet(inactiveTopicPoliciesToHash, []interface{}{
			map[string]interface{}{
				"enable_delete_while_inactive": false,
				"max_inactive_duration":        "0s",
				"delete_mode":                  "delete_when_no_subscriptions",
			},
		})
	}

	time.Sleep(time.Millisecond * 100)
	maxConsumers, err := client.GetMaxConsumers(*topicName)
	if err == nil {
		topicConfigMap["max_consumers"] = maxConsumers
	} else if !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetMaxConsumers: %w", err))
	}

	time.Sleep(time.Millisecond * 100)
	maxProducers, err := client.GetMaxProducers(*topicName)
	if err == nil {
		topicConfigMap["max_producers"] = maxProducers
	} else if !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetMaxProducers: %w", err))
	}

	time.Sleep(time.Millisecond * 100)
	messageTTL, err := client.GetMessageTTL(*topicName)
	if err == nil {
		topicConfigMap["message_ttl_seconds"] = messageTTL
	} else if !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetMessageTTL: %w", err))
	}

	time.Sleep(time.Millisecond * 100)
	maxUnackedMsgPerConsumer, err := client.GetMaxUnackMessagesPerConsumer(*topicName)
	if err == nil {
		topicConfigMap["max_unacked_messages_per_consumer"] = maxUnackedMsgPerConsumer
	} else if !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetMaxUnackMessagesPerConsumer: %w", err))
	}

	time.Sleep(time.Millisecond * 100)
	maxUnackedMsgPerSubscription, err := client.GetMaxUnackMessagesPerSubscription(*topicName)
	if err == nil {
		topicConfigMap["max_unacked_messages_per_subscription"] = maxUnackedMsgPerSubscription
	} else if !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetMaxUnackMessagesPerSubscription: %w", err))
	}

	time.Sleep(time.Millisecond * 100)
	publishRate, err := client.GetPublishRate(*topicName)
	if err == nil && publishRate != nil {
		topicConfigMap["msg_publish_rate"] = int(publishRate.PublishThrottlingRateInMsg)
		topicConfigMap["byte_publish_rate"] = int(publishRate.PublishThrottlingRateInByte)
	} else if err != nil && !isIgnorableTopicPolicyError(err) {
		return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetPublishRate: %w", err))
	}

	// Only set topic_config if there are configuration values or it's explicitly requested in schema
	if len(topicConfigMap) > 0 {
		err := d.Set("topic_config", []interface{}{topicConfigMap})
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_SET_TOPIC_CONFIG: %w", err))
		}
	}

	// Read topic properties if they are configured
	time.Sleep(time.Millisecond * 100)
	if topicName.IsPersistent() {
		properties, err := client.GetProperties(*topicName)
		if err != nil {
			if !isIgnorableTopicPolicyError(err) {
				return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetProperties: %w", err))
			}
			// When the error is ignorable, it means no properties are set.
			// We set an empty map to clear any existing state.
			properties = make(map[string]string)
		}

		if properties == nil {
			// If the API returns a nil map without an error, treat it as empty.
			properties = make(map[string]string)
		}

		if err := d.Set("topic_properties", properties); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_SET_TOPIC_PROPERTIES: %w", err))
		}
	} else {
		// For non-persistent topics, ensure properties are cleared from state if they exist
		if err := d.Set("topic_properties", make(map[string]string)); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_CLEARING_TOPIC_PROPERTIES: %w", err))
		}
	}

	return nil
}

func resourcePulsarTopicUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("partitions") {
		err := updatePartitions(d, meta, topicName, partitions)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("permission_grant") {
		err := updatePermissionGrant(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("retention_policies") {
		err := updateRetentionPolicies(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("enable_deduplication") {
		err := updateDeduplicationStatus(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("backlog_quota") {
		err := updateBacklogQuota(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("dispatch_rate") {
		err := updateDispatchRate(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("persistence_policies") {
		err := updatePersistencePolicies(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("topic_config") {
		err := updateTopicConfig(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("topic_properties") {
		err := updateTopicProperties(d, meta, topicName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourcePulsarTopicRead(ctx, d, meta)
}

func resourcePulsarTopicDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Topics()

	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(*topicName, true, partitions == 0)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_DELETE_TOPIC: %w", err))
	}

	return nil
}

func getTopic(d *schema.ResourceData, meta interface{}) (*utils.TopicName, bool, error) {
	const found, notFound = true, false

	client := getClientFromMeta(meta).Topics()

	topicName, _, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return nil, false, err
	}

	namespace, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
	if err != nil {
		return nil, false, err
	}

	partitionedTopics, nonPartitionedTopics, err := client.List(*namespace)
	if err != nil {
		return nil, false, err
	}

	for _, topic := range append(partitionedTopics, nonPartitionedTopics...) {
		if topicName.String() == topic {
			return topicName, found, nil
		}
	}

	return nil, notFound, nil
}

func unmarshalTopicNameAndPartitions(d *schema.ResourceData) (*utils.TopicName, int, error) {
	topicName, err := unmarshalTopicName(d)
	if err != nil {
		// -1 indicate invalid partition
		return nil, -1, err
	}
	partitions, err := unmarshalPartitions(d)
	if err != nil {
		// -1 indicate invalid partition
		return nil, -1, err
	}

	return topicName, partitions, nil
}

func unmarshalTopicName(d *schema.ResourceData) (*utils.TopicName, error) {
	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	topicType := d.Get("topic_type").(string)
	topicName := d.Get("topic_name").(string)

	return utils.GetTopicName(topicType + "://" + tenant + "/" + namespace + "/" + topicName)
}

func unmarshalPartitions(d *schema.ResourceData) (int, error) {
	partitions := d.Get("partitions").(int)
	if partitions < 0 {
		// -1 indicate invalid partition
		return -1, errors.Errorf("invalid partition number '%d'", partitions)
	}

	return partitions, nil
}

func updatePermissionGrant(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	permissionGrantConfig := d.Get("permission_grant").(*schema.Set)
	permissionGrants, err := unmarshalPermissionGrants(permissionGrantConfig)

	if err != nil {
		return fmt.Errorf("ERROR_UPDATE_TOPIC_PERMISSION_GRANT: unmarshalPermissionGrants: %w", err)
	}

	for _, grant := range permissionGrants {
		if err = client.GrantPermission(*topicName, grant.Role, grant.Actions); err != nil {
			return fmt.Errorf("ERROR_UPDATE_TOPIC_PERMISSION_GRANT: GrantPermission: %w", err)
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
			if err = client.RevokePermission(*topicName, oldRole); err != nil {
				return fmt.Errorf("ERROR_UPDATE_TOPIC_PERMISSION_GRANT: RevokePermission: %w", err)
			}
		}
	}

	return nil
}

// waitForTopicConfigUpdate implements a polling mechanism to verify topic configuration has been applied
// It uses StateChangeConf to handle retries with increasing backoff and proper timeout handling
func waitForTopicConfigUpdate(d *schema.ResourceData, topicName *utils.TopicName,
	configType string, checkFunc func() (bool, error)) error {
	// Determine appropriate timeout - default to 1 minute if not specified
	timeout := 1 * time.Minute
	if timeouts, ok := d.GetOk("timeouts"); ok {
		if t, ok := timeouts.(map[string]interface{})["update"]; ok {
			if dur, err := time.ParseDuration(t.(string)); err == nil {
				timeout = dur
			}
		}
	}

	stateConf := &rt.StateChangeConf{
		Pending: []string{"pending", "updating"},
		Target:  []string{"success"},
		Refresh: func() (interface{}, string, error) {
			ok, err := checkFunc()
			if err != nil {
				return nil, "", err
			}
			if ok {
				return topicName, "success", nil
			}
			return nil, "updating", nil
		},
		Timeout:      timeout,
		Delay:        2 * time.Second,
		MinTimeout:   1 * time.Second,
		PollInterval: 3 * time.Second,
	}

	_, err := stateConf.WaitForStateContext(context.Background())
	if err != nil {
		return fmt.Errorf("ERROR_UPDATE_TOPIC_%s: timed out waiting for config update: %w", configType, err)
	}

	return nil
}

func updateRetentionPolicies(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
	if retentionPoliciesConfig.Len() == 0 {
		return nil
	}

	if !topicName.IsPersistent() {
		return errors.New("ERROR_UPDATE_RETENTION_POLICIES: SetRetention: " +
			"unsupported set retention policies for non-persistent topic")
	}

	if retentionPoliciesConfig.Len() > 0 {
		var policies utils.RetentionPolicies
		data := retentionPoliciesConfig.List()[0].(map[string]interface{})
		policies.RetentionTimeInMinutes = data["retention_time_minutes"].(int)
		policies.RetentionSizeInMB = int64(data["retention_size_mb"].(int))

		// First apply the retention policy
		if err := client.SetRetention(*topicName, policies); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				return fmt.Errorf("ERROR_UPDATE_RETENTION_POLICIES: SetRetention: %w", err)
			} else {
				return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_RETENTION_POLICIES: SetRetention: %w", err))
			}
		}

		// Then verify it was successfully applied using the new polling mechanism
		return waitForTopicConfigUpdate(d, topicName, "RETENTION_POLICIES", func() (bool, error) {
			// Get the current retention policy and check if it matches what we set
			ret, err := client.GetRetention(*topicName, true)
			if err != nil {
				return false, fmt.Errorf("ERROR_UPDATE_RETENTION_POLICIES: GetRetention: %w", err)
			}

			// Check if the values match what we expect
			if ret.RetentionTimeInMinutes != policies.RetentionTimeInMinutes ||
				ret.RetentionSizeInMB != policies.RetentionSizeInMB {
				return false, nil
			}
			return true, nil
		})
	}

	return nil
}

func updatePartitions(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName, partitions int) error {
	client := getClientFromMeta(meta).Topics()

	// Note: only partition number in partitioned-topic can apply update
	// For more info: https://github.com/streamnative/pulsar-admin-go/blob/master/pkg/admin/topic.go#L33-L36
	if partitions == 0 {
		return errors.New("ERROR_UPDATE_TOPIC_PARTITIONS: only partition topic can apply update")
	}
	_, find, err := getTopic(d, meta)
	if !find || err != nil {
		return errors.New("ERROR_UPDATE_TOPIC_PARTITIONS: only partitions number support update")
	}
	err = client.Update(*topicName, partitions)
	if err != nil {
		return fmt.Errorf("ERROR_UPDATE_TOPIC_PARTITIONS: %w", err)
	}

	return nil
}

type testTimer struct {
	timer *time.Timer
}

func (t *testTimer) Start(duration time.Duration) {
	t.timer = time.NewTimer(6 * time.Second)
}

func (t *testTimer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}

func (t *testTimer) C() <-chan time.Time {
	return t.timer.C
}

func retry(operation func() error) error {
	return backoff.RetryNotifyWithTimer(operation, backoff.NewExponentialBackOff(), nil, &testTimer{})
}

func topicDispatchRateToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%d-", m["msg_dispatch_rate"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["byte_dispatch_rate"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["dispatch_rate_period"].(int)))

	relativeToPublishRate := m["relative_to_publish_rate"].(bool)
	relativeToPublishRateInt := 0
	if relativeToPublishRate {
		relativeToPublishRateInt = 1
	}
	buf.WriteString(fmt.Sprintf("%d-", relativeToPublishRateInt))

	return hashcode.String(buf.String())
}

func updateDeduplicationStatus(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	if enableDeduplication, ok := d.GetOk("enable_deduplication"); ok {
		enabled := enableDeduplication.(bool)
		err := client.SetDeduplicationStatus(*topicName, enabled)
		if err != nil {
			return fmt.Errorf("ERROR_UPDATE_TOPIC_DEDUPLICATION_STATUS: %w", err)
		}
	}

	return nil
}

func updateBacklogQuota(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	backlogQuotaConfig := d.Get("backlog_quota").(*schema.Set)
	if backlogQuotaConfig.Len() == 0 {
		return nil
	}

	if !topicName.IsPersistent() {
		return errors.New("ERROR_UPDATE_BACKLOG_QUOTA: SetBacklogQuota: " +
			"unsupported set backlog quota for non-persistent topic")
	}

	backlogQuotas, err := unmarshalBacklogQuota(backlogQuotaConfig)
	if err != nil {
		return fmt.Errorf("ERROR_UPDATE_BACKLOG_QUOTA: unmarshalBacklogQuota: %w", err)
	}

	for _, item := range backlogQuotas {
		err = client.SetBacklogQuota(*topicName, item.BacklogQuota, item.backlogQuotaType)
		if err != nil {
			return fmt.Errorf("ERROR_UPDATE_BACKLOG_QUOTA: SetBacklogQuota: %w", err)
		}
	}

	return nil
}

func updateDispatchRate(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	dispatchRateConfig := d.Get("dispatch_rate").(*schema.Set)
	if dispatchRateConfig.Len() == 0 {
		return nil
	}

	for _, dr := range dispatchRateConfig.List() {
		data := dr.(map[string]interface{})

		dispatchRateData := utils.DispatchRateData{
			DispatchThrottlingRateInMsg:  int64(data["msg_dispatch_rate"].(int)),
			DispatchThrottlingRateInByte: int64(data["byte_dispatch_rate"].(int)),
			RatePeriodInSecond:           int64(data["dispatch_rate_period"].(int)),
			RelativeToPublishRate:        data["relative_to_publish_rate"].(bool),
		}

		err := client.SetDispatchRate(*topicName, dispatchRateData)
		if err != nil {
			return fmt.Errorf("ERROR_UPDATE_DISPATCH_RATE: SetDispatchRate: %w", err)
		}
	}

	return nil
}

func updatePersistencePolicies(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	persistencePoliciesConfig := d.Get("persistence_policies").(*schema.Set)
	if persistencePoliciesConfig.Len() == 0 {
		return nil
	}

	if !topicName.IsPersistent() {
		return errors.New("ERROR_UPDATE_PERSISTENCE_POLICIES: SetPersistence: " +
			"unsupported set persistence policies for non-persistent topic")
	}

	data := persistencePoliciesConfig.List()[0].(map[string]interface{})

	persistenceData := utils.PersistenceData{
		BookkeeperEnsemble:             int64(data["bookkeeper_ensemble"].(int)),
		BookkeeperWriteQuorum:          int64(data["bookkeeper_write_quorum"].(int)),
		BookkeeperAckQuorum:            int64(data["bookkeeper_ack_quorum"].(int)),
		ManagedLedgerMaxMarkDeleteRate: data["managed_ledger_max_mark_delete_rate"].(float64),
	}

	// First apply the persistence policy
	if err := client.SetPersistence(*topicName, persistenceData); err != nil {
		if !isIgnorableTopicPolicyError(err) {
			return fmt.Errorf("ERROR_UPDATE_PERSISTENCE_POLICIES: SetPersistence: %w", err)
		} else {
			return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_PERSISTENCE_POLICIES: SetPersistence: %w", err))
		}
	}

	// Then verify it was successfully applied using the new polling mechanism
	return waitForTopicConfigUpdate(d, topicName, "PERSISTENCE_POLICIES", func() (bool, error) {
		// Get the current persistence policy and check if it matches what we set
		persistence, err := client.GetPersistence(*topicName)
		if err != nil {
			return false, fmt.Errorf("ERROR_UPDATE_PERSISTENCE_POLICIES: GetPersistence: %w", err)
		}

		// Check if the values match what we expect
		if persistence.BookkeeperEnsemble != persistenceData.BookkeeperEnsemble ||
			persistence.BookkeeperWriteQuorum != persistenceData.BookkeeperWriteQuorum ||
			persistence.BookkeeperAckQuorum != persistenceData.BookkeeperAckQuorum ||
			persistence.ManagedLedgerMaxMarkDeleteRate != persistenceData.ManagedLedgerMaxMarkDeleteRate {
			return false, nil
		}
		return true, nil
	})

}

func updateTopicConfig(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	topicConfigList := d.Get("topic_config").([]interface{})
	if len(topicConfigList) == 0 {
		return nil
	}

	var errs error

	topicConfig := topicConfigList[0].(map[string]interface{})

	// Set compaction threshold
	if val, ok := topicConfig["compaction_threshold"]; ok {
		compactionThresholdVal := int64(val.(int))
		if err := client.SetCompactionThreshold(*topicName, compactionThresholdVal); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				errs = errors.Wrap(errs, fmt.Sprintf("SetCompactionThreshold error: %v", err))
			} else {
				return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_COMPACTION_THRESHOLD: SetCompactionThreshold: %w", err))
			}
		} else {
			// Verify the configuration was applied
			if err := waitForTopicConfigUpdate(d, topicName, "COMPACTION_THRESHOLD", func() (bool, error) {
				compactionThreshold, err := client.GetCompactionThreshold(*topicName, false)
				if err != nil {
					return false, fmt.Errorf("ERROR_UPDATE_COMPACTION_THRESHOLD: GetCompactionThreshold: %w", err)
				}
				if compactionThreshold != compactionThresholdVal {
					return false, nil
				}
				return true, nil
			}); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetCompactionThreshold verification error: %v", err))
			}
		}
	}

	// Set delayed delivery configuration
	if val, ok := topicConfig["delayed_delivery"]; ok && val != nil {
		delayedCfg, ok := val.(*schema.Set)
		if ok && delayedCfg.Len() > 0 {
			data := delayedCfg.List()[0].(map[string]interface{})
			enabled := data["enabled"].(bool)
			tickTime := data["time"].(int)

			delayedDeliveryData := utils.DelayedDeliveryData{
				Active:   enabled,
				TickTime: float64(tickTime),
			}

			if err := client.SetDelayedDelivery(*topicName, delayedDeliveryData); err != nil {
				if !isIgnorableTopicPolicyError(err) {
					errs = errors.Wrap(errs, fmt.Sprintf("SetDelayedDelivery error: %v", err))
				} else {
					return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_DELAYED_DELIVERY: SetDelayedDelivery: %w", err))
				}
			} else {
				// Verify the configuration was applied
				if err := waitForTopicConfigUpdate(d, topicName, "DELAYED_DELIVERY", func() (bool, error) {
					delayedDelivery, err := client.GetDelayedDelivery(*topicName)
					if err != nil {
						return false, fmt.Errorf("ERROR_UPDATE_DELAYED_DELIVERY: GetDelayedDelivery: %w", err)
					}
					if delayedDelivery.Active != enabled || delayedDelivery.TickTime != float64(tickTime) {
						return false, nil
					}
					return true, nil
				}); err != nil {
					errs = errors.Wrap(errs, fmt.Sprintf("SetDelayedDelivery verification error: %v", err))
				}
			}
		}
	}

	// Set inactive topic policies
	if val, ok := topicConfig["inactive_topic"]; ok && val != nil {
		inactiveCfg, ok := val.(*schema.Set)
		if ok && inactiveCfg.Len() > 0 {
			data := inactiveCfg.List()[0].(map[string]interface{})
			enableDelete := data["enable_delete_while_inactive"].(bool)
			maxInactiveDurationStr := data["max_inactive_duration"].(string)
			deleteModeStr := data["delete_mode"].(string)

			var maxInactiveDurationSeconds = (int)(0)
			if len(maxInactiveDurationStr) > 0 {
				if maxInactiveDurationStr[len(maxInactiveDurationStr)-1] == 's' {
					if parsedTime, err := strconv.Atoi(maxInactiveDurationStr[:len(maxInactiveDurationStr)-1]); err == nil {
						maxInactiveDurationSeconds = parsedTime
					}
				}
			}

			deleteMode, err := utils.ParseInactiveTopicDeleteMode(deleteModeStr)
			if err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("ParseInactiveTopicDeleteMode error: %v", err))
				return errs
			}

			inactiveTopicPolicies := utils.NewInactiveTopicPolicies(&deleteMode, maxInactiveDurationSeconds, enableDelete)

			if err := client.SetInactiveTopicPolicies(*topicName, inactiveTopicPolicies); err != nil {
				if !isIgnorableTopicPolicyError(err) {
					errs = errors.Wrap(errs, fmt.Sprintf("SetInactiveTopicPolicies error: %v", err))
				} else {
					return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_INACTIVE_TOPIC_POLICIES: SetInactiveTopicPolicies: %w", err))
				}
			} else {
				// Verify the configuration was applied
				if err := waitForTopicConfigUpdate(d, topicName, "INACTIVE_TOPIC_POLICIES", func() (bool, error) {
					inactiveTopicPoliciesResult, err := client.GetInactiveTopicPolicies(*topicName, false)
					if err != nil {
						return false, fmt.Errorf("ERROR_UPDATE_INACTIVE_TOPIC_POLICIES: GetInactiveTopicPolicies: %w", err)
					}
					if inactiveTopicPoliciesResult.InactiveTopicDeleteMode == nil ||
						*inactiveTopicPoliciesResult.InactiveTopicDeleteMode != deleteMode {
						return false, nil
					}
					if inactiveTopicPoliciesResult.MaxInactiveDurationSeconds != maxInactiveDurationSeconds {
						return false, nil
					}
					if inactiveTopicPoliciesResult.DeleteWhileInactive != enableDelete {
						return false, nil
					}
					return true, nil
				}); err != nil {
					errs = errors.Wrap(errs, fmt.Sprintf("SetInactiveTopicPolicies verification error: %v", err))
				}
			}
		}
	}

	// Set max consumers
	if val, ok := topicConfig["max_consumers"]; ok {
		maxConsVal := val.(int)
		if err := client.SetMaxConsumers(*topicName, maxConsVal); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxConsumers error: %v", err))
			} else {
				return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_MAX_CONSUMERS: SetMaxConsumers: %w", err))
			}
		} else {
			// Verify the configuration was applied
			if err := waitForTopicConfigUpdate(d, topicName, "MAX_CONSUMERS", func() (bool, error) {
				maxConsValue, err := client.GetMaxConsumers(*topicName)
				if err != nil {
					return false, fmt.Errorf("ERROR_UPDATE_MAX_CONSUMERS: GetMaxConsumers: %w", err)
				}
				if maxConsValue != maxConsVal {
					return false, nil
				}
				return true, nil
			}); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxConsumers verification error: %v", err))
			}
		}
	}

	// Set max producers
	if val, ok := topicConfig["max_producers"]; ok {
		maxProdVal := val.(int)
		if err := client.SetMaxProducers(*topicName, maxProdVal); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxProducers error: %v", err))
			} else {
				return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_MAX_PRODUCERS: SetMaxProducers: %w", err))
			}
		} else {
			// Verify the configuration was applied
			if err := waitForTopicConfigUpdate(d, topicName, "MAX_PRODUCERS", func() (bool, error) {
				maxProdValue, err := client.GetMaxProducers(*topicName)
				if err != nil {
					return false, fmt.Errorf("ERROR_UPDATE_MAX_PRODUCERS: GetMaxProducers: %w", err)
				}
				if maxProdValue != maxProdVal {
					return false, nil
				}
				return true, nil
			}); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxProducers verification error: %v", err))
			}
		}
	}

	// Set message TTL (in seconds)
	if val, ok := topicConfig["message_ttl_seconds"]; ok {
		ttlVal := val.(int)
		if err := client.SetMessageTTL(*topicName, ttlVal); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMessageTTL error: %v", err))
			} else {
				return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_MESSAGE_TTL: SetMessageTTL: %w", err))
			}
		} else {
			// Verify the configuration was applied
			if err := waitForTopicConfigUpdate(d, topicName, "MESSAGE_TTL", func() (bool, error) {
				ttlValue, err := client.GetMessageTTL(*topicName)
				if err != nil {
					return false, fmt.Errorf("ERROR_UPDATE_MESSAGE_TTL: GetMessageTTL: %w", err)
				}
				if ttlValue != ttlVal {
					return false, nil
				}
				return true, nil
			}); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMessageTTL verification error: %v", err))
			}
		}
	}

	if maxUnackedMsgPerConsumer, ok := topicConfig["max_unacked_messages_per_consumer"]; ok {
		maxVal := maxUnackedMsgPerConsumer.(int)
		if err := client.SetMaxUnackMessagesPerConsumer(*topicName, maxVal); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxUnackMessagesPerConsumer error: %v", err))
			} else {
				return backoff.Permanent(
					fmt.Errorf("ERROR_UPDATE_MAX_UNACK_MESSAGES_PER_CONSUMER: SetMaxUnackMessagesPerConsumer: %w", err))
			}
		} else {
			// Verify the configuration was applied
			if err := waitForTopicConfigUpdate(d, topicName, "MAX_UNACK_MESSAGES_PER_CONSUMER", func() (bool, error) {
				maxUnackedMsgPerConsumerValue, err := client.GetMaxUnackMessagesPerConsumer(*topicName)
				if err != nil {
					return false, fmt.Errorf("ERROR_UPDATE_MAX_UNACK_MESSAGES_PER_CONSUMER: GetMaxUnackMessagesPerConsumer: %w", err)
				}
				if maxUnackedMsgPerConsumerValue != maxVal {
					return false, nil
				}
				return true, nil
			}); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxUnackMessagesPerConsumer verification error: %v", err))
			}
		}
	}

	if maxUnackedMsgPerSubscription, ok := topicConfig["max_unacked_messages_per_subscription"]; ok {
		maxVal := maxUnackedMsgPerSubscription.(int)
		if err := client.SetMaxUnackMessagesPerSubscription(*topicName, maxVal); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxUnackMessagesPerSubscription error: %v", err))
			} else {
				return backoff.Permanent(
					fmt.Errorf("ERROR_UPDATE_MAX_UNACK_MESSAGES_PER_SUBSCRIPTION: SetMaxUnackMessagesPerSubscription: %w", err))
			}
		} else {
			// Verify the configuration was applied
			if err := waitForTopicConfigUpdate(d, topicName, "MAX_UNACK_MESSAGES_PER_SUBSCRIPTION", func() (bool, error) {
				maxUnackedMsgPerSubscriptionValue, err := client.GetMaxUnackMessagesPerSubscription(*topicName)
				if err != nil {
					return false, fmt.Errorf(
						"ERROR_UPDATE_MAX_UNACK_MESSAGES_PER_SUBSCRIPTION: GetMaxUnackMessagesPerSubscription: %w", err)
				}
				if maxUnackedMsgPerSubscriptionValue != maxVal {
					return false, nil
				}
				return true, nil
			}); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxUnackMessagesPerSubscription verification error: %v", err))
			}
		}
	}

	msgPublishRate, hasMsgRate := topicConfig["msg_publish_rate"]
	bytePublishRate, hasByteRate := topicConfig["byte_publish_rate"]

	if hasMsgRate || hasByteRate {
		publishRateData := utils.PublishRateData{}

		if hasMsgRate {
			publishRateData.PublishThrottlingRateInMsg = int64(msgPublishRate.(int))
		}

		if hasByteRate {
			publishRateData.PublishThrottlingRateInByte = int64(bytePublishRate.(int))
		}

		if err := client.SetPublishRate(*topicName, publishRateData); err != nil {
			if !isIgnorableTopicPolicyError(err) {
				errs = errors.Wrap(errs, fmt.Sprintf("SetPublishRate error: %v", err))
			} else {
				return backoff.Permanent(fmt.Errorf("ERROR_UPDATE_PUBLISH_RATE: SetPublishRate: %w", err))
			}
		} else {
			// Verify the configuration was applied
			if err := waitForTopicConfigUpdate(d, topicName, "PUBLISH_RATE", func() (bool, error) {
				publishRateValue, err := client.GetPublishRate(*topicName)
				if err != nil {
					return false, fmt.Errorf("ERROR_UPDATE_PUBLISH_RATE: GetPublishRate: %w", err)
				}
				if hasMsgRate && publishRateValue.PublishThrottlingRateInMsg != int64(msgPublishRate.(int)) {
					return false, nil
				}
				if hasByteRate && publishRateValue.PublishThrottlingRateInByte != int64(bytePublishRate.(int)) {
					return false, nil
				}
				return true, nil
			}); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetPublishRate verification error: %v", err))
			}
		}
	}

	return errs
}

func delayedDeliveryPoliciesToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%t-", m["enabled"].(bool)))
	buf.WriteString(fmt.Sprintf("%d-", m["time"].(int)))

	return hashcode.String(buf.String())
}

func inactiveTopicPoliciesToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%t-", m["enable_delete_while_inactive"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["max_inactive_duration"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["delete_mode"].(string)))

	return hashcode.String(buf.String())
}

func updateTopicProperties(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	if !topicName.IsPersistent() {
		if props, ok := d.GetOk("topic_properties"); ok {
			if len(props.(map[string]interface{})) > 0 {
				return fmt.Errorf("topic_properties can only be set on persistent topics")
			}
		}
		return nil
	}

	o, n := d.GetChange("topic_properties")
	if o == nil {
		o = make(map[string]interface{})
	}
	if n == nil {
		n = make(map[string]interface{})
	}

	oldMap := o.(map[string]interface{})
	newMap := n.(map[string]interface{})

	// Remove properties that are in the old map but not in the new map
	for k := range oldMap {
		if _, ok := newMap[k]; !ok {
			if err := client.RemoveProperty(*topicName, k); err != nil {
				// It's possible the property was already removed or doesn't exist, so we can ignore 404 errors
				if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
					continue
				}
				return fmt.Errorf("failed to remove property '%s': %w", k, err)
			}
		}
	}

	// Add/Update properties from the new map
	newPropMap := make(map[string]string)
	for k, v := range newMap {
		newPropMap[k] = v.(string)
	}

	if len(newPropMap) > 0 {
		if err := client.UpdateProperties(*topicName, newPropMap); err != nil {
			return fmt.Errorf("UpdateProperties failed: %w", err)
		}
	} else if len(oldMap) > 0 {
		// If the new map is empty, but the old one was not,
		// we might have just deleted all keys.
		// We still call Update with an empty map to be sure,
		// though the loop above should have handled deletions.
		if err := client.UpdateProperties(*topicName, make(map[string]string)); err != nil {
			return fmt.Errorf("UpdateProperties with empty map failed: %w", err)
		}
	}

	return nil
}
