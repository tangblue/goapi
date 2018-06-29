package restful

import (
	"testing"
	"time"
)

func TestRouteBuilder_PathParameter(t *testing.T) {
	p := PathParameter("name", "desc")
	p.WithCollectionFormat(CollectionFormatMulti)
	p.DataType(int(0))
	p.AsRequired()

	b := new(RouteBuilder)
	b.function = dummy
	b.Params(p)
	r := b.Build()
	if r.ParameterDocs[0].CollectionFormat != "multi" {
		t.Error("AllowMultiple invalid")
	}
	if r.ParameterDocs[0].Model != int(0) {
		t.Error("dataType invalid")
	}
	if !r.ParameterDocs[0].Required {
		t.Error("required invalid")
	}
	if r.ParameterDocs[0].In != "path" {
		t.Error("kind invalid")
	}
	if b.ParameterNamed("name") == nil {
		t.Error("access to parameter failed")
	}
}

func TestRouteBuilder(t *testing.T) {
	json := "application/json"
	b := new(RouteBuilder)
	b.Handler(dummy)
	b.Path("/routes").Method("HEAD").Consumes(json).Produces(json).Metadata("test", "test-value").DefaultReturn("default", time.Now())
	r := b.Build()
	if r.Path != "/routes" {
		t.Error("path invalid")
	}
	if r.Produces[0] != json {
		t.Error("produces invalid")
	}
	if r.Consumes[0] != json {
		t.Error("consumes invalid")
	}
	if r.Operation != "dummy" {
		t.Error("Operation not set")
	}
	if r.Metadata["test"] != "test-value" {
		t.Errorf("Metadata not set")
	}
	if _, ok := r.ResponseErrors[0]; !ok {
		t.Fatal("expected default response")
	}
}

func TestAnonymousFuncNaming(t *testing.T) {
	f1 := func() {}
	f2 := func() {}
	if got, want := nameOfFunction(f1), "func1"; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := nameOfFunction(f2), "func2"; got != want {
		t.Errorf("got %v want %v", got, want)
	}
}
