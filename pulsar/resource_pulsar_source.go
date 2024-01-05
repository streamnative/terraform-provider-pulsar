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

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"

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
	resourceSourceCustomRuntimeOptionsKey     = "custom_runtime_options"
	resourceSourceSchemaTypeKey               = "schema_type"
	resourceSourceSecretsKey                  = "secrets"
	// producer config
	resourceSourcePCMaxPendingMsgKey                = "max_pending_messages"
	resourceSourcePCMaxPendingMsgAcrossPartitionKey = "max_pending_messages_across_partitions"
	resourceSourcePCUseThreadLocalProducersKey      = "use_thread_local_producers"
	resourceSourcePCBatchBuilderKey                 = "batch_builder"
	resourceSourcePCCompressionTypeKey              = "compression_type"
	// producer crypto config
	resourceSourcePCCryptoKeyReaderClassNameKey     = "crypto_key_reader_classname"
	resourceSourcePCCryptoKeyReaderConfigKey        = "crypto_key_reader_config"
	resourceSourcePCEncryptionKeysKey               = "encryption_keys"
	resourceSourcePCProducerCryptoFailureActionKey  = "producer_crypto_failure_action"
	resourceSourcePCSConsumerCryptoFailureActionKey = "consumer_crypto_failure_action"
)

var resourceSourceDescriptions = make(map[string]string)

