// https://www.terraform.io/docs/providers/pulsar/r/namespace.html
// generated from terraform resource schema

import { Construct } from 'constructs';
import * as cdktf from 'cdktf';

// Configuration

export interface NamespaceConfig extends cdktf.TerraformMetaArguments {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#enable_deduplication Namespace#enable_deduplication}
  */
  readonly enableDeduplication?: boolean | cdktf.IResolvable;
  /**
  * Pulsar namespaces are logical groupings of topics
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#namespace Namespace#namespace}
  */
  readonly namespace: string;
  /**
  * An administrative unit for allocating capacity and enforcing an 
authentication/authorization scheme
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#tenant Namespace#tenant}
  */
  readonly tenant: string;
  /**
  * backlog_quota block
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#backlog_quota Namespace#backlog_quota}
  */
  readonly backlogQuota?: NamespaceBacklogQuota[];
  /**
  * dispatch_rate block
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#dispatch_rate Namespace#dispatch_rate}
  */
  readonly dispatchRate?: NamespaceDispatchRate[];
  /**
  * namespace_config block
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#namespace_config Namespace#namespace_config}
  */
  readonly namespaceConfig?: NamespaceNamespaceConfig[];
  /**
  * permission_grant block
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#permission_grant Namespace#permission_grant}
  */
  readonly permissionGrant?: NamespacePermissionGrant[];
  /**
  * persistence_policies block
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#persistence_policies Namespace#persistence_policies}
  */
  readonly persistencePolicies?: NamespacePersistencePolicies[];
  /**
  * retention_policies block
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#retention_policies Namespace#retention_policies}
  */
  readonly retentionPolicies?: NamespaceRetentionPolicies[];
}
export interface NamespaceBacklogQuota {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#limit_bytes Namespace#limit_bytes}
  */
  readonly limitBytes: string;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#policy Namespace#policy}
  */
  readonly policy: string;
}

function namespaceBacklogQuotaToTerraform(struct?: NamespaceBacklogQuota): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    limit_bytes: cdktf.stringToTerraform(struct!.limitBytes),
    policy: cdktf.stringToTerraform(struct!.policy),
  }
}

export interface NamespaceDispatchRate {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#dispatch_byte_throttling_rate Namespace#dispatch_byte_throttling_rate}
  */
  readonly dispatchByteThrottlingRate: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#dispatch_msg_throttling_rate Namespace#dispatch_msg_throttling_rate}
  */
  readonly dispatchMsgThrottlingRate: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#rate_period_seconds Namespace#rate_period_seconds}
  */
  readonly ratePeriodSeconds: number;
}

function namespaceDispatchRateToTerraform(struct?: NamespaceDispatchRate): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    dispatch_byte_throttling_rate: cdktf.numberToTerraform(struct!.dispatchByteThrottlingRate),
    dispatch_msg_throttling_rate: cdktf.numberToTerraform(struct!.dispatchMsgThrottlingRate),
    rate_period_seconds: cdktf.numberToTerraform(struct!.ratePeriodSeconds),
  }
}

export interface NamespaceNamespaceConfig {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#anti_affinity Namespace#anti_affinity}
  */
  readonly antiAffinity?: string;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#max_consumers_per_subscription Namespace#max_consumers_per_subscription}
  */
  readonly maxConsumersPerSubscription?: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#max_consumers_per_topic Namespace#max_consumers_per_topic}
  */
  readonly maxConsumersPerTopic?: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#max_producers_per_topic Namespace#max_producers_per_topic}
  */
  readonly maxProducersPerTopic?: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#replication_clusters Namespace#replication_clusters}
  */
  readonly replicationClusters?: string[];
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#schema_compatibility_strategy Namespace#schema_compatibility_strategy}
  */
  readonly schemaCompatibilityStrategy?: string;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#schema_validation_enforce Namespace#schema_validation_enforce}
  */
  readonly schemaValidationEnforce?: boolean | cdktf.IResolvable;
}

