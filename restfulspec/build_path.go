package restfulspec

import (
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/spec"
)

// KeyOpenAPITags is a Metadata key for a restful Route
const KeyOpenAPITags = "openapi.tags"

func buildPaths(ws *restful.WebService, cfg Config, sb *swaggerBuilder) spec.Paths {
	p := spec.Paths{Paths: map[string]spec.PathItem{}}
	for _, each := range ws.Routes() {
		path, patterns := sanitizePath(each.Path)
		existingPathItem, ok := p.Paths[path]
		if !ok {
			existingPathItem = spec.PathItem{}
		}
		p.Paths[path] = buildPathItem(ws, each, existingPathItem, patterns, cfg, sb)
	}
	return p
}

// sanitizePath removes regex expressions from named path params,
// since openapi only supports setting the pattern as a a property named "pattern".
// Expressions like "/api/v1/{name:[a-z]/" are converted to "/api/v1/{name}/".
// The second return value is a map which contains the mapping from the path parameter
// name to the extracted pattern
func sanitizePath(restfulPath string) (string, map[string]string) {
	openapiPath := ""
	patterns := map[string]string{}
	for _, fragment := range strings.Split(restfulPath, "/") {
		if fragment == "" {
			continue
		}
		if strings.HasPrefix(fragment, "{") && strings.Contains(fragment, ":") {
			split := strings.Split(fragment, ":")
			fragment = split[0][1:]
			pattern := split[1][:len(split[1])-1]
			patterns[fragment] = pattern
			fragment = "{" + fragment + "}"
		}
		openapiPath += "/" + fragment
	}
	return openapiPath, patterns
}

func buildPathItem(ws *restful.WebService, r restful.Route, existingPathItem spec.PathItem, patterns map[string]string, cfg Config, sb *swaggerBuilder) spec.PathItem {
	op := buildOperation(ws, r, patterns, cfg, sb)
	switch r.Method {
	case "GET":
		existingPathItem.Get = op
	case "POST":
		existingPathItem.Post = op
	case "PUT":
		existingPathItem.Put = op
	case "DELETE":
		existingPathItem.Delete = op
	case "PATCH":
		existingPathItem.Patch = op
	case "OPTIONS":
		existingPathItem.Options = op
	case "HEAD":
		existingPathItem.Head = op
	}
	return existingPathItem
}

func buildOperation(ws *restful.WebService, r restful.Route, patterns map[string]string, cfg Config, sb *swaggerBuilder) *spec.Operation {
	o := spec.NewOperation(r.Operation)
	o.Description = r.Notes
	o.Summary = stripTags(r.Doc)
	o.Consumes = r.Consumes
	o.Produces = r.Produces
	o.Deprecated = r.Deprecated
	o.Security = r.Security
	if r.Metadata != nil {
		if tags, ok := r.Metadata[KeyOpenAPITags]; ok {
			if tagList, ok := tags.([]string); ok {
				o.Tags = tagList
			}
		}
	}
	// collect any path parameters
	for _, param := range ws.PathParameters() {
		o.Parameters = append(o.Parameters, sb.buildParameter(param, patterns[param.Name]))
	}
	// route specific params
	for _, each := range r.ParameterDocs {
		o.Parameters = append(o.Parameters, sb.buildParameter(each, patterns[each.Name]))
	}
	o.Responses = new(spec.Responses)
	props := &o.Responses.ResponsesProps
	props.StatusCodeResponses = map[int]spec.Response{}
	for k, v := range r.ResponseErrors {
		r := sb.buildResponse(v)
		props.StatusCodeResponses[k] = r
		if v.IsDefault {
			o.Responses.Default = &r
		}
	}
	if len(o.Responses.StatusCodeResponses) == 0 {
		o.Responses.StatusCodeResponses[200] = spec.Response{ResponseProps: spec.ResponseProps{Description: http.StatusText(http.StatusOK)}}
	}
	return o
}

// stringAutoType automatically picks the correct type from an ambiguously typed
// string. Ex. numbers become int, true/false become bool, etc.
func stringAutoType(dataType, ambiguous string) interface{} {
	if ambiguous == "" {
		return nil
	}
	switch dataType {
	case "int", "int8", "int16", "int32", "int64", "byte":
		if parsedInt, err := strconv.ParseInt(ambiguous, 10, 64); err == nil {
			return parsedInt
		}
	case "uint", "uint8", "uint16", "uint32", "uint64":
		if parsedUint, err := strconv.ParseUint(ambiguous, 10, 64); err == nil {
			return parsedUint
		}
	case "float32", "float64":
		if parsedFloat, err := strconv.ParseFloat(ambiguous, 64); err == nil {
			return parsedFloat
		}
	case "bool":
		if parsedBool, err := strconv.ParseBool(ambiguous); err == nil {
			return parsedBool
		}
	}
	return ambiguous
}

