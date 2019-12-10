package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/streamnative/terraform-provider-pulsar/pulsar"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: pulsar.Provider,
	})
}
