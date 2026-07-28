// SPDX-License-Identifier: MPL-2.0

//go:build tools

// Package tools pins CLI tool dependencies for docs generation.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
