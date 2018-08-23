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
	"strings"

	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/swag"
)

// SimpleSchema describe swagger simple schemas for parameters and headers
type SimpleSchema struct {
	Type             string      `json:"type,omitempty"`
	Format           string      `json:"format,omitempty"`
	Items            *Items      `json:"items,omitempty"`
	CollectionFormat string      `json:"collectionFormat,omitempty"`
	Default          interface{} `json:"default,omitempty"`
	Example          interface{} `json:"example,omitempty"`
}

// TypeName return the type (or format) of a simple schema
func (s *SimpleSchema) TypeName() string {
	if s.Format != "" {
		return s.Format
	}
	return s.Type
}

// ItemsTypeName yields the type of items in a simple schema array
func (s *SimpleSchema) ItemsTypeName() string {
	if s.Items == nil {
		return ""
	}
	return s.Items.TypeName()
}

// Typed a fluent builder method for the type of item
func (s *SimpleSchema) Typed(tpe, format string) *SimpleSchema {
	s.Type = tpe
	s.Format = format
	return s
}

// CollectionOf a fluent builder method for an array item
func (s *SimpleSchema) CollectionOf(items *Items, format string) *SimpleSchema {
	s.Type = "array"
	s.Items = items
	s.CollectionFormat = format
	return s
}

// WithDefault sets the default value on this item
func (s *SimpleSchema) WithDefault(defaultValue interface{}) *SimpleSchema {
	s.Default = defaultValue
	return s
}

// WithExample sets the example value on this item
func (s *SimpleSchema) WithExample(exampleValue interface{}) *SimpleSchema {
	s.Example = exampleValue
	return s
}

// CommonValidations describe common JSON-schema validations
type CommonValidations struct {
	Maximum          interface{}   `json:"maximum,omitempty"`
	ExclusiveMaximum bool          `json:"exclusiveMaximum,omitempty"`
	Minimum          interface{}   `json:"minimum,omitempty"`
	ExclusiveMinimum bool          `json:"exclusiveMinimum,omitempty"`
	MaxLength        *int          `json:"maxLength,omitempty"`
	MinLength        *int          `json:"minLength,omitempty"`
	Pattern          string        `json:"pattern,omitempty"`
	MaxItems         *int64        `json:"maxItems,omitempty"`
	MinItems         *int64        `json:"minItems,omitempty"`
	UniqueItems      bool          `json:"uniqueItems,omitempty"`
	MultipleOf       *float64      `json:"multipleOf,omitempty"`
	Enum             []interface{} `json:"enum,omitempty"`
}

// WithMaximum sets a maximum number value
func (v *CommonValidations) WithMaximum(max interface{}, exclusive bool) *CommonValidations {
	v.Maximum = max
	v.ExclusiveMaximum = exclusive
	return v
}

// WithMinimum sets a minimum number value
func (v *CommonValidations) WithMinimum(min interface{}, exclusive bool) *CommonValidations {
	v.Minimum = min
	v.ExclusiveMinimum = exclusive
	return v
}

// WithMaxLength sets a max length value
func (v *CommonValidations) WithMaxLength(max int) *CommonValidations {
	v.MaxLength = &max
	return v
}

// WithMinLength sets a min length value
func (v *CommonValidations) WithMinLength(min int) *CommonValidations {
	v.MinLength = &min
	return v
}

// WithPattern sets a pattern value
func (v *CommonValidations) WithPattern(pattern string) *CommonValidations {
	v.Pattern = pattern
	return v
}

// WithMaxItems sets the max items
func (v *CommonValidations) WithMaxItems(size int64) *CommonValidations {
	v.MaxItems = &size
	return v
}

// WithMinItems sets the min items
func (v *CommonValidations) WithMinItems(size int64) *CommonValidations {
	v.MinItems = &size
	return v
}

// UniqueValues dictates that this array can only have unique items
func (v *CommonValidations) UniqueValues() *CommonValidations {
	v.UniqueItems = true
	return v
}

// AllowDuplicates this array can have duplicates
func (v *CommonValidations) AllowDuplicates() *CommonValidations {
	v.UniqueItems = false
	return v
}

// WithMultipleOf sets a multiple of value
func (v *CommonValidations) WithMultipleOf(number float64) *CommonValidations {
	v.MultipleOf = &number
	return v
}

// WithEnum sets a the enum values (replace)
func (v *CommonValidations) WithEnum(values ...interface{}) *CommonValidations {
	v.Enum = append([]interface{}{}, values...)
	return v
}

// Items a limited subset of JSON-Schema's items object.
// It is used by parameter definitions that are not located in "body".
//
// For more information: http://goo.gl/8us55a#items-object
type Items struct {
	Refable
	CommonValidations
	SimpleSchema
	VendorExtensible
}

// NewItems creates a new instance of items
func NewItems() *Items {
	return &Items{}
}

// UnmarshalJSON hydrates this items instance with the data from JSON
func (i *Items) UnmarshalJSON(data []byte) error {
	var validations CommonValidations
	if err := json.Unmarshal(data, &validations); err != nil {
		return err
	}
	var ref Refable
	if err := json.Unmarshal(data, &ref); err != nil {
		return err
	}
	var simpleSchema SimpleSchema
	if err := json.Unmarshal(data, &simpleSchema); err != nil {
		return err
	}
	var vendorExtensible VendorExtensible
	if err := json.Unmarshal(data, &vendorExtensible); err != nil {
		return err
	}
	i.Refable = ref
	i.CommonValidations = validations
	i.SimpleSchema = simpleSchema
	i.VendorExtensible = vendorExtensible
	return nil
}

// MarshalJSON converts this items object to JSON
func (i Items) MarshalJSON() ([]byte, error) {
	b1, err := json.Marshal(i.CommonValidations)
	if err != nil {
		return nil, err
	}
	b2, err := json.Marshal(i.SimpleSchema)
	if err != nil {
		return nil, err
	}
	b3, err := json.Marshal(i.Refable)
	if err != nil {
		return nil, err
	}
	b4, err := json.Marshal(i.VendorExtensible)
	if err != nil {
		return nil, err
	}
	return swag.ConcatJSON(b4, b3, b1, b2), nil
}

// JSONLookup look up a value by the json property name
func (i Items) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return &i.Ref, nil
	}

	r, _, err := jsonpointer.GetForToken(i.CommonValidations, token)
	if err != nil && !strings.HasPrefix(err.Error(), "object has no field") {
		return nil, err
	}
	if r != nil {
		return r, nil
	}
	r, _, err = jsonpointer.GetForToken(i.SimpleSchema, token)
	return r, err
}
