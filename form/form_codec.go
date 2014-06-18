// The form package contains our codec for
// application/vnd.brewnet.form.  When requesting this codec, you
// should include an additional sub-codec to use for converting to
// text - our codec type only ensures a specific structure to the
// response.  You probably want to use
// "application/vnd.brewnet.form+json" in your Accept header, although
// many other base types are supported.
package form

import (
	"errors"
	"github.com/stretchr/objx"
	"reflect"
	"strings"
)

const (
	mimeCategory    = "application"
	mimeName        = "vnd.brewnet.form"
	baseMime        = mimeCategory + "/" + mimeName
	defaultFullMime = baseMime + "+json"
	defaultSubMime  = "application/json"
)

type Pather interface {
	Path() string
}

type Receiver interface {
	Receive(interface{}) error
	FormFields(name string) objx.Map
}

// BrewnetFormCodec supports Marshaling and Unmarshaling instructions
// for form creation.  You can Marshal any value and it will be turned
// into information about the form that one should use for allowing
// user input for a value of the given type, or Unmarshal form data in
// a request to retrieve a set of input values.
//
// Any data using this MIMEtype will have some basic information, like
// the URL to send data to and the method to use, as values at the
// base level. It will also have a value at the top level that
// contains details used to create fields within the form (e.g. <input
// type="text" id="thing"/>). The key of each field will be the name
// to use when sending form data. Each field will specify the label
// and input type to use, at least. Fields may also include details
// about the contents, e.g. a regexp of accepted values, or whether or
// not some value is required (i.e. must not be empty/null).
//
// Additional documentation for how to parse field types (in HTML notation):
//
//    "text": <input type="text">
//    "password": <input type="password">
//    "dropdown": <select>
//        This type will have a sub-element named "options", which you can use to add <option> elements to the <select> element.
//    "radio":
//        This type will have a sub-element named "options", the same as the "dropdown" type.
//
type BrewnetFormCodec struct{}

func (codec *BrewnetFormCodec) Example() interface{} {
	return objx.Map{
		"action": "https://path/to/endpoint",
		"method": "POST",
		"fields": objx.Map{
			"name": objx.Map{
				"label":    "Name",
				"required": true,
				"type":     "text",
			},
			"address.street1": objx.Map{
				"label":    "Address Line 1",
				"required": false,
				"type":     "text",
			},
			"address.street2": objx.Map{
				"label":    "Address Line 2",
				"required": false,
				"type":     "text",
			},
		},
	}
}

func (codec *BrewnetFormCodec) ContentType() string {
	return defaultFullMime
}

func (codec *BrewnetFormCodec) FileExtension() string {
	return ".brewform"
}

func (codec *BrewnetFormCodec) CanMarshalWithCallback() bool {
	return true
}

func (codec *BrewnetFormCodec) ContentTypeSupported(contentType string) bool {
	if index := strings.IndexRune(contentType, '+'); index != -1 {
		contentType = contentType[:index]
	}
	return contentType == codec.ContentType()
}

func (codec *BrewnetFormCodec) Unmarshal(data []byte, obj interface{}) error {
	return errors.New("Unmarshal is currently a stub")
}

// Marshal takes a target object and returns a []byte representing the
// form that you should use for taking user input for the target
// object.
func (codec *BrewnetFormCodec) Marshal(object interface{}, optionsMSI map[string]interface{}) ([]byte, error) {
	options := objx.Map(optionsMSI)
	src := objx.Map{
		"action": options.Get("target").Str(),
		"method": options.Get("http-method").Str("POST"),
	}
	if pather, ok := object.(Pather); ok {
		src["action"] = pather.Path()
	}
	objType := reflect.TypeOf(object)
	for objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	if objType.Kind() == reflect.Struct {
		src["fields"] = codec.marshalStructFields("", objType, options)
	} else if receiver, ok := object.(Receiver); ok {
		src["fields"] = receiver.FormFields(options.Get("name").Str())
	} else {
		src["fields"] = objx.Map{
			options.Get("name").Str(): objx.Map{
				"label":    options.Get("label").Str(),
				"required": options.Get("required").Bool(true),
				"type":     codec.FormFieldType(objType),
			},
		}
	}
	// This is currently a stub
	return nil, errors.New("Marshal is currently a stub")
}

func (codec *BrewnetFormCodec) marshalStructFields(prefix string, objType reflect.Type, options map[string]interface{}) objx.Map {
	fields := make(objx.Map)
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		if field.Anonymous {
			anonFields := codec.marshalStructFields(prefix, field.Type, options)

			// Make sure fields from the embedded struct are
			// overwritten by fields already loaded from the embedding
			// struct.
			anonFields.MergeHere(fields)
			fields = anonFields
			continue
		}
		name, tagOptions := codec.fieldNameAndOptions(field)
		if name == "-" {
			continue
		}
		name = prefix + name
		if receiver, ok := reflect.Zero(objType).Interface().(Receiver); ok {
			fields.MergeHere(receiver.FormFields(name))
			continue
		}
		fieldType := field.Type
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			fields.MergeHere(codec.marshalStructFields(name+".", fieldType, options))
			continue
		}
		fieldMap := make(objx.Map)
		if label := tagOptions.Get("label").Str(); label != "" {
			fieldMap["label"] = label
		} else {
			fieldMap["label"] = strings.Title(name)
		}
		fieldMap["required"] = tagOptions.Get("required").Bool(false)
		fieldMap["type"] = codec.FormFieldType(fieldType)
		fields[name] = fieldMap
	}
	return fields
}

func (codec *BrewnetFormCodec) fieldNameAndOptions(field reflect.StructField) (string, objx.Map) {
	tag := field.Tag.Get("request")
	end := strings.IndexRune(tag, ',')
	if end < 0 {
		end = len(tag)
	}
	name := tag[:end]
	options := make(objx.Map)
	for end < len(tag) {
		tag = tag[end+1:]
		end = strings.IndexRune(tag, ',')
		if end < 0 {
			end = len(tag)
		}
		optionStr := tag[:end]
		split := strings.IndexRune(optionStr, '=')
		if split < 0 {
			options.Set(optionStr, true)
		} else {
			key := optionStr[:split]
			value := optionStr[split+1:]
			options.Set(key, value)
		}
	}

	if name == "" {
		possibleNames := []string{
			field.Tag.Get("response"),
			field.Tag.Get("db"),
			strings.ToLower(field.Name),
		}
		for _, possibleName := range possibleNames {
			if possibleName == "" {
				continue
			}
			end := strings.IndexRune(possibleName, ',')
			if end < 0 {
				end = len(possibleName)
			}
			name = possibleName[:end]

			// Only ignore the field if the request tag is "-"
			if name != "" && name != "-" {
				break
			}
		}
	}

	return name, options
}

func (codec *BrewnetFormCodec) FormFieldType(objType reflect.Type) string {
	var inputType string
	switch objType.Kind() {
	case reflect.Bool:
		inputType = "checkbox"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fallthrough
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fallthrough
	case reflect.Float32, reflect.Float64:
		inputType = "number"
	case reflect.Complex64, reflect.Complex128:
		panic("Can't handle complex types yet.")
	case reflect.Array, reflect.Slice:
		panic("Can't handle array or slice types yet")
	case reflect.Chan, reflect.Func:
		panic("Can't handle channel or func types yet")
	case reflect.Map:
		panic("Can't handle map types yet")
	case reflect.String:
		inputType = "text"
	default:
		panic("Unsupported field type for forms")
	}
	return inputType
}
