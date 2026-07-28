// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/jhuiting/terraform-provider-chargebee/internal/provider"
)

// version is set by goreleaser at build time.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with debugging support")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/jhuiting/chargebee",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
