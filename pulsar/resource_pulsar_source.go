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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/streamnative/pulsar-admin-go/pkg/admin"
	"github.com/streamnative/pulsar-admin-go/pkg/rest"
	"github.com/streamnative/pulsar-admin-go/pkg/utils"

	"github.com/streamnative/terraform-provider-pulsar/bytesize"
)

const (
	resourceSourceTenantKey                   = "tenant"
	resourceSourceNamespaceKey                = "namespace"
	resourceSourceNameKey                     = "name"
	resourceSourceArchiveKey                  = "archive"
	resourceSourceProcessingGuaranteesKey     = "processing_guarantees"
	resourceSourceDestinationTopicNamesKey    = "destination_topic_name"
	resourceSourceDeserializationClassnameKey = "deserialization_classname"
	resourceSourceParallelismKey              = "parallelism"
	resourceSourceClassnameKey                = "classname"
	resourceSourceCPUKey                      = "cpu"
	resourceSourceRAMKey                      = "ram_mb"
	resourceSourceDiskKey                     = "disk_mb"
	resourceSourceConfigsKey                  = "configs"
	resourceSourceRuntimeFlagsKey             = "runtime_flags"

	ProcessingGuaranteesAtLeastOnce     = "ATLEAST_ONCE"
	ProcessingGuaranteesAtMostOnce      = "ATMOST_ONCE"
	ProcessingGuaranteesEffectivelyOnce = "EFFECTIVELY_ONCE"
)

var resourceSourceDescriptions = make(map[string]string)

func init() {
	//nolint:lll
	resourceSourceDescriptions = map[string]string{
		resourceSourceTenantKey:                   "The source's tenant",
		resourceSourceNamespaceKey:                "The source's namespace",
		resourceSourceNameKey:                     "The source's name",
		resourceSourceArchiveKey:                  "The path to the NAR archive for the Source. It also supports url-path [http/https/file (file protocol assumes that file already exists on worker host)] from which worker can download the package",
		resourceSourceProcessingGuaranteesKey:     "Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE)",
		resourceSourceDestinationTopicNamesKey:    "The Pulsar topic to which data is sent",
		resourceSourceDeserializationClassnameKey: "The SerDe classname for the source",
		resourceSourceParallelismKey:              "The source's parallelism factor",
		resourceSourceClassnameKey:                "The source's class name if archive is file-url-path (file://)",
		resourceSourceCPUKey:                      "The CPU that needs to be allocated per source instance (applicable only to Docker runtime)",
		resourceSourceRAMKey:                      "The RAM that need to be allocated per source instance (applicable only to the process and Docker runtimes)",
		resourceSourceDiskKey:                     "The disk that need to be allocated per source instance (applicable only to Docker runtime)",
		resourceSourceConfigsKey:                  "User defined configs key/values (JSON string)",
		resourceSourceRuntimeFlagsKey:             "User defined configs key/values (JSON string)",
	}
}

