import { Construct } from 'constructs';
import * as cdktf from 'cdktf';
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
/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html pulsar_cluster}
*/
export declare class Cluster extends cdktf.TerraformResource {
    static readonly tfResourceType: string;
    /**
    * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/cluster.html pulsar_cluster} Resource
    *
    * @param scope The scope in which to define this construct
    * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
    * @param options ClusterConfig
    */
    constructor(scope: Construct, id: string, config: ClusterConfig);
    private _cluster;
    get cluster(): string;
    set cluster(value: string);
    get clusterInput(): string;
    get id(): string;
    private _clusterData;
    get clusterData(): ClusterClusterData[];
    set clusterData(value: ClusterClusterData[]);
    get clusterDataInput(): ClusterClusterData[];
    protected synthesizeAttributes(): {
        [name: string]: any;
    };
}
