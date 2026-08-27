
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_function" "function-1" {
  provider = pulsar

  name        = "function-1"
  tenant      = "public"
  namespace   = "default"
  parallelism = 1

  processing_guarantees = "ATLEAST_ONCE"

  jar       = "function://public/default/api-examples@v1"
  classname = "org.apache.pulsar.functions.api.examples.WordCountFunction"

  inputs = ["public/default/input1", "public/default/input2"]

  topics_pattern = "public/default/pattern-.*"

  custom_serde_inputs = {
    "public/default/serde-input" = "org.apache.pulsar.functions.api.utils.DefaultSerDe"
  }

  custom_schema_inputs = {
    "public/default/schema-input" = jsonencode({ schemaType = "STRING" })
  }

  # input1 is deliberately listed in both inputs and input_specs: the provider must strip it from
  # inputs on the wire, or Pulsar's validateUpdate() folds it back in with a default
  # ConsumerConfig and discards the receiver queue size on every apply.
  input_specs {
    key                 = "public/default/input1"
    receiver_queue_size = 250
    schema_type         = "avro"
    consumer_properties = {
      application = "billing"
    }
  }

  input_specs {
    key                 = "public/default/pattern-.*"
    receiver_queue_size = 251
    is_regex_pattern    = true
  }

  input_specs {
    key                 = "public/default/serde-input"
    receiver_queue_size = 252
    serde_class_name    = "org.apache.pulsar.functions.api.utils.DefaultSerDe"
  }

  input_specs {
    key                 = "public/default/schema-input"
    receiver_queue_size = 253
    schema_type         = "STRING"
  }

  output = "public/default/test-out"

  subscription_name               = "tf-sub"
  subscription_position           = "Latest"
  cleanup_subscription            = true
  forward_source_message_property = true
  retain_key_ordering             = true
  auto_ack                        = true
  max_message_retries             = 101
  dead_letter_topic               = "public/default/dlt"
  log_topic                       = "public/default/lt"
  timeout_ms                      = 6666

  # Producer fields are Optional+Computed. Set reset values explicitly; omitting them preserves
  # broker values from the prior configuration.
  compression_type                       = "LZ4"
  batch_builder                          = "DEFAULT"
  max_pending_messages                   = 0
  max_pending_messages_across_partitions = 0
  use_thread_local_producers             = false

  custom_runtime_options = jsonencode(
    {
      "env" : {
        "PULSAR" : "FUNCTIONS"
      }
  })
}
