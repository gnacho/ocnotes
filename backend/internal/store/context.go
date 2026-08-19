package store

// context tracks shadow-user operations shared by all store consumers.
type context struct{}

func newContext() *context { return &context{} }
