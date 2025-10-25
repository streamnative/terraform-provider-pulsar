## Simple String Schema
resource "pulsar_schema" "user_events" {
  tenant    = "my-tenant"
  namespace = "my-namespace"
  topic     = "user-events"
  type      = "STRING"

  properties = {
    "owner"       = "data-team"
    "description" = "User activity events"
  }
}

## AVRO Schema with Complex Structure
resource "pulsar_schema" "user_profile" {
  tenant    = "my-tenant"
  namespace = "my-namespace"
  topic     = "user-profile"
  type      = "AVRO"

  schema_data = jsonencode({
    type = "record"
    name = "UserProfile"
    fields = [
      {
        name = "id"
        type = "string"
      },
      {
        name = "email"
        type = "string"
      },
      {
        name = "age"
        type = ["null", "int"]
        default = null
      },
      {
        name = "preferences"
        type = {
          type = "record"
          name = "Preferences"
          fields = [
            {
              name = "theme"
              type = "string"
              default = "light"
            },
            {
              name = "notifications"
              type = "boolean"
              default = true
            }
          ]
        }
      }
    ]
  })

  properties = {
    "owner"   = "user-service"
    "version" = "1.0"
  }
}

## JSON Schema
resource "pulsar_schema" "order_events" {
  tenant    = "ecommerce"
  namespace = "orders"
  topic     = "order-created"
  type      = "JSON"

  schema_data = jsonencode({
    type = "object"
    properties = {
      orderId = {
        type = "string"
      }
      customerId = {
        type = "string"
      }
      amount = {
        type = "number"
        minimum = 0
      }
      items = {
        type = "array"
        items = {
          type = "object"
          properties = {
            productId = { type = "string" }
            quantity = { type = "integer", minimum = 1 }
            price = { type = "number", minimum = 0 }
          }
          required = ["productId", "quantity", "price"]
        }
      }
    }
    required = ["orderId", "customerId", "amount", "items"]
  })

  properties = {
    "service" = "order-service"
    "env"     = "production"
  }
}

## Protobuf Schema
resource "pulsar_schema" "metrics" {
  tenant    = "monitoring"
  namespace = "metrics"
  topic     = "system-metrics"
  type      = "PROTOBUF"

  schema_data = <<EOF
syntax = "proto3";

message SystemMetrics {
  string host = 1;
  int64 timestamp = 2;
  double cpu_usage = 3;
  double memory_usage = 4;
  double disk_usage = 5;
  map<string, string> labels = 6;
}
EOF

  properties = {
    "collector" = "prometheus"
    "interval"  = "30s"
  }
}