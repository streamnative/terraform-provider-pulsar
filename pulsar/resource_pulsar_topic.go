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

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/streamnative/terraform-provider-pulsar/hashcode"
)

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
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Required: true,
									},
									"time": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
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
							ValidateFunc: validateGtEq0,
						},
						"byte_publish_rate": {
							Type:         schema.TypeInt,
							Optional:     true,
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

	diags := resourcePulsarTopicRead(ctx, d, meta)
	if diags.HasError() {
		return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
	}
	return []*schema.ResourceData{d}, nil
}

func resourcePulsarTopicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Topics()

	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Create(*topicName, partitions)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC: %w", err))
	}

	err = retry(func() error {
		return updatePermissionGrant(d, meta, topicName)
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_PERMISSION_GRANT: %w", err))
	}

	err = retry(func() error {
		return updateRetentionPolicies(d, meta, topicName)
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_RETENTION_POLICIES: %w", err))
	}

	err = retry(func() error {
		return updateDeduplicationStatus(d, meta, topicName)
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_DEDUPLICATION_STATUS: %w", err))
	}

	err = retry(func() error {
		return updateBacklogQuota(d, meta, topicName)
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_BACKLOG_QUOTA: %w", err))
	}

	err = retry(func() error {
		return updateDispatchRate(d, meta, topicName)
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_DISPATCH_RATE: %w", err))
	}

	err = retry(func() error {
		return updatePersistencePolicies(d, meta, topicName)
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_PERSISTENCE_POLICIES: %w", err))
	}

	err = retry(func() error {
		return updateTopicConfig(d, meta, topicName)
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TOPIC_CONFIG: %w", err))
	}

	return resourcePulsarTopicRead(ctx, d, meta)
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

		setPermissionGrant(d, grants)
	}

	if retPoliciesCfg, ok := d.GetOk("retention_policies"); ok && retPoliciesCfg.(*schema.Set).Len() > 0 {
		if topicName.IsPersistent() {

			ret, err := client.GetRetention(*topicName, true)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC: GetRetention: %w", err))
			}

			_ = d.Set("retention_policies", []interface{}{
				map[string]interface{}{
					"retention_time_minutes": ret.RetentionTimeInMinutes,
					"retention_size_mb":      int(ret.RetentionSizeInMB),
				},
			})
		} else {
			return diag.FromErr(errors.New("ERROR_READ_TOPIC: unsupported get retention policies for non-persistent topic"))
		}
	}

	if _, ok := d.GetOk("enable_deduplication"); ok {
		deduplicationStatus, err := client.GetDeduplicationStatus(*topicName)
		if err != nil {
			if !strings.Contains(err.Error(), "404") {
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
				if !strings.Contains(err.Error(), "404") {
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
			if !strings.Contains(err.Error(), "404") {
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
				if !strings.Contains(err.Error(), "404") {
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

	if _, ok := d.GetOk("topic_config"); ok {
		var topicConfigMap = make(map[string]interface{})

		compactionThreshold, err := client.GetCompactionThreshold(*topicName, true)
		if err == nil && compactionThreshold > 0 {
			topicConfigMap["compaction_threshold"] = int(compactionThreshold)
		}

		delayedDelivery, err := client.GetDelayedDelivery(*topicName)
		if err == nil && delayedDelivery != nil {
			topicConfigMap["delayed_delivery"] = schema.NewSet(schema.HashResource(&schema.Resource{
				Schema: map[string]*schema.Schema{
					"enabled": {
						Type:     schema.TypeBool,
						Required: true,
					},
					"time": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			}), []interface{}{
				map[string]interface{}{
					"enabled": delayedDelivery.Active,
					"time":    fmt.Sprintf("%.1fs", delayedDelivery.TickTime),
				},
			})
		}

		inactiveTopicPolicies, err := client.GetInactiveTopicPolicies(*topicName, true)
		if err == nil && inactiveTopicPolicies.InactiveTopicDeleteMode != nil {
			topicConfigMap["inactive_topic"] = schema.NewSet(schema.HashResource(&schema.Resource{
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
						Type:     schema.TypeString,
						Required: true,
					},
				},
			}), []interface{}{
				map[string]interface{}{
					"enable_delete_while_inactive": inactiveTopicPolicies.DeleteWhileInactive,
					"max_inactive_duration":        fmt.Sprintf("%ds", inactiveTopicPolicies.MaxInactiveDurationSeconds),
					"delete_mode":                  inactiveTopicPolicies.InactiveTopicDeleteMode.String(),
				},
			})
		}

		if maxConsumers, err := client.GetMaxConsumers(*topicName); err == nil && maxConsumers > 0 {
			topicConfigMap["max_consumers"] = maxConsumers
		}

		if maxProducers, err := client.GetMaxProducers(*topicName); err == nil && maxProducers > 0 {
			topicConfigMap["max_producers"] = maxProducers
		}

		if messageTTL, err := client.GetMessageTTL(*topicName); err == nil && messageTTL > 0 {
			topicConfigMap["message_ttl_seconds"] = messageTTL
		}

		if maxUnackedMsgPerConsumer, err := client.GetMaxUnackMessagesPerConsumer(*topicName); err == nil &&
			maxUnackedMsgPerConsumer > 0 {
			topicConfigMap["max_unacked_messages_per_consumer"] = maxUnackedMsgPerConsumer
		}

		if maxUnackedMsgPerSubscription, err := client.GetMaxUnackMessagesPerSubscription(*topicName); err == nil &&
			maxUnackedMsgPerSubscription > 0 {
			topicConfigMap["max_unacked_messages_per_subscription"] = maxUnackedMsgPerSubscription
		}

		if publishRate, err := client.GetPublishRate(*topicName); err == nil && publishRate != nil {
			if publishRate.PublishThrottlingRateInMsg > 0 {
				topicConfigMap["msg_publish_rate"] = int(publishRate.PublishThrottlingRateInMsg)
			}
			if publishRate.PublishThrottlingRateInByte > 0 {
				topicConfigMap["byte_publish_rate"] = int(publishRate.PublishThrottlingRateInByte)
			}
		}

		if len(topicConfigMap) > 0 {
			_ = d.Set("topic_config", []interface{}{topicConfigMap})
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
		if err := client.SetRetention(*topicName, policies); err != nil {
			return fmt.Errorf("ERROR_UPDATE_RETENTION_POLICIES: SetRetention: %w", err)
		}
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

func retry(operation func() error) error {
	return backoff.Retry(operation, backoff.NewExponentialBackOff())
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

	for _, policy := range persistencePoliciesConfig.List() {
		data := policy.(map[string]interface{})

		persistenceData := utils.PersistenceData{
			BookkeeperEnsemble:             int64(data["bookkeeper_ensemble"].(int)),
			BookkeeperWriteQuorum:          int64(data["bookkeeper_write_quorum"].(int)),
			BookkeeperAckQuorum:            int64(data["bookkeeper_ack_quorum"].(int)),
			ManagedLedgerMaxMarkDeleteRate: data["managed_ledger_max_mark_delete_rate"].(float64),
		}

		err := client.SetPersistence(*topicName, persistenceData)
		if err != nil {
			return fmt.Errorf("ERROR_UPDATE_PERSISTENCE_POLICIES: SetPersistence: %w", err)
		}
	}

	return nil
}

func updateTopicConfig(d *schema.ResourceData, meta interface{}, topicName *utils.TopicName) error {
	client := getClientFromMeta(meta).Topics()

	topicConfigList := d.Get("topic_config").([]interface{})
	if len(topicConfigList) == 0 {
		return nil
	}

	var errs error

	topicConfig := topicConfigList[0].(map[string]interface{})

	if compactionThreshold, ok := topicConfig["compaction_threshold"]; ok && compactionThreshold != nil {
		threshold := int64(compactionThreshold.(int))
		if threshold > 0 {
			if err := client.SetCompactionThreshold(*topicName, threshold); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetCompactionThreshold error: %v", err))
			}
		}
	}

	if delayedDeliveryCfg, ok := topicConfig["delayed_delivery"]; ok && delayedDeliveryCfg != nil {
		delayedCfg, ok := delayedDeliveryCfg.(*schema.Set)
		if ok && delayedCfg.Len() > 0 {
			data := delayedCfg.List()[0].(map[string]interface{})
			enabled := data["enabled"].(bool)
			timeStr := data["time"].(string)

			var tickTimeSeconds = (float64)(1.0)
			if len(timeStr) > 0 {
				if timeStr[len(timeStr)-1] == 's' {
					if parsedTime, err := strconv.ParseFloat(timeStr[:len(timeStr)-1], 64); err == nil {
						tickTimeSeconds = parsedTime
					}
				}
			}

			delayedDeliveryData := utils.DelayedDeliveryData{
				Active:   enabled,
				TickTime: tickTimeSeconds,
			}

			if err := client.SetDelayedDelivery(*topicName, delayedDeliveryData); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetDelayedDelivery error: %v", err))
			}
		}
	}

	if inactiveTopicCfg, ok := topicConfig["inactive_topic"]; ok && inactiveTopicCfg != nil {
		inactiveCfg, ok := inactiveTopicCfg.(*schema.Set)
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
				errs = errors.Wrap(errs, fmt.Sprintf("SetInactiveTopicPolicies error: %v", err))
			}
		}
	}

	if maxConsumers, ok := topicConfig["max_consumers"]; ok {
		maxConsVal := maxConsumers.(int)
		if maxConsVal > 0 {
			if err := client.SetMaxConsumers(*topicName, maxConsVal); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxConsumers error: %v", err))
			}
		}
	}

	if maxProducers, ok := topicConfig["max_producers"]; ok {
		maxProdVal := maxProducers.(int)
		if maxProdVal > 0 {
			if err := client.SetMaxProducers(*topicName, maxProdVal); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxProducers error: %v", err))
			}
		}
	}

	if messageTTL, ok := topicConfig["message_ttl_seconds"]; ok {
		ttlVal := messageTTL.(int)
		if ttlVal > 0 {
			if err := client.SetMessageTTL(*topicName, ttlVal); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMessageTTL error: %v", err))
			}
		}
	}

	if maxUnackedMsgPerConsumer, ok := topicConfig["max_unacked_messages_per_consumer"]; ok {
		maxVal := maxUnackedMsgPerConsumer.(int)
		if maxVal > 0 {
			if err := client.SetMaxUnackMessagesPerConsumer(*topicName, maxVal); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxUnackMessagesPerConsumer error: %v", err))
			}
		}
	}

	if maxUnackedMsgPerSubscription, ok := topicConfig["max_unacked_messages_per_subscription"]; ok {
		maxVal := maxUnackedMsgPerSubscription.(int)
		if maxVal > 0 {
			if err := client.SetMaxUnackMessagesPerSubscription(*topicName, maxVal); err != nil {
				errs = errors.Wrap(errs, fmt.Sprintf("SetMaxUnackMessagesPerSubscription error: %v", err))
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
			errs = errors.Wrap(errs, fmt.Sprintf("SetPublishRate error: %v", err))
		}
	}

	return errs
}
