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

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
)

func resourcePulsarTopic() *schema.Resource {
	return &schema.Resource{
		Create: resourcePulsarTopicCreate,
		Read:   resourcePulsarTopicRead,
		Update: resourcePulsarTopicUpdate,
		Delete: resourcePulsarTopicDelete,
		Exists: resourcePulsarTopicExists,
		Importer: &schema.ResourceImporter{
			State: resourcePulsarTopicImport,
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
				Type:     schema.TypeList,
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
		},
	}
}

func resourcePulsarTopicImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	topic, err := utils.GetTopicName(d.Id())
	if err != nil {
		return nil, fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err)
	}

	_ = d.Set("tenant", topic.GetTenant())
	_ = d.Set("namespace", topic.GetNamespace())
	_ = d.Set("topic_type", topic.GetDomain())
	_ = d.Set("topic_name", topic.GetLocalName())

	err = resourcePulsarTopicRead(d, meta)
	return []*schema.ResourceData{d}, err
}

func resourcePulsarTopicCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Topics()

	ok, err := resourcePulsarTopicExists(d, meta)
	if err != nil {
		return err
	}

	if ok {
		return resourcePulsarTopicRead(d, meta)
	}

	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return err
	}

	err = client.Create(*topicName, partitions)
	if err != nil {
		return fmt.Errorf("ERROR_CREATE_TOPIC: %w", err)
	}

	err = updatePermissionGrant(d, meta, topicName)
	if err != nil {
		return fmt.Errorf("ERROR_CREATE_TOPIC_PERMISSION_GRANT: %w", err)
	}

	if topicName.IsPersistent() {
		err = updateRetentionPolicies(d, meta, topicName)
		if err != nil {
			return fmt.Errorf("ERROR_CREATE_TOPIC_RETENTION_POLICIES: %w", err)
		}
	} else {
		retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
		if retentionPoliciesConfig.Len() != 0 {
			return fmt.Errorf("ERROR_CREATE_TOPIC_RETENTION_POLICIES: " +
				"unsupported set retention policies for non-persistent topic")
		}
	}

	return resourcePulsarTopicRead(d, meta)
}

func resourcePulsarTopicRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Topics()

	topicName, found, err := getTopic(d, meta)
	if !found || err != nil {
		d.SetId("")
		return nil
	}
	d.SetId(topicName.String())

	tm, err := client.GetMetadata(*topicName)
	if err != nil {
		return fmt.Errorf("ERROR_READ_TOPIC: GetMetadata: %w", err)
	}

	_ = d.Set("tenant", topicName.GetTenant())
	_ = d.Set("namespace", topicName.GetNamespace())
	_ = d.Set("topic_type", topicName.GetDomain())
	_ = d.Set("topic_name", topicName.GetLocalName())
	_ = d.Set("partitions", tm.Partitions)

	if permissionGrantCfg, ok := d.GetOk("permission_grant"); ok && len(permissionGrantCfg.([]interface{})) > 0 {
		grants, err := client.GetPermissions(*topicName)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC: GetPermissions: %w", err)
		}

		setPermissionGrant(d, grants)
	}

	if retPoliciesCfg, ok := d.GetOk("retention_policies"); ok && retPoliciesCfg.(*schema.Set).Len() > 0 {
		if topicName.IsPersistent() {

			ret, err := client.GetRetention(*topicName, true)
			if err != nil {
				return fmt.Errorf("ERROR_READ_TOPIC: GetRetention: %w", err)
			}

			_ = d.Set("retention_policies", []interface{}{
				map[string]interface{}{
					"retention_time_minutes": ret.RetentionTimeInMinutes,
					"retention_size_mb":      int(ret.RetentionSizeInMB),
				},
			})
		}
	}

	return nil
}

func resourcePulsarTopicUpdate(d *schema.ResourceData, meta interface{}) error {
	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return err
	}

	if d.HasChange("partitions") {
		err := updatePartitions(d, meta, topicName, partitions)
		if err != nil {
			return err
		}
	}

	if d.HasChange("permission_grant") {
		err := updatePermissionGrant(d, meta, topicName)
		if err != nil {
			return err
		}
	}

	if d.HasChange("retention_policies") {
		err := updateRetentionPolicies(d, meta, topicName)
		if err != nil {
			return err
		}
	}

	return resourcePulsarTopicRead(d, meta)
}

func resourcePulsarTopicDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Topics()

	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return err
	}

	err = client.Delete(*topicName, true, partitions == 0)
	if err != nil {
		return fmt.Errorf("ERROR_DELETE_TOPIC: %w", err)
	}

	return nil
}

func resourcePulsarTopicExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	_, found, err := getTopic(d, meta)
	if err != nil {
		return false, fmt.Errorf("ERROR_READ_TOPIC: %w", err)
	}

	return found, nil
}

func getTopic(d *schema.ResourceData, meta interface{}) (*utils.TopicName, bool, error) {
	const found, notFound = true, false

	client := meta.(pulsar.Client).Topics()

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
	client := meta.(pulsar.Client).Topics()

	permissionGrantConfig := d.Get("permission_grant").([]interface{})
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
	for _, oldGrant := range oldPermissionGrants.([]interface{}) {
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
	client := meta.(pulsar.Client).Topics()

	retentionPoliciesConfig := d.Get("retention_policies").(*schema.Set)
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
	client := meta.(pulsar.Client).Topics()

	// Note: only partition number in partitioned-topic can apply update
	// For more info: https://github.com/streamnative/pulsarctl/blob/master/pkg/pulsar/topic.go#L36-L39
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
