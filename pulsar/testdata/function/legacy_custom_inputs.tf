
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_function" "function-legacy-custom-inputs" {
  provider = pulsar

  name        = "function-legacy-custom-inputs"
  tenant      = "public"
  namespace   = "default"
  parallelism = 1

  processing_guarantees = "ATLEAST_ONCE"

  jar       = "function://public/default/api-examples@v1"
  classname = "org.apache.pulsar.functions.api.examples.WordCountFunction"

  custom_serde_inputs = {
    "public/default/legacy-serde-input" = "org.apache.pulsar.functions.api.utils.DefaultSerDe"
  }

  custom_schema_inputs = {
    "public/default/legacy-schema-input" = jsonencode({ schemaType = "STRING" })
  }

  output = "public/default/legacy-custom-output"
}