function namespaceNamespaceConfigToTerraform(struct?: NamespaceNamespaceConfig): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    anti_affinity: cdktf.stringToTerraform(struct!.antiAffinity),
    max_consumers_per_subscription: cdktf.numberToTerraform(struct!.maxConsumersPerSubscription),
    max_consumers_per_topic: cdktf.numberToTerraform(struct!.maxConsumersPerTopic),
    max_producers_per_topic: cdktf.numberToTerraform(struct!.maxProducersPerTopic),
    replication_clusters: cdktf.listMapper(cdktf.stringToTerraform)(struct!.replicationClusters),
    schema_compatibility_strategy: cdktf.stringToTerraform(struct!.schemaCompatibilityStrategy),
    schema_validation_enforce: cdktf.booleanToTerraform(struct!.schemaValidationEnforce),
  }
}

export interface NamespacePermissionGrant {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#actions Namespace#actions}
  */
  readonly actions: string[];
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#role Namespace#role}
  */
  readonly role: string;
}

function namespacePermissionGrantToTerraform(struct?: NamespacePermissionGrant): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    actions: cdktf.listMapper(cdktf.stringToTerraform)(struct!.actions),
    role: cdktf.stringToTerraform(struct!.role),
  }
}

export interface NamespacePersistencePolicies {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#bookkeeper_ack_quorum Namespace#bookkeeper_ack_quorum}
  */
  readonly bookkeeperAckQuorum: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#bookkeeper_ensemble Namespace#bookkeeper_ensemble}
  */
  readonly bookkeeperEnsemble: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#bookkeeper_write_quorum Namespace#bookkeeper_write_quorum}
  */
  readonly bookkeeperWriteQuorum: number;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#managed_ledger_max_mark_delete_rate Namespace#managed_ledger_max_mark_delete_rate}
  */
  readonly managedLedgerMaxMarkDeleteRate: number;
}

function namespacePersistencePoliciesToTerraform(struct?: NamespacePersistencePolicies): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    bookkeeper_ack_quorum: cdktf.numberToTerraform(struct!.bookkeeperAckQuorum),
    bookkeeper_ensemble: cdktf.numberToTerraform(struct!.bookkeeperEnsemble),
    bookkeeper_write_quorum: cdktf.numberToTerraform(struct!.bookkeeperWriteQuorum),
    managed_ledger_max_mark_delete_rate: cdktf.numberToTerraform(struct!.managedLedgerMaxMarkDeleteRate),
  }
}

export interface NamespaceRetentionPolicies {
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#retention_minutes Namespace#retention_minutes}
  */
  readonly retentionMinutes: string;
  /**
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html#retention_size_in_mb Namespace#retention_size_in_mb}
  */
  readonly retentionSizeInMb: string;
}

function namespaceRetentionPoliciesToTerraform(struct?: NamespaceRetentionPolicies): any {
  if (!cdktf.canInspect(struct)) { return struct; }
  return {
    retention_minutes: cdktf.stringToTerraform(struct!.retentionMinutes),
    retention_size_in_mb: cdktf.stringToTerraform(struct!.retentionSizeInMb),
  }
}


/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html pulsar_namespace}
*/
export class Namespace extends cdktf.TerraformResource {

  // =================
  // STATIC PROPERTIES
  // =================
  public static readonly tfResourceType: string = "pulsar_namespace";

  // ===========
  // INITIALIZER
  // ===========

  /**
  * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html pulsar_namespace} Resource
  *
  * @param scope The scope in which to define this construct
  * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
  * @param options NamespaceConfig
  */
  public constructor(scope: Construct, id: string, config: NamespaceConfig) {
    super(scope, id, {
      terraformResourceType: 'pulsar_namespace',
      terraformGeneratorMetadata: {
        providerName: 'pulsar'
      },
      provider: config.provider,
      dependsOn: config.dependsOn,
      count: config.count,
      lifecycle: config.lifecycle
    });
    this._enableDeduplication = config.enableDeduplication;
    this._namespace = config.namespace;
    this._tenant = config.tenant;
    this._backlogQuota = config.backlogQuota;
    this._dispatchRate = config.dispatchRate;
    this._namespaceConfig = config.namespaceConfig;
    this._permissionGrant = config.permissionGrant;
    this._persistencePolicies = config.persistencePolicies;
    this._retentionPolicies = config.retentionPolicies;
  }

  // ==========
  // ATTRIBUTES
  // ==========

