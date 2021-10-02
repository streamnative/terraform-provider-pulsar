"use strict";
// https://www.terraform.io/docs/providers/pulsar/r/tenant.html
// generated from terraform resource schema
Object.defineProperty(exports, "__esModule", { value: true });
exports.Tenant = void 0;
const cdktf = require("cdktf");
/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar/r/tenant.html pulsar_tenant}
*/
class Tenant extends cdktf.TerraformResource {
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
    constructor(scope, id, config) {
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
    get adminRoles() {
        return this.getListAttribute('admin_roles');
    }
    set adminRoles(value) {
        this._adminRoles = value;
    }
    resetAdminRoles() {
        this._adminRoles = undefined;
    }
    // Temporarily expose input value. Use with caution.
    get adminRolesInput() {
        return this._adminRoles;
    }
    get allowedClusters() {
        return this.getListAttribute('allowed_clusters');
    }
    set allowedClusters(value) {
        this._allowedClusters = value;
    }
    resetAllowedClusters() {
        this._allowedClusters = undefined;
    }
    // Temporarily expose input value. Use with caution.
    get allowedClustersInput() {
        return this._allowedClusters;
    }
    // id - computed: true, optional: true, required: false
    get id() {
        return this.getStringAttribute('id');
    }
    get tenant() {
        return this.getStringAttribute('tenant');
    }
    set tenant(value) {
        this._tenant = value;
    }
    // Temporarily expose input value. Use with caution.
    get tenantInput() {
        return this._tenant;
    }
    // =========
    // SYNTHESIS
    // =========
    synthesizeAttributes() {
        return {
            admin_roles: cdktf.listMapper(cdktf.stringToTerraform)(this._adminRoles),
            allowed_clusters: cdktf.listMapper(cdktf.stringToTerraform)(this._allowedClusters),
            tenant: cdktf.stringToTerraform(this._tenant),
        };
    }
}
exports.Tenant = Tenant;
// =================
// STATIC PROPERTIES
// =================
Tenant.tfResourceType = "pulsar_tenant";
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidGVuYW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsidGVuYW50LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7QUFBQSwrREFBK0Q7QUFDL0QsMkNBQTJDOzs7QUFHM0MsK0JBQStCO0FBMEIvQjs7RUFFRTtBQUNGLE1BQWEsTUFBTyxTQUFRLEtBQUssQ0FBQyxpQkFBaUI7SUFPakQsY0FBYztJQUNkLGNBQWM7SUFDZCxjQUFjO0lBRWQ7Ozs7OztNQU1FO0lBQ0YsWUFBbUIsS0FBZ0IsRUFBRSxFQUFVLEVBQUUsTUFBb0I7UUFDbkUsS0FBSyxDQUFDLEtBQUssRUFBRSxFQUFFLEVBQUU7WUFDZixxQkFBcUIsRUFBRSxlQUFlO1lBQ3RDLDBCQUEwQixFQUFFO2dCQUMxQixZQUFZLEVBQUUsUUFBUTthQUN2QjtZQUNELFFBQVEsRUFBRSxNQUFNLENBQUMsUUFBUTtZQUN6QixTQUFTLEVBQUUsTUFBTSxDQUFDLFNBQVM7WUFDM0IsS0FBSyxFQUFFLE1BQU0sQ0FBQyxLQUFLO1lBQ25CLFNBQVMsRUFBRSxNQUFNLENBQUMsU0FBUztTQUM1QixDQUFDLENBQUM7UUFDSCxJQUFJLENBQUMsV0FBVyxHQUFHLE1BQU0sQ0FBQyxVQUFVLENBQUM7UUFDckMsSUFBSSxDQUFDLGdCQUFnQixHQUFHLE1BQU0sQ0FBQyxlQUFlLENBQUM7UUFDL0MsSUFBSSxDQUFDLE9BQU8sR0FBRyxNQUFNLENBQUMsTUFBTSxDQUFDO0lBQy9CLENBQUM7SUFRRCxJQUFXLFVBQVU7UUFDbkIsT0FBTyxJQUFJLENBQUMsZ0JBQWdCLENBQUMsYUFBYSxDQUFDLENBQUM7SUFDOUMsQ0FBQztJQUNELElBQVcsVUFBVSxDQUFDLEtBQWU7UUFDbkMsSUFBSSxDQUFDLFdBQVcsR0FBRyxLQUFLLENBQUM7SUFDM0IsQ0FBQztJQUNNLGVBQWU7UUFDcEIsSUFBSSxDQUFDLFdBQVcsR0FBRyxTQUFTLENBQUM7SUFDL0IsQ0FBQztJQUNELG9EQUFvRDtJQUNwRCxJQUFXLGVBQWU7UUFDeEIsT0FBTyxJQUFJLENBQUMsV0FBVyxDQUFBO0lBQ3pCLENBQUM7SUFJRCxJQUFXLGVBQWU7UUFDeEIsT0FBTyxJQUFJLENBQUMsZ0JBQWdCLENBQUMsa0JBQWtCLENBQUMsQ0FBQztJQUNuRCxDQUFDO0lBQ0QsSUFBVyxlQUFlLENBQUMsS0FBZTtRQUN4QyxJQUFJLENBQUMsZ0JBQWdCLEdBQUcsS0FBSyxDQUFDO0lBQ2hDLENBQUM7SUFDTSxvQkFBb0I7UUFDekIsSUFBSSxDQUFDLGdCQUFnQixHQUFHLFNBQVMsQ0FBQztJQUNwQyxDQUFDO0lBQ0Qsb0RBQW9EO0lBQ3BELElBQVcsb0JBQW9CO1FBQzdCLE9BQU8sSUFBSSxDQUFDLGdCQUFnQixDQUFBO0lBQzlCLENBQUM7SUFFRCx1REFBdUQ7SUFDdkQsSUFBVyxFQUFFO1FBQ1gsT0FBTyxJQUFJLENBQUMsa0JBQWtCLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDdkMsQ0FBQztJQUlELElBQVcsTUFBTTtRQUNmLE9BQU8sSUFBSSxDQUFDLGtCQUFrQixDQUFDLFFBQVEsQ0FBQyxDQUFDO0lBQzNDLENBQUM7SUFDRCxJQUFXLE1BQU0sQ0FBQyxLQUFhO1FBQzdCLElBQUksQ0FBQyxPQUFPLEdBQUcsS0FBSyxDQUFDO0lBQ3ZCLENBQUM7SUFDRCxvREFBb0Q7SUFDcEQsSUFBVyxXQUFXO1FBQ3BCLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQTtJQUNyQixDQUFDO0lBRUQsWUFBWTtJQUNaLFlBQVk7SUFDWixZQUFZO0lBRUYsb0JBQW9CO1FBQzVCLE9BQU87WUFDTCxXQUFXLEVBQUUsS0FBSyxDQUFDLFVBQVUsQ0FBQyxLQUFLLENBQUMsaUJBQWlCLENBQUMsQ0FBQyxJQUFJLENBQUMsV0FBVyxDQUFDO1lBQ3hFLGdCQUFnQixFQUFFLEtBQUssQ0FBQyxVQUFVLENBQUMsS0FBSyxDQUFDLGlCQUFpQixDQUFDLENBQUMsSUFBSSxDQUFDLGdCQUFnQixDQUFDO1lBQ2xGLE1BQU0sRUFBRSxLQUFLLENBQUMsaUJBQWlCLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQztTQUM5QyxDQUFDO0lBQ0osQ0FBQzs7QUFsR0gsd0JBbUdDO0FBakdDLG9CQUFvQjtBQUNwQixvQkFBb0I7QUFDcEIsb0JBQW9CO0FBQ0cscUJBQWMsR0FBVyxlQUFlLENBQUMiLCJzb3VyY2VzQ29udGVudCI6WyIvLyBodHRwczovL3d3dy50ZXJyYWZvcm0uaW8vZG9jcy9wcm92aWRlcnMvcHVsc2FyL3IvdGVuYW50Lmh0bWxcbi8vIGdlbmVyYXRlZCBmcm9tIHRlcnJhZm9ybSByZXNvdXJjZSBzY2hlbWFcblxuaW1wb3J0IHsgQ29uc3RydWN0IH0gZnJvbSAnY29uc3RydWN0cyc7XG5pbXBvcnQgKiBhcyBjZGt0ZiBmcm9tICdjZGt0Zic7XG5cbi8vIENvbmZpZ3VyYXRpb25cblxuZXhwb3J0IGludGVyZmFjZSBUZW5hbnRDb25maWcgZXh0ZW5kcyBjZGt0Zi5UZXJyYWZvcm1NZXRhQXJndW1lbnRzIHtcbiAgLyoqXG4gICogQWRtaW4gcm9sZXMgdG8gYmUgYXR0YWNoZWQgdG8gdGVuYW50XG4gICogXG4gICogRG9jcyBhdCBUZXJyYWZvcm0gUmVnaXN0cnk6IHtAbGluayBodHRwczovL3d3dy50ZXJyYWZvcm0uaW8vZG9jcy9wcm92aWRlcnMvcHVsc2FyL3IvdGVuYW50Lmh0bWwjYWRtaW5fcm9sZXMgVGVuYW50I2FkbWluX3JvbGVzfVxuICAqL1xuICByZWFkb25seSBhZG1pblJvbGVzPzogc3RyaW5nW107XG4gIC8qKlxuICAqIFRlbmFudCB3aWxsIGJlIGFibGUgdG8gaW50ZXJhY3Qgd2l0aCB0aGVzZSBjbHVzdGVyc1xuICAqIFxuICAqIERvY3MgYXQgVGVycmFmb3JtIFJlZ2lzdHJ5OiB7QGxpbmsgaHR0cHM6Ly93d3cudGVycmFmb3JtLmlvL2RvY3MvcHJvdmlkZXJzL3B1bHNhci9yL3RlbmFudC5odG1sI2FsbG93ZWRfY2x1c3RlcnMgVGVuYW50I2FsbG93ZWRfY2x1c3RlcnN9XG4gICovXG4gIHJlYWRvbmx5IGFsbG93ZWRDbHVzdGVycz86IHN0cmluZ1tdO1xuICAvKipcbiAgKiBBbiBhZG1pbmlzdHJhdGl2ZSB1bml0IGZvciBhbGxvY2F0aW5nIGNhcGFjaXR5IGFuZCBlbmZvcmNpbmcgYW4gXG5hdXRoZW50aWNhdGlvbi9hdXRob3JpemF0aW9uIHNjaGVtZVxuICAqIFxuICAqIERvY3MgYXQgVGVycmFmb3JtIFJlZ2lzdHJ5OiB7QGxpbmsgaHR0cHM6Ly93d3cudGVycmFmb3JtLmlvL2RvY3MvcHJvdmlkZXJzL3B1bHNhci9yL3RlbmFudC5odG1sI3RlbmFudCBUZW5hbnQjdGVuYW50fVxuICAqL1xuICByZWFkb25seSB0ZW5hbnQ6IHN0cmluZztcbn1cblxuLyoqXG4qIFJlcHJlc2VudHMgYSB7QGxpbmsgaHR0cHM6Ly93d3cudGVycmFmb3JtLmlvL2RvY3MvcHJvdmlkZXJzL3B1bHNhci9yL3RlbmFudC5odG1sIHB1bHNhcl90ZW5hbnR9XG4qL1xuZXhwb3J0IGNsYXNzIFRlbmFudCBleHRlbmRzIGNka3RmLlRlcnJhZm9ybVJlc291cmNlIHtcblxuICAvLyA9PT09PT09PT09PT09PT09PVxuICAvLyBTVEFUSUMgUFJPUEVSVElFU1xuICAvLyA9PT09PT09PT09PT09PT09PVxuICBwdWJsaWMgc3RhdGljIHJlYWRvbmx5IHRmUmVzb3VyY2VUeXBlOiBzdHJpbmcgPSBcInB1bHNhcl90ZW5hbnRcIjtcblxuICAvLyA9PT09PT09PT09PVxuICAvLyBJTklUSUFMSVpFUlxuICAvLyA9PT09PT09PT09PVxuXG4gIC8qKlxuICAqIENyZWF0ZSBhIG5ldyB7QGxpbmsgaHR0cHM6Ly93d3cudGVycmFmb3JtLmlvL2RvY3MvcHJvdmlkZXJzL3B1bHNhci9yL3RlbmFudC5odG1sIHB1bHNhcl90ZW5hbnR9IFJlc291cmNlXG4gICpcbiAgKiBAcGFyYW0gc2NvcGUgVGhlIHNjb3BlIGluIHdoaWNoIHRvIGRlZmluZSB0aGlzIGNvbnN0cnVjdFxuICAqIEBwYXJhbSBpZCBUaGUgc2NvcGVkIGNvbnN0cnVjdCBJRC4gTXVzdCBiZSB1bmlxdWUgYW1vbmdzdCBzaWJsaW5ncyBpbiB0aGUgc2FtZSBzY29wZVxuICAqIEBwYXJhbSBvcHRpb25zIFRlbmFudENvbmZpZ1xuICAqL1xuICBwdWJsaWMgY29uc3RydWN0b3Ioc2NvcGU6IENvbnN0cnVjdCwgaWQ6IHN0cmluZywgY29uZmlnOiBUZW5hbnRDb25maWcpIHtcbiAgICBzdXBlcihzY29wZSwgaWQsIHtcbiAgICAgIHRlcnJhZm9ybVJlc291cmNlVHlwZTogJ3B1bHNhcl90ZW5hbnQnLFxuICAgICAgdGVycmFmb3JtR2VuZXJhdG9yTWV0YWRhdGE6IHtcbiAgICAgICAgcHJvdmlkZXJOYW1lOiAncHVsc2FyJ1xuICAgICAgfSxcbiAgICAgIHByb3ZpZGVyOiBjb25maWcucHJvdmlkZXIsXG4gICAgICBkZXBlbmRzT246IGNvbmZpZy5kZXBlbmRzT24sXG4gICAgICBjb3VudDogY29uZmlnLmNvdW50LFxuICAgICAgbGlmZWN5Y2xlOiBjb25maWcubGlmZWN5Y2xlXG4gICAgfSk7XG4gICAgdGhpcy5fYWRtaW5Sb2xlcyA9IGNvbmZpZy5hZG1pblJvbGVzO1xuICAgIHRoaXMuX2FsbG93ZWRDbHVzdGVycyA9IGNvbmZpZy5hbGxvd2VkQ2x1c3RlcnM7XG4gICAgdGhpcy5fdGVuYW50ID0gY29uZmlnLnRlbmFudDtcbiAgfVxuXG4gIC8vID09PT09PT09PT1cbiAgLy8gQVRUUklCVVRFU1xuICAvLyA9PT09PT09PT09XG5cbiAgLy8gYWRtaW5fcm9sZXMgLSBjb21wdXRlZDogZmFsc2UsIG9wdGlvbmFsOiB0cnVlLCByZXF1aXJlZDogZmFsc2VcbiAgcHJpdmF0ZSBfYWRtaW5Sb2xlcz86IHN0cmluZ1tdO1xuICBwdWJsaWMgZ2V0IGFkbWluUm9sZXMoKSB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0TGlzdEF0dHJpYnV0ZSgnYWRtaW5fcm9sZXMnKTtcbiAgfVxuICBwdWJsaWMgc2V0IGFkbWluUm9sZXModmFsdWU6IHN0cmluZ1tdICkge1xuICAgIHRoaXMuX2FkbWluUm9sZXMgPSB2YWx1ZTtcbiAgfVxuICBwdWJsaWMgcmVzZXRBZG1pblJvbGVzKCkge1xuICAgIHRoaXMuX2FkbWluUm9sZXMgPSB1bmRlZmluZWQ7XG4gIH1cbiAgLy8gVGVtcG9yYXJpbHkgZXhwb3NlIGlucHV0IHZhbHVlLiBVc2Ugd2l0aCBjYXV0aW9uLlxuICBwdWJsaWMgZ2V0IGFkbWluUm9sZXNJbnB1dCgpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRtaW5Sb2xlc1xuICB9XG5cbiAgLy8gYWxsb3dlZF9jbHVzdGVycyAtIGNvbXB1dGVkOiBmYWxzZSwgb3B0aW9uYWw6IHRydWUsIHJlcXVpcmVkOiBmYWxzZVxuICBwcml2YXRlIF9hbGxvd2VkQ2x1c3RlcnM/OiBzdHJpbmdbXTtcbiAgcHVibGljIGdldCBhbGxvd2VkQ2x1c3RlcnMoKSB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0TGlzdEF0dHJpYnV0ZSgnYWxsb3dlZF9jbHVzdGVycycpO1xuICB9XG4gIHB1YmxpYyBzZXQgYWxsb3dlZENsdXN0ZXJzKHZhbHVlOiBzdHJpbmdbXSApIHtcbiAgICB0aGlzLl9hbGxvd2VkQ2x1c3RlcnMgPSB2YWx1ZTtcbiAgfVxuICBwdWJsaWMgcmVzZXRBbGxvd2VkQ2x1c3RlcnMoKSB7XG4gICAgdGhpcy5fYWxsb3dlZENsdXN0ZXJzID0gdW5kZWZpbmVkO1xuICB9XG4gIC8vIFRlbXBvcmFyaWx5IGV4cG9zZSBpbnB1dCB2YWx1ZS4gVXNlIHdpdGggY2F1dGlvbi5cbiAgcHVibGljIGdldCBhbGxvd2VkQ2x1c3RlcnNJbnB1dCgpIHtcbiAgICByZXR1cm4gdGhpcy5fYWxsb3dlZENsdXN0ZXJzXG4gIH1cblxuICAvLyBpZCAtIGNvbXB1dGVkOiB0cnVlLCBvcHRpb25hbDogdHJ1ZSwgcmVxdWlyZWQ6IGZhbHNlXG4gIHB1YmxpYyBnZXQgaWQoKSB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0U3RyaW5nQXR0cmlidXRlKCdpZCcpO1xuICB9XG5cbiAgLy8gdGVuYW50IC0gY29tcHV0ZWQ6IGZhbHNlLCBvcHRpb25hbDogZmFsc2UsIHJlcXVpcmVkOiB0cnVlXG4gIHByaXZhdGUgX3RlbmFudDogc3RyaW5nO1xuICBwdWJsaWMgZ2V0IHRlbmFudCgpIHtcbiAgICByZXR1cm4gdGhpcy5nZXRTdHJpbmdBdHRyaWJ1dGUoJ3RlbmFudCcpO1xuICB9XG4gIHB1YmxpYyBzZXQgdGVuYW50KHZhbHVlOiBzdHJpbmcpIHtcbiAgICB0aGlzLl90ZW5hbnQgPSB2YWx1ZTtcbiAgfVxuICAvLyBUZW1wb3JhcmlseSBleHBvc2UgaW5wdXQgdmFsdWUuIFVzZSB3aXRoIGNhdXRpb24uXG4gIHB1YmxpYyBnZXQgdGVuYW50SW5wdXQoKSB7XG4gICAgcmV0dXJuIHRoaXMuX3RlbmFudFxuICB9XG5cbiAgLy8gPT09PT09PT09XG4gIC8vIFNZTlRIRVNJU1xuICAvLyA9PT09PT09PT1cblxuICBwcm90ZWN0ZWQgc3ludGhlc2l6ZUF0dHJpYnV0ZXMoKTogeyBbbmFtZTogc3RyaW5nXTogYW55IH0ge1xuICAgIHJldHVybiB7XG4gICAgICBhZG1pbl9yb2xlczogY2RrdGYubGlzdE1hcHBlcihjZGt0Zi5zdHJpbmdUb1RlcnJhZm9ybSkodGhpcy5fYWRtaW5Sb2xlcyksXG4gICAgICBhbGxvd2VkX2NsdXN0ZXJzOiBjZGt0Zi5saXN0TWFwcGVyKGNka3RmLnN0cmluZ1RvVGVycmFmb3JtKSh0aGlzLl9hbGxvd2VkQ2x1c3RlcnMpLFxuICAgICAgdGVuYW50OiBjZGt0Zi5zdHJpbmdUb1RlcnJhZm9ybSh0aGlzLl90ZW5hbnQpLFxuICAgIH07XG4gIH1cbn1cbiJdfQ==