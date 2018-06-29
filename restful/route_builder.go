package restful

// Copyright 2013 Ernest Micklei. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/tangblue/goapi/restful/log"
	"github.com/tangblue/goapi/spec"
)

// RouteBuilder is a helper to construct Routes.
type RouteBuilder struct {
	rootPath    string
	currentPath string
	produces    []string
	consumes    []string
	httpMethod  string        // required
	function    RouteFunction // required
	filters     []FilterFunction
	conditions  []RouteSelectionConditionFunction

	typeNameHandleFunc TypeNameHandleFunction // required

	// documentation
	doc                     string
	notes                   string
	operation               string
	readSample, writeSample interface{}
	parameters              []*Parameter
	errorMap                map[int]*ResponseError
	metadata                map[string]interface{}
	deprecated              bool
	securities              []map[string][]string
}

// Do evaluates each argument with the RouteBuilder itself.
// This allows you to follow DRY principles without breaking the fluent programming style.
// Example:
// 		ws.Route(ws.DELETE("/{name}").Handler(t.deletePerson).Do(Return200, Return500))
//
//		func Return500(b *RouteBuilder) {
//			b.Return(500, "Internal Server Error", restful.ServiceError{})
//		}
func (b *RouteBuilder) Do(oneArgBlocks ...func(*RouteBuilder)) *RouteBuilder {
	for _, each := range oneArgBlocks {
		each(b)
	}
	return b
}

// Handler bind the route to a function.
// If this route is matched with the incoming Http Request then call this function with the *Request,*Response pair. Required.
func (b *RouteBuilder) Handler(function RouteFunction) *RouteBuilder {
	b.function = function
	return b
}

func (b *RouteBuilder) Security(name string, scopes []string) *RouteBuilder {
	if b.securities == nil {
		b.securities = []map[string][]string{}
	}
	b.securities = append(b.securities, map[string][]string{name: scopes})
	return b
}

// Method specifies what HTTP method to match. Required.
func (b *RouteBuilder) Method(method string) *RouteBuilder {
	b.httpMethod = method
	return b
}

// Produce specifies what MIME types can be produced ; the matched one will appear in the Content-Type Http header.
func (b *RouteBuilder) Produces(mimeTypes ...string) *RouteBuilder {
	b.produces = mimeTypes
	return b
}

// Consume specifies what MIME types can be consumes ; the Accept Http header must matched any of these
func (b *RouteBuilder) Consumes(mimeTypes ...string) *RouteBuilder {
	b.consumes = mimeTypes
	return b
}

// Path specifies the relative (w.r.t WebService root path) URL path to match. Default is "/".
func (b *RouteBuilder) Path(subPath string) *RouteBuilder {
	b.currentPath = subPath
	return b
}

func (b *RouteBuilder) ParamPath(subPath string, parameters ...*Parameter) *RouteBuilder {
	if len(parameters) == 0 {
		b.currentPath = subPath
	} else {
		var s []interface{} = make([]interface{}, len(parameters))
		for i, v := range parameters {
			s[i] = v
		}
		b.currentPath = fmt.Sprintf(subPath, s...)

		if b.parameters == nil {
			b.parameters = []*Parameter{}
		}
		b.parameters = append(b.parameters, parameters...)
	}
	return b
}

// Doc tells what this route is all about. Optional.
func (b *RouteBuilder) Doc(documentation string) *RouteBuilder {
	b.doc = documentation
	return b
}

// Note is a verbose explanation of the operation behavior. Optional.
func (b *RouteBuilder) Note(notes string) *RouteBuilder {
	b.notes = notes
	return b
}

