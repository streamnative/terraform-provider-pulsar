package pulsar

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"math"
	"strconv"
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
			"namespace_list": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: descriptions["namespace_list"],
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"enable_duplication": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: descriptions["enable_duplication"],
			},
			"encrypt_topics": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: descriptions["encrypt_topics"],
			},
			"max_producers_per_topic": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     strconv.Itoa(math.MaxInt64),
				Description: descriptions["max_producers_per_topic"],
			},
			"max_consumers_per_subscription": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     strconv.Itoa(math.MaxInt64),
				Description: descriptions["max_consumers_per_subscription"],
			},
			"max_consumers_per_topic": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     strconv.Itoa(math.MaxInt64),
				Description: descriptions["max_consumers_per_topic"],
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
			"persistence_policy": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: descriptions["persistence_policy"],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bk_ensemble": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"bk_write_quorum": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"bk_ack_quorum": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"managed_ledger_max_mark_delete_rate": {
							Type:     schema.TypeFloat,
							Required: true,
						},
					},
				},
				Set: persistencePolicyToHash,
			},
			"backlog_quota": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: descriptions["backlog_quota"],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"limit": {
							Type:     schema.TypeString,
							Required: true,
						},
						"policy": {
							Type:     schema.TypeString,
							Required: true,
							//Optional:     true,
							//ExactlyOneOf: []string{"producer_request_hold", "producer_exception", "consumer_backlog_eviction"},
						},
					},
				},
				Set: backlogQuotaToHash,
			},
		},
	}
}

func backlogQuotaToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(strconv.Itoa(m["limit"].(int)))
	buf.WriteString(fmt.Sprintf("%s", m["policy"].(string)))

	return hashcode.String(buf.String())
}

func persistencePolicyToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(strconv.Itoa(m["bk_ensemble"].(int)))
	buf.WriteString(strconv.Itoa(m["bk_write_quorum"].(int)))
	buf.WriteString(strconv.Itoa(m["bk_ack_quorum"].(int)))
	buf.WriteString(fmt.Sprintf("%f", m["managed_ledger_max_mark_delete_rate"].(float64)))

	return hashcode.String(buf.String())
}

func dispatchRateToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(strconv.Itoa(m["dispatch_msg_throttling_rate"].(int)))
	buf.WriteString(strconv.Itoa(m["rate_period_seconds"].(int)))
	buf.WriteString(strconv.Itoa(m["dispatch_byte_throttling_rate"].(int)))

	return hashcode.String(buf.String())
}

func resourcePulsarNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)

	ns := fmt.Sprintf("%s/%s", tenant, namespace)

	if err := client.CreateNamespace(ns); err != nil {
		return fmt.Errorf("ERROR_CREATE_NAMESPACE: %w", err)
	}

	_ = d.Set("namespace", namespace)

	return resourcePulsarNamespaceRead(d, meta)
}

func resourcePulsarNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	nsList, err := client.GetNamespaces(tenant)
	if err != nil {
		return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
	}

	_ = d.Set("namespace", namespace)
	_ = d.Set("namespace_list", nsList)
	_ = d.Set("tenant", tenant)
	d.SetId(namespace)

	return nil
}

func resourcePulsarNamespaceUpdate(d *schema.ResourceData, meta interface{}) error {
	//client := meta.(pulsar.Client).Namespaces()

	_ = meta
	//enableDuplication := d.Get("enable_duplication").(bool)
	//encryptTopics := d.Get("encrypt_topics").(bool)
	//
	//maxProducersTopic := d.Get("max_producers_per_topic").(int)
	//maxConsumersSubscriptions := d.Get("max_consumers_per_subscription").(int)
	//maxConsumersTopic := d.Get("max_consumers_per_topic").(int)
	//compactionThreshold := d.Get("dispatch_rate").(int64)
	//
	namespace := d.Get("namespace").(string)
	tenant := d.Get("tenant").(string)
	//
	//persistencePolicy := d.Get("persistence_policy").(utils.PersistencePolicies)
	//dispatchRate := d.Get("dispatch_rate").(utils.DispatchRate)
	//backlogQuota := d.Get("backlog_quota").(utils.BacklogQuota)
	//
	//nsInfo, err := utils.GetNameSpaceName(tenant, namespace)
	//if err != nil {
	//	return err
	//}
	//
	//client.SetDeduplicationStatus(namespace, enableDuplication)
	//client.SetPersistence(namespace, persistencePolicy)
	//client.SetBacklogQuota(namespace, backlogQuota)
	//
	//client.SetMaxProducersPerTopic(*nsInfo, maxProducersTopic)
	//client.SetMaxConsumersPerSubscription(*nsInfo, maxConsumersSubscriptions)
	//client.SetMaxConsumersPerTopic(*nsInfo, maxConsumersTopic)
	//client.SetEncryptionRequiredStatus(*nsInfo, encryptTopics)
	//client.SetSubscriptionDispatchRate(*nsInfo, dispatchRate)
	//client.SetCompactionThreshold(*nsInfo, compactionThreshold)

	_ = d.Set("tenant", tenant)
	_ = d.Set("namespace", namespace)
	d.SetId(namespace)
	return nil
}

func resourcePulsarNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	namespace := d.Get("namespace").(string)

	if err := client.DeleteNamespace(namespace); err != nil {
		return err
	}

	_ = d.Set("namespace", "")
	_ = d.Set("namespace_list", nil)

	return nil
}

func resourcePulsarNamespaceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Namespaces()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)

	nsList, err := client.GetNamespaces(tenant)
	if err != nil {
		return false, err
	}

	for _, ns := range nsList {
		if ns == namespace {
			return true, nil
		}
	}

	return false, nil
}
