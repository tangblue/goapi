package restfulspec

import (
	"testing"

	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/spec"
)

func TestRouteToPath(t *testing.T) {
	description := "get the <strong>a</strong> <em>b</em> test\nthis is the test description"
	notes := "notes\nblah blah"

	ws := new(restful.WebService)
	ws.Path("/tests/{v}")
	ws.Params(ws.PathParameter("v", "value of v").DataType("default-v"))
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_XML)
	ws.Route(ws.GET("/a/{b}").Handler(dummy).
		Doc(description).
		Note(notes).
		Params(ws.PathParameter("b", "value of b").DataType("default-b")).
		Params(ws.QueryParameter("q", "value of q").DataType("default-q")).
		Return(200, "list of a b tests", []Sample{}).
		Write([]Sample{}))
	ws.Route(ws.GET("/a/{b}/{c:[a-z]+}/{d:[1-9]+}/e").Handler(dummy).
		Params(ws.PathParameter("b", "value of b").DataType("default-b")).
		Params(ws.PathParameter("c", "with regex").DataType("abc")).
		Params(ws.PathParameter("d", "with regex").DataType("abcef")).
		Params(ws.QueryParameter("q", "value of q").DataType("default-q")).
		Return(200, "list of a b tests", []Sample{}).
		Write([]Sample{}))

	sb := &swaggerBuilder{}
	sb.def.Definitions = spec.Definitions{}
	p := buildPaths(ws, Config{}, sb)
	t.Log(asJSON(p))

	if p.Paths["/tests/{v}/a/{b}"].Get.Parameters[0].Type != "string" {
		t.Error("Parameter type is not set.")
	}
	if _, exists := p.Paths["/tests/{v}/a/{b}/{c}/{d}/e"]; !exists {
		t.Error("Expected path to exist after it was sanitized.")
	}

	if p.Paths["/tests/{v}/a/{b}"].Get.Description != notes {
		t.Errorf("GET description incorrect")
	}
	if p.Paths["/tests/{v}/a/{b}"].Get.Summary != "get the a b test\nthis is the test description" {
		t.Errorf("GET summary incorrect")
	}
	response := p.Paths["/tests/{v}/a/{b}"].Get.Responses.StatusCodeResponses[200]
	if response.Schema.Type[0] != "array" {
		t.Errorf("response type incorrect")
	}
	if response.Schema.Items.Schema.Ref.String() != "#/definitions/restfulspec.Sample" {
		t.Errorf("response element type incorrect")
	}

	// Test for patterns
	path := p.Paths["/tests/{v}/a/{b}/{c}/{d}/e"]
	checkPattern(t, path, "c", "[a-z]+")
	checkPattern(t, path, "d", "[1-9]+")
	checkPattern(t, path, "v", "")
}

func getParameter(path spec.PathItem, name string) (*spec.Parameter, bool) {
	for _, param := range path.Get.Parameters {
		if param.Name == name {
			return &param, true
		}
	}
	return nil, false
}

func checkPattern(t *testing.T, path spec.PathItem, paramName string, pattern string) {
	param, exists := getParameter(path, paramName)
	if !exists {
		t.Errorf("Expected Parameter %s to exist", paramName)
	}
	if param.Pattern != pattern {
		t.Errorf("Expected pattern %s to equal %s", param.Pattern, pattern)
	}
}

func TestMultipleMethodsRouteToPath(t *testing.T) {
	ws := new(restful.WebService)
	ws.Path("/tests/a")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_XML)
	ws.Route(ws.GET("/a/b").Handler(dummy).
		Doc("get a b test").
		Return(200, "list of a b tests", []Sample{}).
		Write([]Sample{}))
	ws.Route(ws.POST("/a/b").Handler(dummy).
		Doc("post a b test").
		Return(200, "list of a b tests", []Sample{}).
		Return(500, "internal server error", []Sample{}).
		Read(Sample{}).
		Write([]Sample{}))

	sb := &swaggerBuilder{}
	sb.def.Definitions = spec.Definitions{}
	p := buildPaths(ws, Config{}, sb)
	t.Log(asJSON(p))

	if p.Paths["/tests/a/a/b"].Get.Summary != "get a b test" {
		t.Errorf("GET summary incorrect")
	}
	if p.Paths["/tests/a/a/b"].Post.Summary != "post a b test" {
		t.Errorf("POST summary incorrect")
	}
	if _, exists := p.Paths["/tests/a/a/b"].Post.Responses.StatusCodeResponses[500]; !exists {
		t.Errorf("Response code 500 not added to spec.")
	}

	expectedRef := spec.MustCreateRef("#/definitions/restfulspec.Sample")
	postBodyparam := p.Paths["/tests/a/a/b"].Post.Parameters[0]
	postBodyRef := postBodyparam.Schema.Ref
	if postBodyRef.String() != expectedRef.String() {
		t.Errorf("Expected: %s, Got: %s", expectedRef.String(), postBodyRef.String())
	}

	if postBodyparam.Format != "" || postBodyparam.Type != "" || postBodyparam.Default != nil {
		t.Errorf("Invalid parameter property is set on body property")
	}
}

