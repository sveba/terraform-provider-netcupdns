package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/svetob/terraform-provider-netcupdns/internal/provider"
)

//go:generate terraform fmt -recursive ./examples/

// Run the docs generation tool
//go:generate go run -mod=mod github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name netcupdns

var (
	version string = "dev"
	commit  string = ""
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/svetob/netcupdns",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New, opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
