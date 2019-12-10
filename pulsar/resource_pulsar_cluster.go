package pulsar

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
)

func resourcePulsarCluster() *schema.Resource {

	return &schema.Resource{
		Create: resourcePulsarClusterCreate,
		Read:   resourcePulsarClusterRead,
		Update: resourcePulsarClusterUpdate,
		Delete: resourcePulsarClusterDelete,
		Exists: resourcePulsarClusterExists,

		Schema: map[string]*schema.Schema{
			"cluster": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["cluster"],
			},
			"cluster_data": {
				Type:        schema.TypeSet,
				Description: descriptions["cluster_data"],
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"web_service_url": {
							Type:     schema.TypeString,
							Required: true,
						},
						"broker_service_url": {
							Type:     schema.TypeString,
							Required: true,
						},
						"peer_clusters": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
				Set: clusterDataToHash,
			},
		},
	}
}

func clusterDataToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["web_service_url"].(string)))
	buf.WriteString(fmt.Sprintf("%s", m["broker_service_url"].(string)))
	peerClusters := m["peer_clusters"].([]interface{})

	for _, pc := range peerClusters {
		buf.WriteString(fmt.Sprintf("%s-", pc.(string)))
	}

	return hashcode.String(buf.String())
}

func resourcePulsarClusterCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Clusters()

	cluster := d.Get("cluster").(string)
	clusterDataSet := d.Get("cluster_data").(*schema.Set)

	clusterData := utils.ClusterData{
		Name: cluster,
	}
	// @TODO code cleanup and add helper methods for code beautification
	for _, cd := range clusterDataSet.List() {
		data := cd.(map[string]interface{})

		clusterData.ServiceURL = data["web_service_url"].(string)
		clusterData.BrokerServiceURL = data["broker_service_url"].(string)
		clusterData.PeerClusterNames = handleHCLArrayV2(data["peer_clusters"].([]interface{}))
	}

	if err := client.Create(clusterData); err != nil {
		return fmt.Errorf("ERROR_CREATE_CLUSTER: %w", err)
	}

	_ = d.Set("cluster", cluster)
	return resourcePulsarClusterRead(d, meta)
}

func resourcePulsarClusterRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Clusters()

	cluster := d.Get("cluster").(string)

	clusterData, err := client.Get(cluster)
	if err != nil {
		return fmt.Errorf("ERROR_READ_CLUSTER_DATA: %w", err)
	}

	d.SetId(cluster)
	_ = d.Set("broker_service_url", clusterData.BrokerServiceURL)
	_ = d.Set("web_service_url", clusterData.ServiceURL)
	_ = d.Set("peer_clusters", clusterData.PeerClusterNames)

	return nil
}

func resourcePulsarClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Clusters()

	clusterDataSet := d.Get("cluster_data").(*schema.Set)
	cluster := d.Get("cluster").(string)

	clusterData := utils.ClusterData{
		Name: cluster,
	}
	// @TODO code cleanup and add helper methods for code beautification
	for _, cd := range clusterDataSet.List() {
		data := cd.(map[string]interface{})

		clusterData.ServiceURL = data["web_service_url"].(string)
		clusterData.BrokerServiceURL = data["broker_service_url"].(string)
		clusterData.PeerClusterNames = handleHCLArrayV2(data["peer_clusters"].([]interface{}))
	}

	if err := client.Update(clusterData); err != nil {
		return fmt.Errorf("ERROR_UPDATE_CLUSTER_DATA: %w", err)
	}

	_ = d.Set("cluster_data", clusterDataSet)
	d.SetId(cluster)

	return nil
}

func resourcePulsarClusterDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Clusters()

	Cluster := d.Get("cluster").(string)

	if err := client.Delete(Cluster); err != nil {
		return err
	}

	_ = d.Set("cluster", "")
	_ = d.Set("cluster_data", nil)

	return nil
}

func resourcePulsarClusterExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Clusters()

	cluster := d.Get("cluster").(string)

	_, err := client.Get(cluster)
	if err != nil {
		return false, err
	}

	return true, nil
}
