// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pulsar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resourcePackageTypeKey        = "type"
	resourcePackageTenantKey      = "tenant"
	resourcePackageNamespaceKey   = "namespace"
	resourcePackageNameKey        = "name"
	resourcePackageVersionKey     = "version"
	resourcePackagePathKey        = "path"
	resourcePackageDescriptionKey = "description"
	resourcePackageContactKey     = "contact"
	resourcePackagePropertiesKey  = "properties"

	resourcePackagePackageURLKey     = "package_url"
	resourcePackageCreateTimeKey     = "create_time"
	resourcePackageModificationTime  = "modification_time"
	resourcePackageFileChecksumKey   = "file_checksum"
	resourcePackageFileSizeKey       = "file_size"
	resourcePackageFileNameKey       = "file_name"
	resourcePackageSourceChecksumKey = "source_hash"
)

const (
	packageMetadataPropertyTenant     = "tenant"
	packageMetadataPropertyNamespace  = "namespace"
	packageMetadataPropertyName       = "functionName"
	packageMetadataPropertyFileName   = "fileName"
	packageMetadataPropertyFileSize   = "fileSize"
	packageMetadataPropertyChecksum   = "checksum"
	packageMetadataPropertyManagedBy  = "managedByMeshWorkerService"
	packageMetadataPropertyLibDir     = "libDir"
	packageMetadataManagedByTerraform = "terraform-provider-pulsar"
)

var reservedPackageMetadataKeys = map[string]struct{}{
	packageMetadataPropertyTenant:    {},
	packageMetadataPropertyNamespace: {},
	packageMetadataPropertyName:      {},
	packageMetadataPropertyFileName:  {},
	packageMetadataPropertyFileSize:  {},
	packageMetadataPropertyChecksum:  {},
	packageMetadataPropertyManagedBy: {},
	packageMetadataPropertyLibDir:    {},
}

type packageFileInfo struct {
	checksum string
	size     int64
	name     string
}

func resourcePulsarPackage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarPackageCreate,
		ReadContext:   resourcePulsarPackageRead,
		UpdateContext: resourcePulsarPackageUpdate,
		DeleteContext: resourcePulsarPackageDelete,
		CustomizeDiff: resourcePulsarPackageCustomizeDiff,
		Description: "Manages Pulsar packages for functions, sources, and sinks. Packages bundle executable artifacts " +
			"that can be referenced by other Pulsar resources. The provider uploads the package archive and keeps the " +
			"metadata (description, contact, and custom properties) in sync with Terraform state.",

		Importer: &schema.ResourceImporter{
			StateContext: resourcePulsarPackageImport,
		},

		Schema: map[string]*schema.Schema{
			resourcePackageTypeKey: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Package type. Supported values: `function`, `sink`, `source`.",
				ValidateFunc: validatePackageType,
			},
			resourcePackageTenantKey: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Tenant that owns the package.",
			},
			resourcePackageNamespaceKey: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Namespace that owns the package.",
			},
			resourcePackageNameKey: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Package name.",
			},
			resourcePackageVersionKey: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Package version. Changing the version creates a new package resource.",
			},
			resourcePackagePathKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Local path to the package archive. Changing the file contents will trigger an update.",
			},
			resourcePackageDescriptionKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional package description stored in Pulsar metadata.",
			},
			resourcePackageContactKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Contact information stored in Pulsar metadata.",
			},
			resourcePackagePropertiesKey: {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Additional metadata properties. ",
			},
			resourcePackagePackageURLKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully-qualified package URL in the format `<type>://<tenant>/<namespace>/<name>@<version>`.",
			},
			resourcePackageCreateTimeKey: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Server-side package creation timestamp (epoch millis).",
			},
			resourcePackageModificationTime: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Server-side package modification timestamp (epoch millis).",
			},
			resourcePackageFileChecksumKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Checksum reported by Pulsar for the uploaded package.",
			},
			resourcePackageFileSizeKey: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Package size reported by Pulsar (in bytes).",
			},
			resourcePackageFileNameKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Package file name stored in metadata.",
			},
			resourcePackageSourceChecksumKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA256 checksum of the local package file. Used to detect file changes.",
			},
		},
	}
}

func resourcePulsarPackageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Packages()
	packageURL, err := packageURLFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	path := d.Get(resourcePackagePathKey).(string)
	info, err := calculatePackageFileInfo(path)
	if err != nil {
		return diag.FromErr(err)
	}

	description := d.Get(resourcePackageDescriptionKey).(string)
	contact := d.Get(resourcePackageContactKey).(string)
	properties := buildPackageMetadataProperties(d, &info)

	if err = client.Upload(packageURL, path, description, contact, properties); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(packageURL)
	_ = d.Set(resourcePackagePackageURLKey, packageURL)
	_ = d.Set(resourcePackageSourceChecksumKey, info.checksum)

	return resourcePulsarPackageRead(ctx, d, meta)
}

func resourcePulsarPackageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Packages()

	packageURL, err := ensurePackageID(d)
	if err != nil {
		return diag.FromErr(err)
	}

	metadata, err := client.GetMetadata(packageURL)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set(resourcePackagePackageURLKey, packageURL)
	_ = d.Set(resourcePackageDescriptionKey, metadata.Description)
	_ = d.Set(resourcePackageContactKey, metadata.Contact)
	_ = d.Set(resourcePackageCreateTimeKey, metadata.CreateTime)
	_ = d.Set(resourcePackageModificationTime, metadata.ModificationTime)

	userProps := filterUserManagedProperties(metadata.Properties)
	if err = d.Set(resourcePackagePropertiesKey, convertToInterfaceMap(userProps)); err != nil {
		return diag.FromErr(err)
	}

	if metadata.Properties != nil {
		if checksum, ok := metadata.Properties[packageMetadataPropertyChecksum]; ok {
			_ = d.Set(resourcePackageFileChecksumKey, checksum)
		}
		if size, ok := metadata.Properties[packageMetadataPropertyFileSize]; ok {
			if parsed, parseErr := strconv.ParseInt(size, 10, 64); parseErr == nil {
				_ = d.Set(resourcePackageFileSizeKey, parsed)
			}
		}
		if name, ok := metadata.Properties[packageMetadataPropertyFileName]; ok {
			_ = d.Set(resourcePackageFileNameKey, name)
		}
	}

	setLocalSourceChecksum(ctx, d)

	return nil
}

func resourcePulsarPackageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Packages()
	packageURL, err := packageURLFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	path := d.Get(resourcePackagePathKey).(string)
	description := d.Get(resourcePackageDescriptionKey).(string)
	contact := d.Get(resourcePackageContactKey).(string)

	reupload := d.HasChange(resourcePackagePathKey) || d.HasChange(resourcePackageSourceChecksumKey)

	if reupload {
		info, calcErr := calculatePackageFileInfo(path)
		if calcErr != nil {
			return diag.FromErr(calcErr)
		}
		properties := buildPackageMetadataProperties(d, &info)
		if err = client.Upload(packageURL, path, description, contact, properties); err != nil {
			return diag.FromErr(err)
		}
		_ = d.Set(resourcePackageSourceChecksumKey, info.checksum)
	} else if d.HasChanges(resourcePackageDescriptionKey, resourcePackageContactKey, resourcePackagePropertiesKey) {
		properties := buildPackageMetadataProperties(d, nil)
		if err = client.UpdateMetadata(packageURL, description, contact, properties); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourcePulsarPackageRead(ctx, d, meta)
}

func resourcePulsarPackageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Packages()

	packageURL, err := ensurePackageID(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = client.Delete(packageURL); err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourcePulsarPackageCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	path := diff.Get(resourcePackagePathKey).(string)
	if path == "" {
		return fmt.Errorf("path must be provided")
	}

	info, err := calculatePackageFileInfo(path)
	if err != nil {
		return err
	}

	if err = diff.SetNew(resourcePackageSourceChecksumKey, info.checksum); err != nil {
		return err
	}

	return nil
}

func resourcePulsarPackageImport(ctx context.Context, d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	packageName, err := utils.GetPackageName(d.Id())
	if err != nil {
		return nil, err
	}

	_ = d.Set(resourcePackageTypeKey, packageName.GetType().String())
	_ = d.Set(resourcePackageTenantKey, packageName.GetTenant())
	_ = d.Set(resourcePackageNamespaceKey, packageName.GetNamespace())
	_ = d.Set(resourcePackageNameKey, packageName.GetName())
	_ = d.Set(resourcePackageVersionKey, packageName.GetVersion())

	diags := resourcePulsarPackageRead(ctx, d, meta)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to import %s: %s", d.Id(), diags[0].Summary)
	}

	return []*schema.ResourceData{d}, nil
}

