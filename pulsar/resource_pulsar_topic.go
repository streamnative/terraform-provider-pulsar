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
		},
	}
}

func resourcePulsarTopicImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	topic, err := utils.GetTopicName(d.Id())
	if err != nil {
		return nil, err
	}

	client := meta.(pulsar.Client).Topics()

	tm, err := client.GetMetadata(*topic)
	if err != nil {
		return nil, err
	}

	_ = d.Set("tenant", topic.GetTenant())
	_ = d.Set("namespace", topic.GetNamespace())
	_ = d.Set("topic_type", topic.GetDomain())
	_ = d.Set("topic_name", topic.GetLocalName())
	_ = d.Set("partitions", tm.Partitions)

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

	return resourcePulsarTopicRead(d, meta)
}

func resourcePulsarTopicRead(d *schema.ResourceData, meta interface{}) error {
	topicName, found, err := getTopic(d, meta)
	if !found || err != nil {
		d.SetId("")
		return nil
	}

	d.SetId(topicName.String())
	return nil
}

func resourcePulsarTopicUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Topics()

	topicName, partitions, err := unmarshalTopicNameAndPartitions(d)
	if err != nil {
		return err
	}

	// Note: only partition number in partitioned-topic can apply update
	// For more info: https://github.com/streamnative/pulsarctl/blob/master/pkg/pulsar/topic.go#L36-L39
	if partitions == 0 {
		return errors.New("ERROR_UPDATE_TOPIC: only partition topic can apply update")
	}
	_, find, err := getTopic(d, meta)
	if !find || err != nil {
		return errors.New("ERROR_UPDATE_TOPIC: only partitions number support update")
	}
	err = client.Update(*topicName, partitions)
	if err != nil {
		return fmt.Errorf("ERROR_UPDATE_TOPIC: %w", err)
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