func stringIntType(t reflect.Type, v int64, err error) interface{} {
	if err != nil {
		return nil
	}

	ret := reflect.New(t).Elem()
	ret.SetInt(v)
	return ret.Interface()
}

func stringUintType(t reflect.Type, v uint64, err error) interface{} {
	if err != nil {
		return nil
	}

	ret := reflect.New(t).Elem()
	ret.SetUint(v)
	return ret.Interface()
}

func stringFloatType(t reflect.Type, v float64, err error) interface{} {
	if err != nil {
		return nil
	}

	ret := reflect.New(t).Elem()
	ret.SetFloat(v)
	return ret.Interface()
}

func stringBoolType(t reflect.Type, v bool, err error) interface{} {
	if err != nil {
		return nil
	}

	ret := reflect.New(t).Elem()
	ret.SetBool(v)
	return ret.Interface()
}

func stringReflectType(t reflect.Type, ambiguous string) interface{} {
	switch t.Kind() {
	case reflect.String:
		ret := reflect.New(t).Elem()
		ret.SetString(ambiguous)
		return ret.Interface()

	case reflect.Int8:
		v, err := strconv.ParseInt(ambiguous, 0, 8)
		return stringIntType(t, v, err)
	case reflect.Int16:
		v, err := strconv.ParseInt(ambiguous, 0, 16)
		return stringIntType(t, v, err)
	case reflect.Int, reflect.Int32:
		v, err := strconv.ParseInt(ambiguous, 0, 32)
		return stringIntType(t, v, err)
	case reflect.Int64:
		v, err := strconv.ParseInt(ambiguous, 0, 64)
		return stringIntType(t, v, err)

	case reflect.Uint8:
		v, err := strconv.ParseUint(ambiguous, 0, 8)
		return stringUintType(t, v, err)
	case reflect.Uint16:
		v, err := strconv.ParseUint(ambiguous, 0, 16)
		return stringUintType(t, v, err)
	case reflect.Uint, reflect.Uint32:
		v, err := strconv.ParseUint(ambiguous, 0, 32)
		return stringUintType(t, v, err)
	case reflect.Uint64:
		v, err := strconv.ParseUint(ambiguous, 0, 64)
		return stringUintType(t, v, err)

	case reflect.Float32:
		v, err := strconv.ParseFloat(ambiguous, 32)
		return stringFloatType(t, v, err)
	case reflect.Float64:
		v, err := strconv.ParseFloat(ambiguous, 64)
		return stringFloatType(t, v, err)

	case reflect.Bool:
		v, err := strconv.ParseBool(ambiguous)
		return stringBoolType(t, v, err)
	}

	return nil
}

// stripTags takes a snippet of HTML and returns only the text content.
// For example, `<b>&lt;Hi!&gt;</b> <br>` -> `&lt;Hi!&gt; `.
func stripTags(html string) string {
	re := regexp.MustCompile("<[^>]*>")
	return re.ReplaceAllString(html, "")
}

func isPrimitiveType(modelName string) bool {
	if len(modelName) == 0 {
		return false
	}
	return strings.Contains("uint uint8 uint16 uint32 uint64 int int8 int16 int32 int64 float32 float64 bool string byte rune time.Time", modelName)
}

func jsonSchemaType(modelName string) string {
	schemaMap := map[string]string{
		"uint":   "integer",
		"uint8":  "integer",
		"uint16": "integer",
		"uint32": "integer",
		"uint64": "integer",

		"int":   "integer",
		"int8":  "integer",
		"int16": "integer",
		"int32": "integer",
		"int64": "integer",

		"byte":      "string",
		"float64":   "number",
		"float32":   "number",
		"bool":      "boolean",
		"time.Time": "string",
	}
	mapped, ok := schemaMap[modelName]
	if !ok {
		return modelName // use as is (custom or struct)
	}
	return mapped
}

func jsonSchemaFormat(modelName string) string {
	schemaMap := map[string]string{
		"int":   "int32",
		"int8":  "int8",
		"int16": "int16",
		"int32": "int32",
		"int64": "int64",

		"uint":   "uint32",
		"uint8":  "uint8",
		"uint16": "uint16",
		"uint32": "uint32",
		"uint64": "uint64",

		"byte":       "byte",
		"float32":    "float",
		"float64":    "double",
		"time.Time":  "date-time",
		"*time.Time": "date-time",
	}
	mapped, ok := schemaMap[modelName]
	if !ok {
		return "" // no format
	}
	return mapped
}
