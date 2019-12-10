provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_tenant" "my_tenant" {
  tenant = "thanos"
  allowed_clusters = [
    "pulsar-cluster-1"]
}