func validatePackageType(val interface{}, key string) (warns []string, errs []error) {
	value, ok := val.(string)
	if !ok {
		errs = append(errs, fmt.Errorf("%s must be a string", key))
		return
	}

	switch value {
	case utils.PackageTypeFunction.String(), utils.PackageTypeSink.String(), utils.PackageTypeSource.String():
		return
	default:
		errs = append(errs, fmt.Errorf("%s must be one of %q, %q, or %q",
			key, utils.PackageTypeFunction, utils.PackageTypeSink, utils.PackageTypeSource))
		return
	}
}

func ensurePackageID(d *schema.ResourceData) (string, error) {
	if d.Id() != "" {
		return d.Id(), nil
	}
	return packageURLFromResourceData(d)
}

func packageURLFromResourceData(d *schema.ResourceData) (string, error) {
	packageType := utils.PackageType(d.Get(resourcePackageTypeKey).(string))
	tenant := d.Get(resourcePackageTenantKey).(string)
	namespace := d.Get(resourcePackageNamespaceKey).(string)
	name := d.Get(resourcePackageNameKey).(string)
	version := d.Get(resourcePackageVersionKey).(string)

	packageName, err := utils.GetPackageNameWithComponents(packageType, tenant, namespace, name, version)
	if err != nil {
		return "", err
	}

	return packageName.String(), nil
}

func buildPackageMetadataProperties(d *schema.ResourceData, info *packageFileInfo) map[string]string {
	props := make(map[string]string)
	if raw, ok := d.GetOk(resourcePackagePropertiesKey); ok && raw != nil {
		for k, v := range raw.(map[string]interface{}) {
			props[k] = v.(string)
		}
	}

	props[packageMetadataPropertyTenant] = d.Get(resourcePackageTenantKey).(string)
	props[packageMetadataPropertyNamespace] = d.Get(resourcePackageNamespaceKey).(string)
	props[packageMetadataPropertyName] = d.Get(resourcePackageNameKey).(string)
	props[packageMetadataPropertyManagedBy] = packageMetadataManagedByTerraform

	if info == nil {
		info = existingPackageFileInfoFromState(d)
	}

	if info != nil {
		props[packageMetadataPropertyFileName] = info.name
		props[packageMetadataPropertyFileSize] = strconv.FormatInt(info.size, 10)
		props[packageMetadataPropertyChecksum] = info.checksum
	}

	return props
}

func existingPackageFileInfoFromState(d *schema.ResourceData) *packageFileInfo {
	name := d.Get(resourcePackageFileNameKey).(string)
	checksum := d.Get(resourcePackageFileChecksumKey).(string)

	var size int64
	if sizeVal, ok := d.Get(resourcePackageFileSizeKey).(int); ok {
		size = int64(sizeVal)
	}

	if name == "" && checksum == "" && size == 0 {
		return nil
	}

	return &packageFileInfo{
		name:     name,
		checksum: checksum,
		size:     size,
	}
}

func calculatePackageFileInfo(path string) (packageFileInfo, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return packageFileInfo{}, err
	}

	defer file.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return packageFileInfo{}, err
	}

	return packageFileInfo{
		checksum: hex.EncodeToString(hasher.Sum(nil)),
		size:     size,
		name:     filepath.Base(path),
	}, nil
}

func filterUserManagedProperties(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}

	out := make(map[string]string)
	for key, value := range in {
		if _, reserved := reservedPackageMetadataKeys[key]; reserved {
			continue
		}
		out[key] = value
	}

	return out
}

func convertToInterfaceMap(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func setLocalSourceChecksum(ctx context.Context, d *schema.ResourceData) {
	path := d.Get(resourcePackagePathKey).(string)
	if path == "" {
		return
	}

	info, err := calculatePackageFileInfo(path)
	if err != nil {
		tflog.Warn(ctx, "unable to read local package file for checksum", map[string]interface{}{
			"path": path, "error": err.Error(),
		})
		return
	}

	_ = d.Set(resourcePackageSourceChecksumKey, info.checksum)
}
