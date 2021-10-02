import { Construct } from 'constructs';
import * as cdktf from 'cdktf';
export interface TenantConfig extends cdktf.TerraformMetaArguments {
    /**
    * Admin roles to be attached to tenant
    *
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/tenant.html#admin_roles Tenant#admin_roles}
    */
    readonly adminRoles?: string[];
    /**
    * Tenant will be able to interact with these clusters
    *
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/tenant.html#allowed_clusters Tenant#allowed_clusters}
    */
    readonly allowedClusters?: string[];
    /**
    * An administrative unit for allocating capacity and enforcing an
  authentication/authorization scheme
    *
    * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar/r/tenant.html#tenant Tenant#tenant}
    */
    readonly tenant: string;
}
/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/tenant.html pulsar_tenant}
*/
export declare class Tenant extends cdktf.TerraformResource {
    static readonly tfResourceType: string;
    /**
    * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/tenant.html pulsar_tenant} Resource
    *
    * @param scope The scope in which to define this construct
    * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
    * @param options TenantConfig
    */
    constructor(scope: Construct, id: string, config: TenantConfig);
    private _adminRoles?;
    get adminRoles(): string[];
    set adminRoles(value: string[]);
    resetAdminRoles(): void;
    get adminRolesInput(): string[] | undefined;
    private _allowedClusters?;
    get allowedClusters(): string[];
    set allowedClusters(value: string[]);
    resetAllowedClusters(): void;
    get allowedClustersInput(): string[] | undefined;
    get id(): string;
    private _tenant;
    get tenant(): string;
    set tenant(value: string);
    get tenantInput(): string;
    protected synthesizeAttributes(): {
        [name: string]: any;
    };
}
