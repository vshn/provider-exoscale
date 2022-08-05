//go:build tools

// Package tools is a place to put any tooling dependencies as imports.
// Go modules will be forced to download and install them.
package e2e

import (
	// Kuttl
	_ "github.com/kudobuilder/kuttl/cmd/kubectl-kuttl"
)