  // enable_deduplication - computed: false, optional: true, required: false
  private _enableDeduplication?: boolean | cdktf.IResolvable;
  public get enableDeduplication() {
    return this.getBooleanAttribute('enable_deduplication');
  }
  public set enableDeduplication(value: boolean | cdktf.IResolvable ) {
    this._enableDeduplication = value;
  }
  public resetEnableDeduplication() {
    this._enableDeduplication = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get enableDeduplicationInput() {
    return this._enableDeduplication
  }

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

  // backlog_quota - computed: false, optional: true, required: false
  private _backlogQuota?: NamespaceBacklogQuota[];
  public get backlogQuota() {
    return this.interpolationForAttribute('backlog_quota') as any;
  }
  public set backlogQuota(value: NamespaceBacklogQuota[] ) {
    this._backlogQuota = value;
  }
  public resetBacklogQuota() {
    this._backlogQuota = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get backlogQuotaInput() {
    return this._backlogQuota
  }

  // dispatch_rate - computed: false, optional: true, required: false
  private _dispatchRate?: NamespaceDispatchRate[];
  public get dispatchRate() {
    return this.interpolationForAttribute('dispatch_rate') as any;
  }
  public set dispatchRate(value: NamespaceDispatchRate[] ) {
    this._dispatchRate = value;
  }
  public resetDispatchRate() {
    this._dispatchRate = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get dispatchRateInput() {
    return this._dispatchRate
  }

  // namespace_config - computed: false, optional: true, required: false
  private _namespaceConfig?: NamespaceNamespaceConfig[];
  public get namespaceConfig() {
    return this.interpolationForAttribute('namespace_config') as any;
  }
  public set namespaceConfig(value: NamespaceNamespaceConfig[] ) {
    this._namespaceConfig = value;
  }
  public resetNamespaceConfig() {
    this._namespaceConfig = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get namespaceConfigInput() {
    return this._namespaceConfig
  }

  // permission_grant - computed: false, optional: true, required: false
  private _permissionGrant?: NamespacePermissionGrant[];
  public get permissionGrant() {
    return this.interpolationForAttribute('permission_grant') as any;
  }
  public set permissionGrant(value: NamespacePermissionGrant[] ) {
    this._permissionGrant = value;
  }
  public resetPermissionGrant() {
    this._permissionGrant = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get permissionGrantInput() {
    return this._permissionGrant
  }

  // persistence_policies - computed: false, optional: true, required: false
  private _persistencePolicies?: NamespacePersistencePolicies[];
  public get persistencePolicies() {
    return this.interpolationForAttribute('persistence_policies') as any;
  }
  public set persistencePolicies(value: NamespacePersistencePolicies[] ) {
    this._persistencePolicies = value;
  }
  public resetPersistencePolicies() {
    this._persistencePolicies = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get persistencePoliciesInput() {
    return this._persistencePolicies
  }

  // retention_policies - computed: false, optional: true, required: false
  private _retentionPolicies?: NamespaceRetentionPolicies[];
  public get retentionPolicies() {
    return this.interpolationForAttribute('retention_policies') as any;
  }
  public set retentionPolicies(value: NamespaceRetentionPolicies[] ) {
    this._retentionPolicies = value;
  }
  public resetRetentionPolicies() {
    this._retentionPolicies = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get retentionPoliciesInput() {
    return this._retentionPolicies
  }

  // =========
  // SYNTHESIS
  // =========

  protected synthesizeAttributes(): { [name: string]: any } {
    return {
      enable_deduplication: cdktf.booleanToTerraform(this._enableDeduplication),
      namespace: cdktf.stringToTerraform(this._namespace),
      tenant: cdktf.stringToTerraform(this._tenant),
      backlog_quota: cdktf.listMapper(namespaceBacklogQuotaToTerraform)(this._backlogQuota),
      dispatch_rate: cdktf.listMapper(namespaceDispatchRateToTerraform)(this._dispatchRate),
      namespace_config: cdktf.listMapper(namespaceNamespaceConfigToTerraform)(this._namespaceConfig),
      permission_grant: cdktf.listMapper(namespacePermissionGrantToTerraform)(this._permissionGrant),
      persistence_policies: cdktf.listMapper(namespacePersistencePoliciesToTerraform)(this._persistencePolicies),
      retention_policies: cdktf.listMapper(namespaceRetentionPoliciesToTerraform)(this._retentionPolicies),
    };
  }
}
