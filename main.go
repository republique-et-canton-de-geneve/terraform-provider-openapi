package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/provider"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with debugger support")
	flag.Parse()

	err := providerserver.Serve(
		context.Background(),
		provider.New(version),
		providerserver.ServeOpts{
			Address: "registry.terraform.io/republique-et-canton-de-geneve/openapi",
			Debug:   debug,
		},
	)
	if err != nil {
		log.Fatal(err.Error())
	}
}