func init() {
	//nolint:lll
	resourceSourceDescriptions = map[string]string{
		resourceSourceTenantKey:                         "The source's tenant",
		resourceSourceNamespaceKey:                      "The source's namespace",
		resourceSourceNameKey:                           "The source's name",
		resourceSourceArchiveKey:                        "The path to the NAR archive for the Source. It also supports url-path [http/https/file (file protocol assumes that file already exists on worker host)] from which worker can download the package",
		resourceSourceProcessingGuaranteesKey:           "Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE)",
		resourceSourceDestinationTopicNamesKey:          "The Pulsar topic to which data is sent",
		resourceSourceDeserializationClassnameKey:       "The SerDe classname for the source",
		resourceSourceParallelismKey:                    "The source's parallelism factor",
		resourceSourceClassnameKey:                      "The source's class name if archive is file-url-path (file://)",
		resourceSourceCPUKey:                            "The CPU that needs to be allocated per source instance (applicable only to Docker runtime)",
		resourceSourceRAMKey:                            "The RAM that need to be allocated per source instance (applicable only to the process and Docker runtimes)",
		resourceSourceDiskKey:                           "The disk that need to be allocated per source instance (applicable only to Docker runtime)",
		resourceSourceConfigsKey:                        "User defined configs key/values (JSON string)",
		resourceSourceRuntimeFlagsKey:                   "User defined configs key/values (JSON string)",
		resourceSourceCustomRuntimeOptionsKey:           "A string that encodes options to customize the runtime, see docs for configured runtime for details",
		resourceSourceSchemaTypeKey:                     "The schema type (either a builtin schema like 'avro', 'json', etc.. or custom Schema class name to be used to encode messages emitted from the source",
		resourceSourceSecretsKey:                        "The map of secretName to an object that encapsulates how the secret is fetched by the underlying secrets provider",
		resourceSourcePCMaxPendingMsgKey:                "The maximum size of a queue holding pending messages",
		resourceSourcePCMaxPendingMsgAcrossPartitionKey: "The maximum number of pending messages across partitions",
		resourceSourcePCUseThreadLocalProducersKey:      "Whether to use thread local producers",
		resourceSourcePCBatchBuilderKey:                 "BatchBuilder provides two types of batch construction methods, DEFAULT and KEY_BASED.",
		resourceSourcePCCompressionTypeKey:              "Set the compression type for the producer. By default, message payloads are not compressed. Supported compression types are: LZ4, ZLIB, ZSTD, SNAPPY and NONE",
		resourceSourcePCCryptoKeyReaderClassNameKey:     "The classname for the crypto key reader that can be used to access the keys in the keystore",
		resourceSourcePCCryptoKeyReaderConfigKey:        "The config for the crypto key reader that can be used to access the keys in the keystore",
		resourceSourcePCEncryptionKeysKey:               "One or more public keys to encrypt data key. It can be used to encrypt data key with multiple keys.",
		resourceSourcePCProducerCryptoFailureActionKey:  "The desired action if producer fail to encrypt data, one of FAIL, SEND",
		resourceSourcePCSConsumerCryptoFailureActionKey: "The desired action if consumer fail to decrypt data, one of FAIL, DISCARD, CONSUME",
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
				Computed:    true,
				Description: resourceSourceDescriptions[resourceSourceCPUKey],
			},
			resourceSourceRAMKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: resourceSourceDescriptions[resourceSourceRAMKey],
			},
			resourceSourceDiskKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: resourceSourceDescriptions[resourceSourceDiskKey],
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
			resourceSourceCustomRuntimeOptionsKey: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  resourceSourceDescriptions[resourceSourceCustomRuntimeOptionsKey],
				ValidateFunc: jsonValidateFunc,
			},
			resourceSourceSchemaTypeKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourceSchemaTypeKey],
			},
			resourceSourceSecretsKey: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  resourceSourceDescriptions[resourceSourceSecretsKey],
				ValidateFunc: jsonValidateFunc,
			},
			resourceSourcePCMaxPendingMsgKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCMaxPendingMsgKey],
			},
			resourceSourcePCMaxPendingMsgAcrossPartitionKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCMaxPendingMsgAcrossPartitionKey],
			},
			resourceSourcePCUseThreadLocalProducersKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCUseThreadLocalProducersKey],
			},
			resourceSourcePCBatchBuilderKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCBatchBuilderKey],
			},
			resourceSourcePCCompressionTypeKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCCompressionTypeKey],
			},
			resourceSourcePCCryptoKeyReaderClassNameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCCryptoKeyReaderClassNameKey],
			},
			resourceSourcePCCryptoKeyReaderConfigKey: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  resourceSourceDescriptions[resourceSourcePCCryptoKeyReaderConfigKey],
				ValidateFunc: jsonValidateFunc,
			},
			resourceSourcePCEncryptionKeysKey: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCEncryptionKeysKey],
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			resourceSourcePCProducerCryptoFailureActionKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCProducerCryptoFailureActionKey],
			},
			resourceSourcePCSConsumerCryptoFailureActionKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSourceDescriptions[resourceSourcePCSConsumerCryptoFailureActionKey],
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

	if sourceConfig.CustomRuntimeOptions != "" {
		orig, ok := d.GetOk(resourceSourceCustomRuntimeOptionsKey)
		if ok {
			s, err := ignoreServerSetCustomRuntimeOptions(orig.(string), sourceConfig.CustomRuntimeOptions)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set(resourceSourceCustomRuntimeOptionsKey, s)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if len(sourceConfig.SchemaType) != 0 {
		err = d.Set(resourceSourceSchemaTypeKey, sourceConfig.SchemaType)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(sourceConfig.Secrets) != 0 {
		s, err := json.Marshal(sourceConfig.Secrets)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "cannot marshal secrets from sourceConfig"))
		}
		err = d.Set(resourceSourceSecretsKey, string(s))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if sourceConfig.ProducerConfig != nil {
		if sourceConfig.ProducerConfig.MaxPendingMessages > 0 {
			err = d.Set(resourceSourcePCMaxPendingMsgKey, sourceConfig.ProducerConfig.MaxPendingMessages)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		if sourceConfig.ProducerConfig.MaxPendingMessagesAcrossPartitions > 0 {
			err = d.Set(resourceSourcePCMaxPendingMsgAcrossPartitionKey,
				sourceConfig.ProducerConfig.MaxPendingMessagesAcrossPartitions)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		err = d.Set(resourceSourcePCUseThreadLocalProducersKey, sourceConfig.ProducerConfig.UseThreadLocalProducers)
		if err != nil {
			return diag.FromErr(err)
		}

		if len(sourceConfig.ProducerConfig.BatchBuilder) != 0 {
			err = d.Set(resourceSourcePCBatchBuilderKey, sourceConfig.ProducerConfig.BatchBuilder)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		if len(sourceConfig.ProducerConfig.CompressionType) != 0 {
			err = d.Set(resourceSourcePCCompressionTypeKey, sourceConfig.ProducerConfig.CompressionType)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		if sourceConfig.ProducerConfig.CryptoConfig != nil {
			cryptoConfig := sourceConfig.ProducerConfig.CryptoConfig

			err = d.Set(resourceSourcePCCryptoKeyReaderClassNameKey, cryptoConfig.CryptoKeyReaderClassName)
			if err != nil {
				return diag.FromErr(err)
			}

			if len(cryptoConfig.CryptoKeyReaderConfig) != 0 {
				c, err := json.Marshal(cryptoConfig.CryptoKeyReaderConfig)
				if err != nil {
					return diag.FromErr(errors.Wrap(err, "cannot marshal config from crypto key reader"))
				}
				err = d.Set(resourceSourcePCCryptoKeyReaderConfigKey, string(c))
				if err != nil {
					return diag.FromErr(err)
				}
			}

			if len(cryptoConfig.EncryptionKeys) != 0 {
				err = d.Set(resourceSourcePCEncryptionKeysKey, cryptoConfig.EncryptionKeys)
				if err != nil {
					return diag.FromErr(err)
				}
			}

			if len(cryptoConfig.ProducerCryptoFailureAction) != 0 {
				err = d.Set(resourceSourcePCProducerCryptoFailureActionKey, cryptoConfig.ProducerCryptoFailureAction)
				if err != nil {
					return diag.FromErr(err)
				}
			}

			if len(cryptoConfig.ConsumerCryptoFailureAction) != 0 {
				err = d.Set(resourceSourcePCSConsumerCryptoFailureActionKey, cryptoConfig.ConsumerCryptoFailureAction)
				if err != nil {
					return diag.FromErr(err)
				}
			}
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

	if inter, ok := d.GetOk(resourceSourceCustomRuntimeOptionsKey); ok {
		sourceConfig.CustomRuntimeOptions = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceSchemaTypeKey); ok {
		sourceConfig.SchemaType = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourceSecretsKey); ok {
		var secrets map[string]interface{}
		secretsJSON := inter.(string)

		err := json.Unmarshal([]byte(secretsJSON), &secrets)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal the secrets: %s", secretsJSON)
		}

		sourceConfig.Secrets = secrets
	}

	producerConfig := &utils.ProducerConfig{}
	if inter, ok := d.GetOk(resourceSourcePCMaxPendingMsgKey); ok {
		producerConfig.MaxPendingMessages = inter.(int)
	}

	if inter, ok := d.GetOk(resourceSourcePCMaxPendingMsgAcrossPartitionKey); ok {
		producerConfig.MaxPendingMessagesAcrossPartitions = inter.(int)
	}

	if inter, ok := d.GetOk(resourceSourcePCUseThreadLocalProducersKey); ok {
		producerConfig.UseThreadLocalProducers = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceSourcePCBatchBuilderKey); ok {
		producerConfig.BatchBuilder = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourcePCCompressionTypeKey); ok {
		producerConfig.CompressionType = inter.(string)
	}

	cryptoConfig := &utils.CryptoConfig{}
	if inter, ok := d.GetOk(resourceSourcePCCryptoKeyReaderClassNameKey); ok {
		cryptoConfig.CryptoKeyReaderClassName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourcePCCryptoKeyReaderConfigKey); ok {
		var cryptoKeyReaderConfig map[string]interface{}
		cryptoKeyReaderConfigJSON := inter.(string)

		err := json.Unmarshal([]byte(cryptoKeyReaderConfigJSON), &cryptoKeyReaderConfig)
		if err != nil {
			return nil, errors.Wrapf(err,
				"cannot unmarshal the cryptoKeyReaderConfig: %s", cryptoKeyReaderConfigJSON)
		}

		cryptoConfig.CryptoKeyReaderConfig = cryptoKeyReaderConfig
	}

	if inter, ok := d.GetOk(resourceSourcePCEncryptionKeysKey); ok {
		encryptionKeysSet := inter.(*schema.Set)
		var encryptionKeys []string

		for _, item := range encryptionKeysSet.List() {
			encryptionKeys = append(encryptionKeys, item.(string))
		}

		cryptoConfig.EncryptionKeys = encryptionKeys
	}

	if inter, ok := d.GetOk(resourceSourcePCProducerCryptoFailureActionKey); ok {
		cryptoConfig.ProducerCryptoFailureAction = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSourcePCSConsumerCryptoFailureActionKey); ok {
		cryptoConfig.ConsumerCryptoFailureAction = inter.(string)
	}

	producerConfig.CryptoConfig = cryptoConfig
	sourceConfig.ProducerConfig = producerConfig

	return sourceConfig, nil
}
