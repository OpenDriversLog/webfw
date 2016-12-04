package webfw

type IModel interface {
	// because of http://stackoverflow.com/questions/19554209/template-wont-evaluate-fields-that-are-interface-type-as-the-underlying-type
	// we need this "Custom" to get access the data that is extended by our individual models.
	C() interface{}
}
type Model struct {
}

// func C() is used adding new fields
func (Model) C() interface{} {
	return nil
}