func TestReadArrayObjectInBody(t *testing.T) {
	ws := new(restful.WebService)
	ws.Path("/tests/a")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_XML)

	ws.Route(ws.POST("/a/b").Handler(dummy).
		Doc("post a b test with array in body").
		Return(200, "list of a b tests", []Sample{}).
		Return(500, "internal server error", []Sample{}).
		Read([]Sample{}).
		Write([]Sample{}))

	sb := &swaggerBuilder{}
	sb.def.Definitions = spec.Definitions{}
	p := buildPaths(ws, Config{}, sb)
	t.Log(asJSON(p))

	postInfo := p.Paths["/tests/a/a/b"].Post

	if postInfo.Summary != "post a b test with array in body" {
		t.Errorf("POST description incorrect")
	}
	if _, exists := postInfo.Responses.StatusCodeResponses[500]; !exists {
		t.Errorf("Response code 500 not added to spec.")
	}
	// indentify  element model type in body array
	expectedItemRef := spec.MustCreateRef("#/definitions/restfulspec.Sample")
	postBody := postInfo.Parameters[0]
	if postBody.Schema.Ref.String() != "" {
		t.Errorf("you shouldn't have body Ref setting when using array in body!")
	}
	// check body array dy item ref
	postBodyitems := postBody.Schema.Items.Schema.Ref
	if postBodyitems.String() != expectedItemRef.String() {
		t.Errorf("Expected: %s, Got: %s", expectedItemRef.String(), expectedItemRef.String())
	}

	if postBody.Format != "" || postBody.Type != "" || postBody.Default != nil {
		t.Errorf("Invalid parameter property is set on body property")
	}
}

// TestWritesPrimitive ensures that if an operation returns a primitive, then it
// is used as such (and not a ref to a definition).
func TestWritesPrimitive(t *testing.T) {
	ws := new(restful.WebService)
	ws.Path("/tests/returns")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/primitive").Handler(dummy).
		Doc("post that returns a string").
		Return(200, "primitive string", "(this is a string)").
		Write("(this is a string)"))

	ws.Route(ws.POST("/custom").Handler(dummy).
		Doc("post that returns a custom structure").
		Return(200, "sample object", Sample{}).
		Write(Sample{}))

	sb := &swaggerBuilder{}
	sb.def.Definitions = spec.Definitions{}
	p := buildPaths(ws, Config{}, sb)
	t.Log(asJSON(p))

	// Make sure that the operation that returns a primitive type is correct.
	if pathInfo, okay := p.Paths["/tests/returns/primitive"]; !okay {
		t.Errorf("Could not find path")
	} else {
		postInfo := pathInfo.Post

		if postInfo.Summary != "post that returns a string" {
			t.Errorf("POST description incorrect")
		}
		response := postInfo.Responses.StatusCodeResponses[200]
		if response.Schema.Ref.String() != "" {
			t.Errorf("Expected no ref; got: %s", response.Schema.Ref.String())
		}
		if len(response.Schema.Type) != 1 {
			t.Errorf("Expected exactly one type; got: %d", len(response.Schema.Type))
		}
		if response.Schema.Type[0] != "string" {
			t.Errorf("Expected a type of string; got: %s", response.Schema.Type[0])
		}
	}

	// Make sure that the operation that returns a custom type is correct.
	if pathInfo, okay := p.Paths["/tests/returns/custom"]; !okay {
		t.Errorf("Could not find path")
	} else {
		postInfo := pathInfo.Post

		if postInfo.Summary != "post that returns a custom structure" {
			t.Errorf("POST description incorrect")
		}
		response := postInfo.Responses.StatusCodeResponses[200]
		if response.Schema.Ref.String() != "#/definitions/restfulspec.Sample" {
			t.Errorf("Expected ref '#/definitions/restfulspec.Sample'; got: %s", response.Schema.Ref.String())
		}
		if len(response.Schema.Type) != 0 {
			t.Errorf("Expected exactly zero types; got: %d", len(response.Schema.Type))
		}
	}
}
