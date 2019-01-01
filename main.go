package main

import (
	"github.com/edahlseng/terraform-provider-repository-template/repository-template"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: repositoryTemplate.Provider})
}