// Read tells what resource type will be read from the request payload. Optional.
// A parameter of type "body" is added ,required is set to true and the dataType is set to the qualified name of the sample's type.
func (b *RouteBuilder) Read(sample interface{}, optionalDescription ...string) *RouteBuilder {
	fn := b.typeNameHandleFunc
	if fn == nil {
		fn = reflectTypeName
	}
	typeAsName := fn(sample)
	description := ""
	if len(optionalDescription) > 0 {
		description = optionalDescription[0]
	}
	b.readSample = sample
	bodyParameter := BodyParameter("body", description)
	bodyParameter.DataType(sample)
	bodyParameter.Typed(typeAsName, "")
	b.Params(bodyParameter)
	return b
}

// ParameterNamed returns a Parameter already known to the RouteBuilder. Return nil if not.
// Use this to modify or extend information for the Parameter (through its Data()).
func (b RouteBuilder) ParameterNamed(name string) (p *Parameter) {
	for _, each := range b.parameters {
		if each.Name == name {
			return each
		}
	}
	return p
}

// Write tells what resource type will be written as the response payload. Optional.
func (b *RouteBuilder) Write(sample interface{}) *RouteBuilder {
	b.writeSample = sample
	return b
}

// Params allows you to document the parameters of the Route. It adds a new Parameter (does not check for duplicates).
func (b *RouteBuilder) Params(parameters ...*Parameter) *RouteBuilder {
	if b.parameters == nil {
		b.parameters = []*Parameter{}
	}
	b.parameters = append(b.parameters, parameters...)
	return b
}

// Operation allows you to document what the actual method/function call is of the Route.
// Unless called, the operation name is derived from the RouteFunction set using Handler(..).
func (b *RouteBuilder) Operation(name string) *RouteBuilder {
	b.operation = name
	return b
}

// Return allows you to document what responses (errors or regular) can be expected.
// The model parameter is optional ; either pass a struct instance or use nil if not applicable.
func (b *RouteBuilder) Return(code int, message string, model interface{}) *RouteBuilder {
	// lazy init because there is no NewRouteBuilder (yet)
	if b.errorMap == nil {
		b.errorMap = map[int]*ResponseError{}
	}
	b.errorMap[code] = NewResponseError(code, message, model)
	return b
}

// DefaultReturn is a special Return call that sets the default of the response ; the code is zero.
func (b *RouteBuilder) DefaultReturn(message string, model interface{}) *RouteBuilder {
	b.Return(0, message, model)
	// Modify the ResponseError just added/updated
	b.errorMap[0].IsDefault = true
	return b
}

func (b *RouteBuilder) ReturnResponses(errs ...*ResponseError) *RouteBuilder {
	// lazy init because there is no NewRouteBuilder (yet)
	if b.errorMap == nil {
		b.errorMap = map[int]*ResponseError{}
	}
	for _, e := range errs {
		b.errorMap[e.Code] = e
	}
	return b
}

// Metadata adds or updates a key=value pair to the metadata map.
func (b *RouteBuilder) Metadata(key string, value interface{}) *RouteBuilder {
	if b.metadata == nil {
		b.metadata = map[string]interface{}{}
	}
	b.metadata[key] = value
	return b
}

// Deprecate sets the value of deprecated to true.  Deprecated routes have a special UI treatment to warn against use
func (b *RouteBuilder) Deprecate() *RouteBuilder {
	b.deprecated = true
	return b
}

// ResponseError represents a response; not necessarily an error.
type ResponseError struct {
	spec.Response
	Code      int
	Model     interface{}
	IsDefault bool
	RefName   string
}

func NewResponseError(code int, message string, model interface{}) *ResponseError {
	r := &ResponseError{
		Code:      code,
		Model:     model,
		IsDefault: false,
	}
	r.WithDescription(message)

	return r
}

func (r *ResponseError) SetRefName(refName string) *ResponseError {
	r.RefName = refName
	return r
}

func (r *ResponseError) Header(name, description string, v interface{}) *ResponseError {
	h := spec.ResponseHeader().WithDescription(description)
	h.SimpleSchema.WithExample(v)
	r.AddHeader(name, h)
	return r
}

