package tflegacy

// ResourceData is used to query and set the attributes of a resource.
//
// ResourceData is the primary argument received for CRUD operations on
// a resource as well as configuration of a provider. It is a powerful
// structure that can be used to not only query data, but check for changes,
// define partial state updates, etc.
//
// The most relevant methods to take a look at are Get, Set, and Partial.
type ResourceData struct {
	// Settable (internally)
	schema   map[string]*Schema
	config   *ResourceConfig
	state    *InstanceState
	diff     *InstanceDiff
	meta     map[string]interface{}
	timeouts *ResourceTimeout

	// Don't set
	//multiReader *MultiLevelFieldReader
	//setWriter   *MapFieldWriter
	//newState    *InstanceState
	//partial     bool
	//partialMap  map[string]struct{}
	//once        sync.Once
	//isNew       bool

	panicOnError bool
}

// getResult is the internal structure that is generated when a Get
// is called that contains some extra data that might be used.
type getResult struct {
	Value          interface{}
	ValueProcessed interface{}
	Computed       bool
	Exists         bool
	Schema         *Schema
}
