// https://www.terraform.io/docs/providers/pulsar/r/cluster.html
// generated from terraform resource schema

import { Construct } from 'constructs';
import * as cdktf from 'cdktf';

// Configuration

export interface ClusterConfig extends cdktf.TerraformMetaArguments {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html#cluster Cluster#cluster}
  */
  readonly cluster: string;
  /**
  * cluster_data block
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html#cluster_data Cluster#cluster_data}
  */
  readonly clusterData: ClusterClusterData[];
}
export interface ClusterClusterData {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html#broker_service_url Cluster#broker_service_url}
  */
  readonly brokerServiceUrl: string;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html#broker_service_url_tls Cluster#broker_service_url_tls}
  */
  readonly brokerServiceUrlTls?: string;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html#peer_clusters Cluster#peer_clusters}
  */
  readonly peerClusters?: string[];
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html#web_service_url Cluster#web_service_url}
  */
  readonly webServiceUrl: string;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html#web_service_url_tls Cluster#web_service_url_tls}
  */
  readonly webServiceUrlTls?: string;
}

function clusterClusterDataToTerraform(struct?: ClusterClusterData): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    broker_service_url: cdktf.stringToTerraform(struct!.brokerServiceUrl),
    broker_service_url_tls: cdktf.stringToTerraform(struct!.brokerServiceUrlTls),
    peer_clusters: cdktf.listMapper(cdktf.stringToTerraform)(struct!.peerClusters),
    web_service_url: cdktf.stringToTerraform(struct!.webServiceUrl),
    web_service_url_tls: cdktf.stringToTerraform(struct!.webServiceUrlTls),
  }
}


/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html pulsar_cluster}
*/
export class Cluster extends cdktf.TerraformResource {

  // =================
  // STATIC PROPERTIES
  // =================
  public static readonly tfResourceType: string = "pulsar_cluster";

  // ===========
  // INITIALIZER
  // ===========

  /**
  * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html pulsar_cluster} Resource
  *
  * @param scope The scope in which to define this construct
  * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
  * @param options ClusterConfig
  */
  public constructor(scope: Construct, id: string, config: ClusterConfig) {
    super(scope, id, {
      terraformResourceType: 'pulsar_cluster',
      terraformGeneratorMetadata: {
        providerName: 'pulsar'
      },
      provider: config.provider,
      dependsOn: config.dependsOn,
      count: config.count,
      lifecycle: config.lifecycle
    });
    this._cluster = config.cluster;
    this._clusterData = config.clusterData;
  }

  // ==========
  // ATTRIBUTES
  // ==========

  // cluster - computed: false, optional: false, required: true
  private _cluster: string;
  public get cluster() {
    return this.getStringAttribute('cluster');
  }
  public set cluster(value: string) {
    this._cluster = value;
  }
  // Temporarily expose input value. Use with caution.
  public get clusterInput() {
    return this._cluster
  }

  // id - computed: true, optional: true, required: false
  public get id() {
    return this.getStringAttribute('id');
  }

  // cluster_data - computed: false, optional: false, required: true
  private _clusterData: ClusterClusterData[];
  public get clusterData() {
    return this.interpolationForAttribute('cluster_data') as any;
  }
  public set clusterData(value: ClusterClusterData[]) {
    this._clusterData = value;
  }
  // Temporarily expose input value. Use with caution.
  public get clusterDataInput() {
    return this._clusterData
  }

  // =========
  // SYNTHESIS
  // =========

  protected synthesizeAttributes(): { [name: string]: any } {
    return {
      cluster: cdktf.stringToTerraform(this._cluster),
      cluster_data: cdktf.listMapper(clusterClusterDataToTerraform)(this._clusterData),
    };
  }
}
