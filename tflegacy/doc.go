// Package tflegacy is the final resting place for various legacy Terraform
// types and associated functions that used to live in Terraform packages like
// "helper/schema", "helper/resource", and "terraform" itself, so that
// resource type implementations written in terms of these can be fixed up just
// by simple package selector rewriting (replace existing imports with this
// tflegacy package) and then a provider can be adapted to the new SDK one
// resource type at a time.
package tflegacy
