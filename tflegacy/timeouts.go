package tflegacy

import (
	oldsdk "github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	TimeoutsConfigKey = oldsdk.TimeoutsConfigKey
	TimeoutDefault    = oldsdk.TimeoutDefault
	TimeoutCreate     = oldsdk.TimeoutCreate
	TimeoutRead       = oldsdk.TimeoutRead
	TimeoutUpdate     = oldsdk.TimeoutUpdate
	TimeoutDelete     = oldsdk.TimeoutDelete
)
