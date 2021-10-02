import { Construct } from 'constructs';
import * as cdktf from 'cdktf';
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
/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html pulsar_namespace}
*/
export declare class Namespace extends cdktf.TerraformResource {
    static readonly tfResourceType: string;
    /**
    * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/namespace.html pulsar_namespace} Resource
    *
    * @param scope The scope in which to define this construct
    * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
    * @param options NamespaceConfig
    */
    constructor(scope: Construct, id: string, config: NamespaceConfig);
    private _enableDeduplication?;
    get enableDeduplication(): boolean | cdktf.IResolvable;
    set enableDeduplication(value: boolean | cdktf.IResolvable);
    resetEnableDeduplication(): void;
    get enableDeduplicationInput(): boolean | cdktf.IResolvable | undefined;
    get id(): string;
    private _namespace;
    get namespace(): string;
    set namespace(value: string);
    get namespaceInput(): string;
    private _tenant;
    get tenant(): string;
    set tenant(value: string);
    get tenantInput(): string;
    private _backlogQuota?;
    get backlogQuota(): NamespaceBacklogQuota[];
    set backlogQuota(value: NamespaceBacklogQuota[]);
    resetBacklogQuota(): void;
    get backlogQuotaInput(): NamespaceBacklogQuota[] | undefined;
    private _dispatchRate?;
    get dispatchRate(): NamespaceDispatchRate[];
    set dispatchRate(value: NamespaceDispatchRate[]);
    resetDispatchRate(): void;
    get dispatchRateInput(): NamespaceDispatchRate[] | undefined;
    private _namespaceConfig?;
    get namespaceConfig(): NamespaceNamespaceConfig[];
    set namespaceConfig(value: NamespaceNamespaceConfig[]);
    resetNamespaceConfig(): void;
    get namespaceConfigInput(): NamespaceNamespaceConfig[] | undefined;
    private _permissionGrant?;
    get permissionGrant(): NamespacePermissionGrant[];
    set permissionGrant(value: NamespacePermissionGrant[]);
    resetPermissionGrant(): void;
    get permissionGrantInput(): NamespacePermissionGrant[] | undefined;
    private _persistencePolicies?;
    get persistencePolicies(): NamespacePersistencePolicies[];
    set persistencePolicies(value: NamespacePersistencePolicies[]);
    resetPersistencePolicies(): void;
    get persistencePoliciesInput(): NamespacePersistencePolicies[] | undefined;
    private _retentionPolicies?;
    get retentionPolicies(): NamespaceRetentionPolicies[];
    set retentionPolicies(value: NamespaceRetentionPolicies[]);
    resetRetentionPolicies(): void;
    get retentionPoliciesInput(): NamespaceRetentionPolicies[] | undefined;
    protected synthesizeAttributes(): {
        [name: string]: any;
    };
}
