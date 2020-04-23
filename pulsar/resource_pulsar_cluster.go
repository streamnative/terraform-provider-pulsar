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

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/cli"
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
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				_ = d.Set("cluster", d.Id())
				err := resourcePulsarClusterRead(d, meta)
				return []*schema.ResourceData{d}, err
			},
		},
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
				MinItems:    1,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"web_service_url": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateURL,
						},
						"web_service_url_tls": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateURL,
						},
						"broker_service_url": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateURL,
						},
						"broker_service_url_tls": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateURL,
						},
						"peer_clusters": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateNotBlank,
							},
						},
					},
				},
				Set: clusterDataToHash,
			},
		},
	}
}

func resourcePulsarClusterCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Clusters()

	ok, err := resourcePulsarClusterExists(d, meta)
	if err != nil {
		return err
	}

	if ok {
		return resourcePulsarClusterRead(d, meta)
	}

	cluster := d.Get("cluster").(string)
	clusterDataSet := d.Get("cluster_data").(*schema.Set)

	clusterData := unmarshalClusterData(clusterDataSet)
	clusterData.Name = cluster

	if err := client.Create(*clusterData); err != nil {
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

	peerClusterNames := make([]interface{}, len(clusterData.PeerClusterNames))
	for i, cl := range clusterData.PeerClusterNames {
		peerClusterNames[i] = cl
	}

	d.SetId(cluster)
	_ = d.Set("cluster_data", schema.NewSet(clusterDataToHash, []interface{}{
		map[string]interface{}{
			"web_service_url":        clusterData.ServiceURL,
			"web_service_url_tls":    clusterData.ServiceURLTls,
			"broker_service_url":     clusterData.BrokerServiceURL,
			"broker_service_url_tls": clusterData.BrokerServiceURLTls,
			"peer_clusters":          peerClusterNames,
		},
	}))

	return nil
}

func resourcePulsarClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Clusters()

	clusterDataSet := d.Get("cluster_data").(*schema.Set)
	cluster := d.Get("cluster").(string)

	clusterData := unmarshalClusterData(clusterDataSet)
	clusterData.Name = cluster

	if err := client.Update(*clusterData); err != nil {
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
		return fmt.Errorf("ERROR_DELETE_CLUSTER: %w", err)
	}

	_ = d.Set("cluster", "")
	_ = d.Set("cluster_data", nil)

	return nil
}

func resourcePulsarClusterExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Clusters()

	cluster := d.Get("cluster").(string)

	if _, err := client.Get(cluster); err != nil {
		if cliErr, ok := err.(cli.Error); ok && cliErr.Code == 404 {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func clusterDataToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["web_service_url"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["web_service_url_tls"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["broker_service_url"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["broker_service_url_tls"].(string)))
	peerClusters := m["peer_clusters"].([]interface{})

	for _, pc := range peerClusters {
		buf.WriteString(fmt.Sprintf("%s-", pc.(string)))
	}

	return hashcode.String(buf.String())
}

func unmarshalClusterData(input *schema.Set) *utils.ClusterData {
	var cd utils.ClusterData

	for _, v := range input.List() {
		data := v.(map[string]interface{})

		cd.ServiceURL = data["web_service_url"].(string)
		cd.ServiceURLTls = data["web_service_url_tls"].(string)
		cd.BrokerServiceURL = data["broker_service_url"].(string)
		cd.BrokerServiceURLTls = data["broker_service_url_tls"].(string)
		cd.PeerClusterNames = handleHCLArrayV2(data["peer_clusters"].([]interface{}))
	}

	return &cd
}
