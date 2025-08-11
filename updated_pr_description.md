This adds a new terraform resource for managing Pulsar namespace permissions independently from the namespace resource, plus a critical fix to prevent resource conflicts.

- Implements full CRUD operations for permission grants
- **Fixes namespace resource interference with external permissions**
- Includes acceptance tests covering lifecycle, updates, and drift detection
- Allows granular permission management per role/namespace pair
- Compatible with existing namespace permission_grant blocks
- Follows existing codebase patterns and includes extensive documentation

The new resource enables users to manage permissions separately from namespace lifecycle, providing more flexibility in Terraform configurations. **Both resources can now safely coexist without conflicts.**

### Motivation

Currently, Pulsar namespace permissions can only be managed through `permission_grant` blocks within the `pulsar_namespace` resource. This approach has limitations:

1. **Tight coupling**: Permission lifecycle is tied to namespace lifecycle
2. **Limited flexibility**: Cannot manage permissions independently
3. **Resource conflicts**: The namespace resource would interfere with externally managed permissions

**Critical Issue Discovered**: The original namespace resource implementation read ALL permissions from Pulsar and managed them as if they were defined in Terraform. This caused conflicts when different teams or tools managed permissions independently - the namespace resource would claim ownership of external permissions and potentially remove them during updates.

### Modifications

- **Added new resource**: `pulsar_permission_grant` with complete CRUD implementation
- **Fixed namespace resource**: Now only manages permissions explicitly defined in its configuration
- **Provider registration**: Registered resource in `provider.go` ResourcesMap
- **Schema definition**: `namespace`, `role`, and `actions` fields with proper validation
- **API integration**: Uses existing Pulsar admin client methods for permission management
- **Resource ID format**: Uses `namespace/role` pattern for uniqueness
- **Comprehensive testing**: 4 acceptance tests covering basic lifecycle, updates, drift detection, and resource coexistence
- **Conflict prevention**: Added `setPermissionGrantFiltered()` to ensure namespace resource only manages its own permissions

### Verifying this change

- [x] Make sure that the change passes the CI checks.

This change added tests and can be verified as follows:

- *Added `TestPermissionGrant` for basic resource lifecycle and idempotency testing*
- *Added `TestPermissionGrantUpdate` for testing action modifications (2→3→1 actions)*
- *Added `TestPermissionGrantExternallyRemoved` for drift detection when permissions are modified outside Terraform*
- *Added `TestNamespaceDoesNotInterfereWithExternalPermissions` to verify both resources can coexist safely*
- *All tests use real Pulsar instances and validate actual API behavior*
- *Tests include proper cleanup verification to ensure no resource leaks*

**Usage Example:**
```hcl
# Standalone permission grants
resource "pulsar_permission_grant" "app_producer" {
  namespace = "public/default" 
  role      = "my-app-producer"
  actions   = ["produce"]
}

resource "pulsar_permission_grant" "app_consumer" {
  namespace = "public/default"
  role      = "my-app-consumer" 
  actions   = ["consume"]
}

# Can coexist with namespace-managed permissions
resource "pulsar_namespace" "example" {
  tenant    = "public"
  namespace = "default"
  
  permission_grant {
    role    = "namespace-managed-role"
    actions = ["produce", "consume"]
  }
}
```

### Documentation

<!-- DO NOT REMOVE THIS SECTION. CHECK THE PROPER BOX ONLY. -->

- [x] `doc` <!-- Your PR contains doc changes. -->
- [ ] `doc-required` <!-- Your PR changes impact docs and you will update later -->
- [ ] `doc-not-needed` <!-- Your PR changes do not impact docs -->
- [ ] `doc-complete` <!-- Docs have been already added -->

Added documentation for the new resource:
- Generated documentation: docs/resources/permission_grant.md with complete schema reference
- Working examples: examples/permission_grants/main.tf demonstrating practical usage patterns  
- Schema descriptions: All fields include clear descriptions for auto-generated docs
- Multiple use cases: Examples show producer-only, consumer-only, and admin access patterns