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
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestFilterUserManagedProperties(t *testing.T) {
	props := map[string]string{
		packageMetadataPropertyTenant:   "tenant",
		packageMetadataPropertyFileName: "pkg.nar",
		"custom":                        "value",
	}

	filtered := filterUserManagedProperties(props)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 property, got %d", len(filtered))
	}

	if filtered["custom"] != "value" {
		t.Fatalf("expected custom property to be kept")
	}
}

func TestBuildPackageMetadataProperties(t *testing.T) {
	resource := resourcePulsarPackage()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		resourcePackageTypeKey:       "function",
		resourcePackageTenantKey:     "tenant-a",
		resourcePackageNamespaceKey:  "ns",
		resourcePackageNameKey:       "pkg",
		resourcePackageVersionKey:    "v1",
		resourcePackagePathKey:       "/tmp/pkg.nar",
		resourcePackagePropertiesKey: map[string]interface{}{"custom": "value"},
	})

	info := &packageFileInfo{
		checksum: "abc",
		size:     42,
		name:     "pkg.nar",
	}

	props := buildPackageMetadataProperties(data, info)

	expected := map[string]string{
		packageMetadataPropertyTenant:    "tenant-a",
		packageMetadataPropertyNamespace: "ns",
		packageMetadataPropertyName:      "pkg",
		packageMetadataPropertyFileName:  "pkg.nar",
		packageMetadataPropertyFileSize:  "42",
		packageMetadataPropertyChecksum:  "abc",
		packageMetadataPropertyManagedBy: packageMetadataManagedByTerraform,
		"custom":                         "value",
	}

	for k, v := range expected {
		if props[k] != v {
			t.Fatalf("expected %s=%s, got %s", k, v, props[k])
		}
	}
}

func TestBuildPackageMetadataPropertiesUsesExistingValues(t *testing.T) {
	resource := resourcePulsarPackage()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		resourcePackageTypeKey:      "function",
		resourcePackageTenantKey:    "tenant-a",
		resourcePackageNamespaceKey: "ns",
		resourcePackageNameKey:      "pkg",
		resourcePackageVersionKey:   "v1",
		resourcePackagePathKey:      "/tmp/pkg.nar",
	})

	_ = data.Set(resourcePackageFileNameKey, "existing.nar")
	_ = data.Set(resourcePackageFileSizeKey, 100)
	_ = data.Set(resourcePackageFileChecksumKey, "old")

	props := buildPackageMetadataProperties(data, nil)

	if props[packageMetadataPropertyFileName] != "existing.nar" {
		t.Fatalf("expected existing file name to be reused, got %s", props[packageMetadataPropertyFileName])
	}

	if props[packageMetadataPropertyFileSize] != "100" {
		t.Fatalf("expected existing file size to be reused, got %s", props[packageMetadataPropertyFileSize])
	}

	if props[packageMetadataPropertyChecksum] != "old" {
		t.Fatalf("expected existing checksum to be reused, got %s", props[packageMetadataPropertyChecksum])
	}
}

func TestExistingPackageFileInfoFromState(t *testing.T) {
	resource := resourcePulsarPackage()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		resourcePackageTypeKey:      "function",
		resourcePackageTenantKey:    "tenant-a",
		resourcePackageNamespaceKey: "ns",
		resourcePackageNameKey:      "pkg",
		resourcePackageVersionKey:   "v1",
		resourcePackagePathKey:      "/tmp/pkg.nar",
	})

	_ = data.Set(resourcePackageFileNameKey, "pkg.nar")
	_ = data.Set(resourcePackageFileSizeKey, 0)
	_ = data.Set(resourcePackageFileChecksumKey, "abc")

	info := existingPackageFileInfoFromState(data)
	if info == nil {
		t.Fatalf("expected file info")
	}

	if info.name != "pkg.nar" || info.checksum != "abc" || info.size != 0 {
		t.Fatalf("unexpected info %#v", info)
	}
}

func TestCalculatePackageFileInfo(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "pkg.nar")
	content := []byte("hello pulsar package")

	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	info, err := calculatePackageFileInfo(filePath)
	if err != nil {
		t.Fatalf("failed to calculate info: %v", err)
	}

	if info.size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), info.size)
	}

	if info.name != "pkg.nar" {
		t.Fatalf("expected name pkg.nar, got %s", info.name)
	}

	expectedChecksum := "032cc2f30d10d8598ae15a53bfbf4e93724d5c70a4b6dd8167bcda59fd955d8f"
	if info.checksum != expectedChecksum {
		t.Fatalf("expected checksum %s, got %s", expectedChecksum, info.checksum)
	}
}
