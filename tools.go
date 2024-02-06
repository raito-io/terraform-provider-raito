//go:build tools
// +build tools

package main

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"

	_ "github.com/raito-io/enumer"
)
