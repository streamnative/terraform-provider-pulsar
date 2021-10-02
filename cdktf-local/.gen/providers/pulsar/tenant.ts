// https://www.terraform.io/docs/providers/pulsar/r/tenant.html
// generated from terraform resource schema

import { Construct } from 'constructs';
import * as cdktf from 'cdktf';

// Configuration

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
export class Tenant extends cdktf.TerraformResource {

  // =================
  // STATIC PROPERTIES
  // =================
  public static readonly tfResourceType: string = "pulsar_tenant";

  // ===========
  // INITIALIZER
  // ===========

  /**
  * Create a new {@link https://www.terraform.io/docs/providers/pulsar/r/tenant.html pulsar_tenant} Resource
  *
  * @param scope The scope in which to define this construct
  * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
  * @param options TenantConfig
  */
  public constructor(scope: Construct, id: string, config: TenantConfig) {
    super(scope, id, {
      terraformResourceType: 'pulsar_tenant',
      terraformGeneratorMetadata: {
        providerName: 'pulsar'
      },
      provider: config.provider,
      dependsOn: config.dependsOn,
      count: config.count,
      lifecycle: config.lifecycle
    });
    this._adminRoles = config.adminRoles;
    this._allowedClusters = config.allowedClusters;
    this._tenant = config.tenant;
  }

  // ==========
  // ATTRIBUTES
  // ==========

  // admin_roles - computed: false, optional: true, required: false
  private _adminRoles?: string[];
  public get adminRoles() {
    return this.getListAttribute('admin_roles');
  }
  public set adminRoles(value: string[] ) {
    this._adminRoles = value;
  }
  public resetAdminRoles() {
    this._adminRoles = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get adminRolesInput() {
    return this._adminRoles
  }

  // allowed_clusters - computed: false, optional: true, required: false
  private _allowedClusters?: string[];
  public get allowedClusters() {
    return this.getListAttribute('allowed_clusters');
  }
  public set allowedClusters(value: string[] ) {
    this._allowedClusters = value;
  }
  public resetAllowedClusters() {
    this._allowedClusters = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get allowedClustersInput() {
    return this._allowedClusters
  }

  // id - computed: true, optional: true, required: false
  public get id() {
    return this.getStringAttribute('id');
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

  // =========
  // SYNTHESIS
  // =========

  protected synthesizeAttributes(): { [name: string]: any } {
    return {
      admin_roles: cdktf.listMapper(cdktf.stringToTerraform)(this._adminRoles),
      allowed_clusters: cdktf.listMapper(cdktf.stringToTerraform)(this._allowedClusters),
      tenant: cdktf.stringToTerraform(this._tenant),
    };
  }
}