func (b *RouteBuilder) servicePath(path string) *RouteBuilder {
	b.rootPath = path
	return b
}

// Filter appends a FilterFunction to the end of filters for this Route to build.
func (b *RouteBuilder) Filter(filter FilterFunction) *RouteBuilder {
	b.filters = append(b.filters, filter)
	return b
}

// If sets a condition function that controls matching the Route based on custom logic.
// The condition function is provided the HTTP request and should return true if the route
// should be considered.
//
// Efficiency note: the condition function is called before checking the method, produces, and
// consumes criteria, so that the correct HTTP status code can be returned.
//
// Lifecycle note: no filter functions have been called prior to calling the condition function,
// so the condition function should not depend on any context that might be set up by container
// or route filters.
func (b *RouteBuilder) If(condition RouteSelectionConditionFunction) *RouteBuilder {
	b.conditions = append(b.conditions, condition)
	return b
}

// If no specific Route path then set to rootPath
// If no specific Produce then set to rootProduces
// If no specific Consume then set to rootConsumes
func (b *RouteBuilder) copyDefaults(rootProduces, rootConsumes []string) {
	if len(b.produces) == 0 {
		b.produces = rootProduces
	}
	if len(b.consumes) == 0 {
		b.consumes = rootConsumes
	}
}

// typeNameHandler sets the function that will convert types to strings in the parameter
// and model definitions.
func (b *RouteBuilder) typeNameHandler(handler TypeNameHandleFunction) *RouteBuilder {
	b.typeNameHandleFunc = handler
	return b
}

// Build creates a new Route using the specification details collected by the RouteBuilder
func (b *RouteBuilder) Build() Route {
	pathExpr, err := newPathExpression(b.currentPath)
	if err != nil {
		log.Printf("[restful] Invalid path:%s because:%v", b.currentPath, err)
		os.Exit(1)
	}
	if b.function == nil {
		log.Printf("[restful] No function specified for route:" + b.currentPath)
		os.Exit(1)
	}
	operationName := b.operation
	if len(operationName) == 0 && b.function != nil {
		// extract from definition
		operationName = nameOfFunction(b.function)
	}
	route := Route{
		Method:         b.httpMethod,
		Path:           concatPath(b.rootPath, b.currentPath),
		Produces:       b.produces,
		Consumes:       b.consumes,
		Function:       b.function,
		Filters:        b.filters,
		If:             b.conditions,
		relativePath:   b.currentPath,
		pathExpr:       pathExpr,
		Doc:            b.doc,
		Notes:          b.notes,
		Operation:      operationName,
		ParameterDocs:  b.parameters,
		ResponseErrors: b.errorMap,
		ReadSample:     b.readSample,
		WriteSample:    b.writeSample,
		Metadata:       b.metadata,
		Deprecated:     b.deprecated,
		Security:       b.securities}
	route.postBuild()
	return route
}

func concatPath(path1, path2 string) string {
	return strings.TrimRight(path1, "/") + "/" + strings.TrimLeft(path2, "/")
}

var anonymousFuncCount int32

// nameOfFunction returns the short name of the function f for documentation.
// It uses a runtime feature for debugging ; its value may change for later Go versions.
func nameOfFunction(f interface{}) string {
	fun := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
	tokenized := strings.Split(fun.Name(), ".")
	last := tokenized[len(tokenized)-1]
	last = strings.TrimSuffix(last, ")·fm") // < Go 1.5
	last = strings.TrimSuffix(last, ")-fm") // Go 1.5
	last = strings.TrimSuffix(last, "·fm")  // < Go 1.5
	last = strings.TrimSuffix(last, "-fm")  // Go 1.5
	if last == "func1" {                    // this could mean conflicts in API docs
		val := atomic.AddInt32(&anonymousFuncCount, 1)
		last = "func" + fmt.Sprintf("%d", val)
		atomic.StoreInt32(&anonymousFuncCount, val)
	}
	return last
}
