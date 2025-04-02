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
	"context"
	"fmt"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourcePulsarSubscription() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarSubscriptionCreate,
		ReadContext:   resourcePulsarSubscriptionRead,
		UpdateContext: resourcePulsarSubscriptionUpdate,
		DeleteContext: resourcePulsarSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourcePulsarSubscriptionImport,
		},
		Schema: map[string]*schema.Schema{
			"topic_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: TopicNameValidatorDiag,
				Description:      "The topic in the format of {topic_type}://{tenant}/{namespace}/{topic_name}",
			},
			"subscription_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The subscription name",
			},
			"position": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "latest",
				ValidateFunc: validation.StringInSlice(
					[]string{"earliest", "latest"},
					false,
				),
				Description: "The initial position (earliest or latest)",
			},
		},
	}
}

func TopicNameValidatorDiag(i interface{}, p cty.Path) diag.Diagnostics {
	_, err := utils.GetTopicName(i.(string))
	if err != nil {
		return diag.Errorf("ERROR_PARSE_TOPIC_NAME: %s", err.Error())
	}
	return nil
}

func resourcePulsarSubscriptionImport(ctx context.Context, d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	// Format is expected to be: {topic}@{subscription_name}
	parts := strings.Split(d.Id(), "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("ERROR_PARSE_SUBSCRIPTION_NAME: invalid import format, expected {topic}:{subscription_name}")
	}

	topic := parts[0]
	subscriptionName := parts[1]

	// Parse topic name into components
	topicName, err := utils.GetTopicName(topic)
	if err != nil {
		return nil, fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err)
	}

	_ = d.Set("topic_name", topicName.String())
	_ = d.Set("subscription_name", subscriptionName)

	diags := resourcePulsarSubscriptionRead(ctx, d, meta)
	if diags.HasError() {
		return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
	}
	return []*schema.ResourceData{d}, nil
}

func resourcePulsarSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)

	topicName := d.Get("topic_name").(string)
	subscriptionName := d.Get("subscription_name").(string)
	position := d.Get("position").(string)

	topicFQN, err := utils.GetTopicName(topicName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
	}

	// Check if the topic exists
	exists, err := topicExists(d, client)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CHECK_TOPIC_EXISTS: %w", err))
	}
	if !exists {
		return diag.FromErr(fmt.Errorf("ERROR_TOPIC_NOT_FOUND: %s", topicFQN.String()))
	}

	// Check if subscription already exists
	subscriptions, err := getSubscriptions(topicFQN, client)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_GET_SUBSCRIPTIONS: %w", err))
	}

	for _, sub := range subscriptions {
		if sub == subscriptionName {
			// Subscription already exists
			d.SetId(fmt.Sprintf("%s:%s", topicFQN.String(), subscriptionName))
			return resourcePulsarSubscriptionRead(ctx, d, meta)
		}
	}

	// Create the subscription with the desired position
	err = createSubscription(topicFQN, subscriptionName, position, client)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_SUBSCRIPTION: %w", err))
	}

	d.SetId(fmt.Sprintf("%s:%s", topicFQN.String(), subscriptionName))
	return resourcePulsarSubscriptionRead(ctx, d, meta)
}

func resourcePulsarSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)

	topicName := d.Get("topic_name").(string)
	subscriptionName := d.Get("subscription_name").(string)

	topicFQN, err := utils.GetTopicName(topicName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
	}

	// Check if the topic exists
	exists, err := topicExists(d, client)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("ERROR_CHECK_TOPIC_EXISTS: %w", err))
	}
	if !exists {
		d.SetId("")
		return nil
	}

	// Check if the subscription exists
	subscriptions, err := getSubscriptions(topicFQN, client)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("ERROR_GET_SUBSCRIPTIONS: %w", err))
	}

	found := false
	for _, sub := range subscriptions {
		if sub == subscriptionName {
			found = true
			break
		}
	}

	if !found {
		d.SetId("")
		return nil
	}

	d.SetId(fmt.Sprintf("%s@%s", topicFQN.String(), subscriptionName))
	return nil
}

func resourcePulsarSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Subscription properties like position are not updateable after creation
	// so we just need to verify it still exists
	return resourcePulsarSubscriptionRead(ctx, d, meta)
}

func resourcePulsarSubscriptionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)

	topicName := d.Get("topic_name").(string)
	subscriptionName := d.Get("subscription_name").(string)

	topicFQN, err := utils.GetTopicName(topicName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
	}

	err = deleteSubscription(topicFQN, subscriptionName, client)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			// If the subscription does not exist, consider the delete successful
			return nil
		}
		return diag.FromErr(fmt.Errorf("ERROR_DELETE_SUBSCRIPTION: %w", err))
	}

	return nil
}

func topicExists(d *schema.ResourceData, client admin.Client) (bool, error) {

	topicName := d.Get("topic_name").(string)

	topicFQN, err := utils.GetTopicName(topicName)
	if err != nil {
		return false, err
	}

	namespaceName, err := utils.GetNameSpaceName(topicFQN.GetTenant(), topicFQN.GetNamespace())
	if err != nil {
		return false, err
	}

	partitionedTopics, nonPartitionedTopics, err := client.Topics().List(*namespaceName)
	if err != nil {
		return false, err
	}

	for _, topic := range append(partitionedTopics, nonPartitionedTopics...) {
		if topicFQN.String() == topic {
			return true, nil
		}
	}

	return false, nil
}

// getSubscriptions uses the appropriate REST API endpoint to get subscriptions for a topic
func getSubscriptions(topicName *utils.TopicName, client admin.Client) ([]string, error) {
	var subscriptions []string
	if topicName == nil {
		return subscriptions, fmt.Errorf("ERROR_GET_SUBSCRIPTIONS: topicName is nil")
	}
	subscriptions, err := client.Subscriptions().List(*topicName)
	return subscriptions, err
}

// createSubscription uses the appropriate REST API endpoint to create a subscription
func createSubscription(topicName *utils.TopicName, subscriptionName string,
	position string, client admin.Client) error {
	var messageID utils.MessageID
	if position == "earliest" {
		messageID = utils.Earliest
	} else {
		messageID = utils.Latest
	}

	return client.Subscriptions().Create(*topicName, subscriptionName, messageID)
}

// deleteSubscription uses the appropriate REST API endpoint to delete a subscription
func deleteSubscription(topicName *utils.TopicName, subscriptionName string, client admin.Client) error {
	return client.Subscriptions().Delete(*topicName, subscriptionName)
}
