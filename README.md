codecs
======

This repository contains codecs for encoding and decoding the vendor
extension mimetypes that we use for our API.  This is the Go
repository, and codecs should be imported into Go code from this
repository directly.  We may provide support for other languages in
one of our other repositories, or you can feel free to clone the
functionality in this library for whatever language you need.

These codecs are designed to be used with
[stretchr's codec system](https://github.com/stretchr/codecs).  Add
them to a CodecService (e.g. goweb.DefaultWebCodecService) with
`codecService.AddCodec(codec)` where codec is a pointer to an instance
of one of our codec structs.

See
[the documentation at godoc.org](http://godoc.org/github.com/brewnet/codecs)
for details on what sorts of assumptions our different types of codecs
allow you to make, and what sorts of data they return.