func resourcePulsarSource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarSourceCreate,
		ReadContext:   resourcePulsarSourceRead,
		UpdateContext: resourcePulsarSourceUpdate,
		DeleteContext: resourcePulsarSourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				id := d.Id()

				parts := strings.Split(id, "/")
				if len(parts) != 3 {
					return nil, errors.New("id should be tenant/namespace/name format")
				}

				_ = d.Set(resourceSourceTenantKey, parts[0])
				_ = d.Set(resourceSourceNamespaceKey, parts[1])
				_ = d.Set(resourceSourceNameKey, parts[2])

				diags := resourcePulsarSourceRead(ctx, d, meta)
				if diags.HasError() {
					return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
				}
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			resourceSourceTenantKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSourceDescriptions[resourceSourceTenantKey],
			},
			resourceSourceNamespaceKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSourceDescriptions[resourceSourceNamespaceKey],
			},
			resourceSourceNameKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSourceDescriptions[resourceSourceNameKey],
			},
			resourceSourceArchiveKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSourceDescriptions[resourceSourceArchiveKey],
			},
			resourceSourceProcessingGuaranteesKey: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ProcessingGuaranteesAtLeastOnce,
				ValidateFunc: func(val interface{}, key string) ([]string, []error) {
					v := val.(string)
					supported := []string{
						ProcessingGuaranteesAtLeastOnce,
						ProcessingGuaranteesAtMostOnce,
						ProcessingGuaranteesEffectivelyOnce,
					}

					found := false
					for _, item := range supported {
						if v == item {
							found = true
							break
						}
					}
					if !found {
						return nil, []error{
							fmt.Errorf("%s is unsupported, shold be one of %s", v,
								strings.Join(supported, ",")),
						}
					}

					return nil, nil
				},
				Description: resourceSourceDescriptions[resourceSourceProcessingGuaranteesKey],
			},
			resourceSourceDestinationTopicNamesKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSourceDescriptions[resourceSourceDestinationTopicNamesKey],
			},
			resourceSourceDeserializationClassnameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourceDeserializationClassnameKey],
			},
			resourceSourceParallelismKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: resourceSourceDescriptions[resourceSourceParallelismKey],
			},
			resourceSourceClassnameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: resourceSourceDescriptions[resourceSourceClassnameKey],
			},
			resourceSourceCPUKey: {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourceCPUKey],
				Default:     utils.NewDefaultResources().CPU,
			},
			resourceSourceRAMKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourceRAMKey],
				Default:     int(bytesize.FormBytes(uint64(utils.NewDefaultResources().RAM)).ToMegaBytes()),
			},
			resourceSourceDiskKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourceDiskKey],
				Default:     int(bytesize.FormBytes(uint64(utils.NewDefaultResources().Disk)).ToMegaBytes()),
			},
			resourceSourceConfigsKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: resourceSourceDescriptions[resourceSourceConfigsKey],
			},
			resourceSourceRuntimeFlagsKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourceRuntimeFlagsKey],
			},
		},
	}
}

func resourcePulsarSourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(admin.Client).Sources()

	sourceConfig, err := marshalSourceConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if isPackageURLSupported(sourceConfig.Archive) {
		err = client.CreateSourceWithURL(sourceConfig, sourceConfig.Archive)
	} else {
		err = client.CreateSource(sourceConfig, sourceConfig.Archive)
	}
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("@@@Create source: %v", err))
		return diag.Errorf("ERROR_CREATE_SOURCE: %v", err)
	}
	tflog.Debug(ctx, "@@@Create source complete")

	return resourcePulsarSourceRead(ctx, d, meta)
}

func resourcePulsarSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(admin.Client).Sources()

	tenant := d.Get(resourceSourceTenantKey).(string)
	namespace := d.Get(resourceSourceNamespaceKey).(string)
	name := d.Get(resourceSourceNameKey).(string)

	d.SetId(fmt.Sprintf("%s/%s/%s", tenant, namespace, name))

	sourceConfig, err := client.GetSource(tenant, namespace, name)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			return diag.Errorf("ERROR_SOURCE_NOT_FOUND")
		}
		return diag.FromErr(errors.Wrapf(err, "failed to get %s source from %s/%s", name, tenant, namespace))
	}

	// When the archive is built-in resource, it is not empty, otherwise it is empty.
	if sourceConfig.Archive != "" {
		err = d.Set(resourceSourceArchiveKey, sourceConfig.Archive)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set(resourceSourceProcessingGuaranteesKey, sourceConfig.ProcessingGuarantees)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(resourceSourceDestinationTopicNamesKey, sourceConfig.TopicName)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(sourceConfig.SerdeClassName) != 0 {
		err = d.Set(resourceSourceDeserializationClassnameKey, sourceConfig.SerdeClassName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set(resourceSourceParallelismKey, sourceConfig.Parallelism)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(resourceSourceClassnameKey, sourceConfig.ClassName)
	if err != nil {
		return diag.FromErr(err)
	}

	if sourceConfig.Resources != nil {
		err = d.Set(resourceSourceCPUKey, sourceConfig.Resources.CPU)
		if err != nil {
			return diag.FromErr(err)
		}

		err = d.Set(resourceSourceRAMKey, bytesize.FormBytes(uint64(sourceConfig.Resources.RAM)).ToMegaBytes())
		if err != nil {
			return diag.FromErr(err)
		}

		err = d.Set(resourceSourceDiskKey, bytesize.FormBytes(uint64(sourceConfig.Resources.Disk)).ToMegaBytes())
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(sourceConfig.Configs) != 0 {
		b, err := json.Marshal(sourceConfig.Configs)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "cannot marshal configs from sourceConfig"))
		}

		err = d.Set(resourceSourceConfigsKey, string(b))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(sourceConfig.RuntimeFlags) != 0 {
		err = d.Set(resourceSourceRuntimeFlagsKey, sourceConfig.RuntimeFlags)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourcePulsarSourceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(admin.Client).Sources()

	sourceConfig, err := marshalSourceConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOptions := utils.NewUpdateOptions()
	if isPackageURLSupported(sourceConfig.Archive) {
		err = client.UpdateSourceWithURL(sourceConfig, sourceConfig.Archive, updateOptions)
	} else {
		err = client.UpdateSource(sourceConfig, sourceConfig.Archive, updateOptions)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePulsarSourceRead(ctx, d, meta)
}

func resourcePulsarSourceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(admin.Client).Sources()

	tenant := d.Get(resourceSourceTenantKey).(string)
	namespace := d.Get(resourceSourceNamespaceKey).(string)
	name := d.Get(resourceSourceNameKey).(string)

	return diag.FromErr(client.DeleteSource(tenant, namespace, name))
}

func marshalSourceConfig(d *schema.ResourceData) (*utils.SourceConfig, error) {
	sourceConfig := &utils.SourceConfig{}

	if inter, ok := d.GetOk(resourceSourceTenantKey); ok {
		sourceConfig.Tenant = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceNamespaceKey); ok {
		sourceConfig.Namespace = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceNameKey); ok {
		sourceConfig.Name = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceArchiveKey); ok {
		pattern := inter.(string)
		sourceConfig.Archive = pattern
	}

	if inter, ok := d.GetOk(resourceSourceProcessingGuaranteesKey); ok {
		sourceConfig.ProcessingGuarantees = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceDestinationTopicNamesKey); ok {
		sourceConfig.TopicName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceDeserializationClassnameKey); ok {
		sourceConfig.SerdeClassName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceParallelismKey); ok {
		sourceConfig.Parallelism = inter.(int)
	}

	if inter, ok := d.GetOk(resourceSourceClassnameKey); ok {
		sourceConfig.ClassName = inter.(string)
	}

	resources := utils.NewDefaultResources()

	if inter, ok := d.GetOk(resourceSourceCPUKey); ok {
		value := inter.(float64)
		resources.CPU = value
	}

	if inter, ok := d.GetOk(resourceSourceRAMKey); ok {
		value := bytesize.FormMegaBytes(uint64(inter.(int))).ToBytes()
		resources.RAM = int64(value)
	}

	if inter, ok := d.GetOk(resourceSourceDiskKey); ok {
		value := bytesize.FormMegaBytes(uint64(inter.(int))).ToBytes()
		resources.Disk = int64(value)
	}

	sourceConfig.Resources = resources

	if inter, ok := d.GetOk(resourceSourceConfigsKey); ok {
		var configs map[string]interface{}
		configsJSON := inter.(string)

		err := json.Unmarshal([]byte(configsJSON), &configs)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal the configs: %s", configsJSON)
		}

		sourceConfig.Configs = configs
	}

	if inter, ok := d.GetOk(resourceSourceRuntimeFlagsKey); ok {
		sourceConfig.RuntimeFlags = inter.(string)
	}

	return sourceConfig, nil
}

func isPackageURLSupported(functionPkgURL string) bool {
	return strings.HasPrefix(functionPkgURL, "http://") ||
		strings.HasPrefix(functionPkgURL, "https://") ||
		strings.HasPrefix(functionPkgURL, "file://") ||
		strings.HasPrefix(functionPkgURL, "function://") ||
		strings.HasPrefix(functionPkgURL, "sink://") ||
		strings.HasPrefix(functionPkgURL, "source://")
}
