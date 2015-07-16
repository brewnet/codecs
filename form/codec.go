// Package form contains our codec for application/vnd.brewnet.form.
// When requesting this codec, you should include an additional
// sub-codec to use for converting to text - our codec type only
// ensures a specific structure to the response.  You probably want to
// use "application/vnd.brewnet.form+json" in your Accept header,
// although that's not necessarily the only base type supported.
package form

import (
	"errors"
	"reflect"
	"strings"

	"github.com/nelsam/requests"
	"github.com/nelsam/silverback"
	"github.com/nelsam/silverback/codecs"
)

const (
	mimeType        = "application"
	mimeSubType     = "vnd.brewnet.form"
	defaultFullMime = BaseMIMEType + "+json"
	defaultSubMime  = "application/json"

	// BaseMIMEType contains the form codec's base MIME type.  Most of
	// the time, another type will be appended after a "+", to denote
	// the full format of the type.
	BaseMIMEType = mimeType + "/" + mimeSubType
)

// Codec supports Marshaling and Unmarshaling instructions for form
// creation.  You can Marshal any value and it will be turned into
// information about the form that one should use for allowing user
// input for a value of the given type, or Unmarshal form data in a
// request to retrieve a set of input values.
//
// Any data using this MIME type will have some basic information,
// like the URL to send data to and the method to use, as values at
// the base level. It will also have a value at the top level that
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
//    "selection": <select> or <radio>
//        This type will have a sub-element named "options", which you can use to add <option> elements to the <select> element.
type Codec struct {
	domain      string
	subCodec    silverback.Codec
	matchedType silverback.MIMEType
}

// NewCodec creates a new form codec for the provided domain.
func NewCodec(domain string) *Codec {
	return &Codec{domain: domain}
}

func (codec *Codec) New(matched silverback.MIMEType) silverback.Codec {
	if matched.Type != mimeType {
		return nil
	}
	newCodec := &Codec{
		domain: codec.domain,
	}
	switch matched.SubType {
	case "*", mimeSubType + "+json":
		newCodec.subCodec = new(codecs.JSON)
	default:
		return nil
	}
	return newCodec
}

func (codec *Codec) Types() []silverback.MIMEType {
	return []silverback.MIMEType{
		{
			Type:    mimeType,
			SubType: mimeSubType + "+json",
		},
	}
}

func (codec *Codec) Unmarshal(data []byte, obj interface{}) error {
	return errors.New("Unmarshal is currently a stub")
}

// Marshal takes a target object and returns a []byte representing the
// form that you should use for taking user input for the target
// object.
func (codec *Codec) Marshal(object interface{}) ([]byte, error) {
	domain := codec.domain
	if domain[len(domain)-1] == '/' {
		domain = domain[:len(domain)-1]
	}
	src := map[string]interface{}{
		"method": codec.matchedType.Options["method"],
	}
	if pather, ok := object.(Pather); ok {
		src["action"] = domain + "/" + pather.Path()
	}
	objType := reflect.TypeOf(object)
	for objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	if objType.Kind() != reflect.Struct {
		return nil, errors.New("Cannot marshal non-struct types to forms")
	}
	src["fields"] = codec.marshalStructFields("", objType)
	return codec.subCodec.Marshal(src)
}

func (codec *Codec) marshalStructFields(prefix string, objType reflect.Type) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		if field.Anonymous {
			anonFields := codec.marshalStructFields(prefix, field.Type)

			// Make sure fields from the embedded struct are
			// overwritten by fields already loaded from the embedding
			// struct.
			for key, val := range fields {
				anonFields[key] = val
			}
			fields = anonFields
			continue
		}
		name, tagOptions := codec.fieldNameAndOptions(field)
		if name == "-" {
			continue
		}
		name = prefix + name
		fieldType := field.Type
		receiveTyper, ok := reflect.Zero(fieldType).Interface().(requests.ReceiveTyper)
		if !ok {
			receiveTyper, ok = reflect.New(fieldType).Interface().(requests.ReceiveTyper)
		}
		if ok {
			fieldType = reflect.TypeOf(receiveTyper.ReceiveType())
			continue
		}
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			for newKey, newVal := range codec.marshalStructFields(name+".", fieldType) {
				fields[newKey] = newVal
			}
			continue
		}
		fieldMap := make(map[string]interface{})
		if label, ok := tagOptions["label"]; ok {
			fieldMap["label"] = label.(string)
		} else {
			title := strings.Title(strings.Replace(name, "_", " ", -1))
			fieldMap["label"] = title
		}
		fieldMap["required"] = false
		if required, ok := tagOptions["required"]; ok {
			fieldMap["required"] = required.(bool)
		}
		fieldMap["type"] = codec.FormFieldType(fieldType)
		fields[name] = fieldMap
	}
	return fields
}

func (codec *Codec) fieldNameAndOptions(field reflect.StructField) (string, map[string]interface{}) {
	tag := field.Tag.Get("request")
	end := strings.IndexRune(tag, ',')
	if end < 0 {
		end = len(tag)
	}
	name := tag[:end]
	options := make(map[string]interface{})
	for end < len(tag) {
		tag = tag[end+1:]
		end = strings.IndexRune(tag, ',')
		if end < 0 {
			end = len(tag)
		}
		optionStr := tag[:end]
		split := strings.IndexRune(optionStr, '=')
		if split < 0 {
			options[optionStr] = true
		} else {
			key := optionStr[:split]
			value := optionStr[split+1:]
			options[key] = value
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

func (codec *Codec) FormFieldType(objType reflect.Type) string {
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
