package codecs

// A Converter is any type that can convert itself to one of the types
// the brewnet codecs support.  This can be useful when you have
// resources as fields on other resources - as long as a resource
// implements Converter, its Convert() method will be called to get
// its type to use in the response.
//
// At the top level (e.g. when a type is either the whole data of a
// response or one of the elements of a top level slice), the
// Converter interface is ignored - it will only be used on elements
// contained by another resource.
type Converter interface {
	Convert() interface{}
}
