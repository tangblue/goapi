// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var parameter = Parameter{
	VendorExtensible: VendorExtensible{Extensions: map[string]interface{}{
		"x-framework": "swagger-go",
	}},
	Refable: Refable{Ref: MustCreateRef("Dog")},
	CommonValidations: CommonValidations{
		Maximum:          float64(100),
		ExclusiveMaximum: true,
		ExclusiveMinimum: true,
		Minimum:          float64(5),
		MaxLength:        intPtr(100),
		MinLength:        intPtr(5),
		Pattern:          "\\w{1,5}\\w+",
		MaxItems:         int64Ptr(100),
		MinItems:         int64Ptr(5),
		UniqueItems:      true,
		MultipleOf:       float64Ptr(5),
		Enum:             []interface{}{"hello", "world"},
	},
	SimpleSchema: SimpleSchema{
		Type:             "string",
		Format:           "date",
		CollectionFormat: "csv",
		Items: &Items{
			Refable: Refable{Ref: MustCreateRef("Cat")},
		},
		Default: "8",
	},
	ParamProps: ParamProps{
		Name:        "param-name",
		In:          "header",
		Required:    true,
		Schema:      &Schema{SchemaProps: SchemaProps{Type: []string{"string"}}},
		Description: "the description of this parameter",
	},
}

var parameterJSON = `{
	"items": {
		"$ref": "Cat"
	},
	"x-framework": "swagger-go",
  "$ref": "Dog",
  "description": "the description of this parameter",
  "maximum": 100,
  "minimum": 5,
  "exclusiveMaximum": true,
  "exclusiveMinimum": true,
  "maxLength": 100,
  "minLength": 5,
  "pattern": "\\w{1,5}\\w+",
  "maxItems": 100,
  "minItems": 5,
  "uniqueItems": true,
  "multipleOf": 5,
  "enum": ["hello", "world"],
  "type": "string",
  "format": "date",
	"name": "param-name",
	"in": "header",
	"required": true,
	"schema": {
		"type": "string"
	},
	"collectionFormat": "csv",
	"default": "8"
}`

func TestIntegrationParameter(t *testing.T) {
	var actual Parameter
	if assert.NoError(t, json.Unmarshal([]byte(parameterJSON), &actual)) {
		assert.EqualValues(t, actual, parameter)
	}

	assertParsesJSON(t, parameterJSON, parameter)
}

func TestParameterSerialization(t *testing.T) {
	items := &Items{
		SimpleSchema: SimpleSchema{Type: "string"},
	}
	stringTyped := func(p *Parameter) *Parameter {
		p.Typed("string", "")
		return p
	}
	collectionOf := func(p *Parameter, items *Items) *Parameter {
		p.CollectionOf(items, "multi")
		return p
	}

	intItems := &Items{
		SimpleSchema: SimpleSchema{Type: "int", Format: "int32"},
	}

	assertSerializeJSON(t, stringTyped(QueryParam("")), `{"type":"string","in":"query"}`)

	assertSerializeJSON(t,
		collectionOf(QueryParam(""), items),
		`{"type":"array","items":{"type":"string"},"collectionFormat":"multi","in":"query"}`)

	assertSerializeJSON(t, stringTyped(PathParam("")), `{"type":"string","in":"path","required":true}`)

	assertSerializeJSON(t,
		collectionOf(PathParam(""), items),
		`{"type":"array","items":{"type":"string"},"collectionFormat":"multi","in":"path","required":true}`)

	assertSerializeJSON(t,
		collectionOf(PathParam(""), intItems),
		`{"type":"array","items":{"type":"int","format":"int32"},"collectionFormat":"multi","in":"path","required":true}`)

	assertSerializeJSON(t, stringTyped(HeaderParam("")), `{"type":"string","in":"header","required":true}`)

	assertSerializeJSON(t,
		collectionOf(HeaderParam(""), items),
		`{"type":"array","items":{"type":"string"},"collectionFormat":"multi","in":"header","required":true}`)
	schema := &Schema{SchemaProps: SchemaProps{
		Properties: map[string]Schema{
			"name": Schema{SchemaProps: SchemaProps{
				Type: []string{"string"},
			}},
		},
	}}

	refSchema := &Schema{
		SchemaProps: SchemaProps{Ref: MustCreateRef("Cat")},
	}

	assertSerializeJSON(t,
		BodyParam("", schema),
		`{"type":"object","in":"body","required":true,"schema":{"properties":{"name":{"type":"string"}}}}`)

	assertSerializeJSON(t,
		BodyParam("", refSchema),
		`{"type":"object","in":"body","required":true,"schema":{"$ref":"Cat"}}`)

	// array body param
	assertSerializeJSON(t,
		BodyParam("", ArrayProperty(RefProperty("Cat"))),
		`{"type":"object","in":"body","required":true,"schema":{"type":"array","items":{"$ref":"Cat"}}}`)

}
