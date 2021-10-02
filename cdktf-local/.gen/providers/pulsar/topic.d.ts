import { Construct } from 'constructs';
import * as cdktf from 'cdktf';
export interface TopicConfig extends cdktf.TerraformMetaArguments {
    /**
    * Pulsar namespaces are logical groupings of topics
    *
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#namespace Topic#namespace}
    */
    readonly namespace: string;
    /**
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#partitions Topic#partitions}
    */
    readonly partitions: number;
    /**
    * An administrative unit for allocating capacity and enforcing an
  authentication/authorization scheme
    *
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#tenant Topic#tenant}
    */
    readonly tenant: string;
    /**
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#topic_name Topic#topic_name}
    */
    readonly topicName: string;
    /**
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#topic_type Topic#topic_type}
    */
    readonly topicType: string;
    /**
    * permission_grant block
    *
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#permission_grant Topic#permission_grant}
    */
    readonly permissionGrant?: TopicPermissionGrant[];
}
export interface TopicPermissionGrant {
    /**
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#actions Topic#actions}
    */
    readonly actions: string[];
    /**
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html#role Topic#role}
    */
    readonly role: string;
}
/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html pulsar_topic}
*/
export declare class Topic extends cdktf.TerraformResource {
    static readonly tfResourceType: string;
    /**
    * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html pulsar_topic} Resource
    *
    * @param scope The scope in which to define this construct
    * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
    * @param options TopicConfig
    */
    constructor(scope: Construct, id: string, config: TopicConfig);
    get id(): string;
    private _namespace;
    get namespace(): string;
    set namespace(value: string);
    get namespaceInput(): string;
    private _partitions;
    get partitions(): number;
    set partitions(value: number);
    get partitionsInput(): number;
    private _tenant;
    get tenant(): string;
    set tenant(value: string);
    get tenantInput(): string;
    private _topicName;
    get topicName(): string;
    set topicName(value: string);
    get topicNameInput(): string;
    private _topicType;
    get topicType(): string;
    set topicType(value: string);
    get topicTypeInput(): string;
    private _permissionGrant?;
    get permissionGrant(): TopicPermissionGrant[];
    set permissionGrant(value: TopicPermissionGrant[]);
    resetPermissionGrant(): void;
    get permissionGrantInput(): TopicPermissionGrant[] | undefined;
    protected synthesizeAttributes(): {
        [name: string]: any;
    };
}
