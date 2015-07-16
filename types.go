// Package codecs contains types that relate to brewnet codecs.  They
// can be used in a response structure to help the codec understand
// what it should be building - for example, including a Link type in
// the response will allow all brewnet codecs to format the link how
// they need to to support their specific MIME type.
package codecs

// A Link is a type that contains a link to another resource.
type Link struct {
	// Location is the location of the linked resource.
	Location string

	// Relationship is an identifier describing how this link relates
	// to the type that contains it.  The relationship should be all
	// lower case and have words separated with "-" characters.
	Relationship string

	// Title is a human-readable description of the resource linked
	// to and how it relates to the type that contains this link.  If
	// Title is empty, it can be generated using the Relationship
	// value, replacing "-" characters with " " characters and
	// converting the first word or all words (as needed) to title
	// case.
	Title string

	// Distance is a value between 0 and 1 that represents how distant
	// the relationship between the containing type and the linked
	// type is.  If the linked type should always be embedded in the
	// containing type's data, the distance should be 0.  If the
	// linked type should always be a simple link or button, never a
	// dropdown or embedded element, the distance should be 1.  It's
	// up to the UI to decide what the values in-between mean (i.e. a
	// value of 0.8 might result in a slide-in element in a
	// high-resolution web browser, but a link to another page in a
	// phone app).
	Distance float32
}

// An Image is a type that contains a link to an image.
type Image struct {
	Link

	// MIME is the MIME type of the linked image.
	MIME string
}

// A Title includes a string of characters and a rank.
type Title struct {
	// Value should be the title's string value.
	Value string

	// Rank should represent the title's importance to the containing
	// context.  A rank of 0 will be emphasized more than a rank of 1.
	Rank int
}
