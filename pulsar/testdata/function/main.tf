
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
    receiver_queue_size = 100
    schema_type         = "avro"
    consumer_properties = {
      application = "billing"
    }
  }

  # Each legacy input form is deliberately overlapped. input_specs must win on both create and
  # update, even though Pulsar's update path otherwise applies the legacy forms last.
  input_specs {
    key                 = "public/default/pattern-.*"
    receiver_queue_size = 101
    is_regex_pattern    = true
  }

  input_specs {
    key                 = "public/default/serde-input"
    receiver_queue_size = 102
    serde_class_name    = "org.apache.pulsar.functions.api.utils.DefaultSerDe"
  }

  input_specs {
    key                 = "public/default/schema-input"
    receiver_queue_size = 103
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

  # Output producer configuration (#220 part A).
  compression_type                       = "ZSTD"
  batch_builder                          = "KEY_BASED"
  max_pending_messages                   = 1000
  max_pending_messages_across_partitions = 50000
  use_thread_local_producers             = true

  custom_runtime_options = jsonencode(
    {
      "env" : {
        "PULSAR" : "FUNCTIONS"
      }
  })
}
