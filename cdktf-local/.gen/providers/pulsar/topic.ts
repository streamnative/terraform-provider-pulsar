// https://www.terraform.io/docs/providers/pulsar/r/topic.html
// generated from terraform resource schema

import { Construct } from 'constructs';
import * as cdktf from 'cdktf';

// Configuration

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

function topicPermissionGrantToTerraform(struct?: TopicPermissionGrant): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    actions: cdktf.listMapper(cdktf.stringToTerraform)(struct!.actions),
    role: cdktf.stringToTerraform(struct!.role),
  }
}


/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html pulsar_topic}
*/
export class Topic extends cdktf.TerraformResource {

  // =================
  // STATIC PROPERTIES
  // =================
  public static readonly tfResourceType: string = "pulsar_topic";

  // ===========
  // INITIALIZER
  // ===========

  /**
  * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/topic.html pulsar_topic} Resource
  *
  * @param scope The scope in which to define this construct
  * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
  * @param options TopicConfig
  */
  public constructor(scope: Construct, id: string, config: TopicConfig) {
    super(scope, id, {
      terraformResourceType: 'pulsar_topic',
      terraformGeneratorMetadata: {
        providerName: 'pulsar'
      },
      provider: config.provider,
      dependsOn: config.dependsOn,
      count: config.count,
      lifecycle: config.lifecycle
    });
    this._namespace = config.namespace;
    this._partitions = config.partitions;
    this._tenant = config.tenant;
    this._topicName = config.topicName;
    this._topicType = config.topicType;
    this._permissionGrant = config.permissionGrant;
  }

  // ==========
  // ATTRIBUTES
  // ==========

  // id - computed: true, optional: true, required: false
  public get id() {
    return this.getStringAttribute('id');
  }

  // namespace - computed: false, optional: false, required: true
  private _namespace: string;
  public get namespace() {
    return this.getStringAttribute('namespace');
  }
  public set namespace(value: string) {
    this._namespace = value;
  }
  // Temporarily expose input value. Use with caution.
  public get namespaceInput() {
    return this._namespace
  }

  // partitions - computed: false, optional: false, required: true
  private _partitions: number;
  public get partitions() {
    return this.getNumberAttribute('partitions');
  }
  public set partitions(value: number) {
    this._partitions = value;
  }
  // Temporarily expose input value. Use with caution.
  public get partitionsInput() {
    return this._partitions
  }

  // tenant - computed: false, optional: false, required: true
  private _tenant: string;
  public get tenant() {
    return this.getStringAttribute('tenant');
  }
  public set tenant(value: string) {
    this._tenant = value;
  }
  // Temporarily expose input value. Use with caution.
  public get tenantInput() {
    return this._tenant
  }

  // topic_name - computed: false, optional: false, required: true
  private _topicName: string;
  public get topicName() {
    return this.getStringAttribute('topic_name');
  }
  public set topicName(value: string) {
    this._topicName = value;
  }
  // Temporarily expose input value. Use with caution.
  public get topicNameInput() {
    return this._topicName
  }

  // topic_type - computed: false, optional: false, required: true
  private _topicType: string;
  public get topicType() {
    return this.getStringAttribute('topic_type');
  }
  public set topicType(value: string) {
    this._topicType = value;
  }
  // Temporarily expose input value. Use with caution.
  public get topicTypeInput() {
    return this._topicType
  }

  // permission_grant - computed: false, optional: true, required: false
  private _permissionGrant?: TopicPermissionGrant[];
  public get permissionGrant() {
    return this.interpolationForAttribute('permission_grant') as any;
  }
  public set permissionGrant(value: TopicPermissionGrant[] ) {
    this._permissionGrant = value;
  }
  public resetPermissionGrant() {
    this._permissionGrant = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get permissionGrantInput() {
    return this._permissionGrant
  }

  // =========
  // SYNTHESIS
  // =========

  protected synthesizeAttributes(): { [name: string]: any } {
    return {
      namespace: cdktf.stringToTerraform(this._namespace),
      partitions: cdktf.numberToTerraform(this._partitions),
      tenant: cdktf.stringToTerraform(this._tenant),
      topic_name: cdktf.stringToTerraform(this._topicName),
      topic_type: cdktf.stringToTerraform(this._topicType),
      permission_grant: cdktf.listMapper(topicPermissionGrantToTerraform)(this._permissionGrant),
    };
  }
}
