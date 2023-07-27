
provider "pulsar" {
  web_service_url = "http://localhost:8080"
  api_version = "3"
}

resource "pulsar_function" "function-1" {
    provider = pulsar

    name = "function-1"
    tenant = "public"
    namespace = "default"
    parallelism = 1

    processing_guarantees = "ATLEAST_ONCE"

    jar = "function://public/default/api-examples@v1"
    classname = "org.apache.pulsar.functions.api.examples.WordCountFunction"

    inputs = ["public/default/input1", "public/default/input2"]

    output = "public/default/test-out"

    subscription_name = "tf-sub"
    subscription_position = "Latest"
    cleanup_subscription = true
    skip_to_latest = true
    forward_source_message_property = true
    retain_key_ordering = true
    auto_ack = true
    max_message_retries = 101
    dead_letter_topic = "public/default/dlt"
    log_topic = "public/default/lt"
    timeout_ms = 6666

    custom_runtime_options = jsonencode(
    {
        "env": {
            "PULSAR": "FUNCTIONS"
        }
    })
}