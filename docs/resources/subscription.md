---
page_title: "Pulsar: pulsar_subscription"
subcategory: ""
description: |-
  Manages a Pulsar subscription.
---

# pulsar_subscription

Manages a Pulsar subscription in Apache Pulsar.

## Example Usage

```terraform
resource "pulsar_tenant" "example" {
  tenant = "example"
  allowed_clusters = ["standalone"]
}

resource "pulsar_namespace" "example" {
  tenant    = pulsar_tenant.example.tenant
  namespace = "example-ns"
}

resource "pulsar_topic" "example" {
  tenant     = pulsar_tenant.example.tenant
  namespace  = pulsar_namespace.example.namespace
  topic_type = "persistent"
  topic_name = "example-topic"
  partitions = 3
}

resource "pulsar_subscription" "example" {
  topic_name        = "${pulsar_topic.example.topic_type}://${pulsar_tenant.example.tenant}/${pulsar_namespace.example.namespace}/${pulsar_topic.example.topic_name}"
  subscription_name = "example-subscription"
  position          = "latest" # Can be "earliest" or "latest"
}
```

## Argument Reference

The following arguments are supported:

- `topic_name` (Required) - The topic in the format of `{topic_type}://{tenant}/{namespace}/{topic_name}`.
- `subscription_name` (Required) - The subscription name.
- `position` (Optional) - The initial position (earliest or latest). Default is "latest".

## Attribute Reference

No additional attributes are exported.

## Import

Subscriptions can be imported using the format `{topic}:{subscription_name}`, e.g.

```
$ terraform import pulsar_subscription.example persistent://tenant/namespace/topic:my-subscription
``` 