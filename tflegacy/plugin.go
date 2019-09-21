package tflegacy

import (
	oldplugin "github.com/hashicorp/terraform-plugin-sdk/plugin"
)

// ServeOpts describes how to serve a plugin (as a go-plugin server) in the
// legacy SDK.
type ServeOpts = oldplugin.ServeOpts

// Serve starts the go-plugin server for a given plugin and keeps serving it
// until the process is killed.
//
// This function never returns.
func Serve(opts *ServeOpts) {
	oldplugin.Serve(opts)
}

// ProviderFunc is a function that produces a provider implementation in the
// legacy SDK.
type ProviderFunc = oldplugin.ProviderFunc
