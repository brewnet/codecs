package form

// An InputTyper can provide details about the type of data it expects
// from input in a request.
type InputTyper interface {
	InputType() interface{}
}
