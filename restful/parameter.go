package restful

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
)

// Copyright 2013 Ernest Micklei. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

const (
	// PathParameterKind = indicator of Request parameter type "path"
	PathParameterKind = iota

	// QueryParameterKind = indicator of Request parameter type "query"
	QueryParameterKind

	// BodyParameterKind = indicator of Request parameter type "body"
	BodyParameterKind

	// HeaderParameterKind = indicator of Request parameter type "header"
	HeaderParameterKind

	// FormParameterKind = indicator of Request parameter type "form"
	FormParameterKind

	// CollectionFormatCSV comma separated values `foo,bar`
	CollectionFormatCSV = CollectionFormat("csv")

	// CollectionFormatSSV space separated values `foo bar`
	CollectionFormatSSV = CollectionFormat("ssv")

	// CollectionFormatTSV tab separated values `foo\tbar`
	CollectionFormatTSV = CollectionFormat("tsv")

	// CollectionFormatPipes pipe separated values `foo|bar`
	CollectionFormatPipes = CollectionFormat("pipes")

	// CollectionFormatMulti corresponds to multiple parameter instances instead of multiple values for a single
	// instance `foo=bar&foo=baz`. This is valid only for QueryParameters and FormParameters
	CollectionFormatMulti = CollectionFormat("multi")
)

type CollectionFormat string

func (cf CollectionFormat) String() string {
	return string(cf)
}

// Parameter is for documententing the parameter used in a Http Request
// ParameterData kinds are Path,Query and Body
type Parameter struct {
	data *ParameterData
}

// ParameterData represents the state of a Parameter.
// It is made public to make it accessible to e.g. the Swagger package.
type ParameterData struct {
	Name, Description, DataFormat string
	Kind                          int
	Required                      bool
	AllowableValues               map[string]string
	AllowMultiple                 bool
	DefaultValue                  interface{}
	MinValue, MaxValue            interface{}
	MinLength, MaxLength          int
	CollectionFormat              string
	Regex                         string
	regex                         *regexp.Regexp
}

// Data returns the state of the Parameter
func (p *Parameter) Data() ParameterData {
	return *p.data
}

func (p *Parameter) String() string {
	path := p.data.Name
	if p.data.Regex != "" {
		path += string(':') + p.data.Regex
	}

	return path
}

// Kind returns the parameter type indicator (see const for valid values)
func (p *Parameter) Kind() int {
	return p.data.Kind
}

func (p *Parameter) bePath() *Parameter {
	p.data.Kind = PathParameterKind
	return p
}
func (p *Parameter) beQuery() *Parameter {
	p.data.Kind = QueryParameterKind
	return p
}
func (p *Parameter) beBody() *Parameter {
	p.data.Kind = BodyParameterKind
	return p
}

func (p *Parameter) beHeader() *Parameter {
	p.data.Kind = HeaderParameterKind
	return p
}

func (p *Parameter) beForm() *Parameter {
	p.data.Kind = FormParameterKind
	return p
}

// Required sets the required field and returns the receiver
func (p *Parameter) Required(required bool) *Parameter {
	p.data.Required = required
	return p
}

// AllowMultiple sets the allowMultiple field and returns the receiver
func (p *Parameter) AllowMultiple(multiple bool) *Parameter {
	p.data.AllowMultiple = multiple
	return p
}

// AllowableValues sets the allowableValues field and returns the receiver
func (p *Parameter) AllowableValues(values map[string]string) *Parameter {
	p.data.AllowableValues = values
	return p
}

// DataType sets the dataType field and returns the receiver
func (p *Parameter) DataType(val interface{}) *Parameter {
	p.data.DefaultValue = val
	return p
}

// DataFormat sets the dataFormat field for Swagger UI
func (p *Parameter) DataFormat(formatName string) *Parameter {
	p.data.DataFormat = formatName
	return p
}

// DefaultValue sets the default value field and returns the receiver
func (p *Parameter) DefaultValue(val interface{}) *Parameter {
	p.data.DefaultValue = val
	return p
}

// Description sets the description value field and returns the receiver
func (p *Parameter) Description(doc string) *Parameter {
	p.data.Description = doc
	return p
}

// CollectionFormat sets the collection format for an array type
func (p *Parameter) CollectionFormat(format CollectionFormat) *Parameter {
	p.data.CollectionFormat = format.String()
	return p
}

func (p *Parameter) Regex(regex string) *Parameter {
	r, err := regexp.Compile(regex)
	if err != nil {
		panic("Bad regex: " + regex)
	}
	p.data.Regex = regex
	p.data.regex = r
	return p
}

func (p *Parameter) ValueRange(min, max interface{}) *Parameter {
	if reflect.TypeOf(min) != reflect.TypeOf(p.data.DefaultValue) {
		panic("bad type: min")
	}
	if reflect.TypeOf(max) != reflect.TypeOf(p.data.DefaultValue) {
		panic("bad type: max")
	}
	p.data.MinValue = min
	p.data.MaxValue = max
	return p
}

func (p *Parameter) LengthRange(min, max int) *Parameter {
	p.data.MinLength = min
	p.data.MaxLength = max
	return p
}

func (p *Parameter) getName() string {
	return p.data.Name
}

func (p *Parameter) GetDataTypeName() string {
	return reflect.TypeOf(p.data.DefaultValue).String()
}

func (p *Parameter) GetDataType() interface{} {
	return p.data.DefaultValue
}

func (p *Parameter) getKind() int {
	return p.data.Kind
}

func (p *Parameter) isRequired() bool {
	return p.data.Required
}

func (p *Parameter) getDefaultValue() interface{} {
	return p.data.DefaultValue
}

var (
	errLTMin      = errors.New("less than minimum")
	errGTMax      = errors.New("great than maximum")
	errTooShort   = errors.New("too short")
	errTooLong    = errors.New("too long")
	errBadPattern = errors.New("bad pattern")
)

func (p *Parameter) getValue(s string) (interface{}, error) {

	switch reflect.TypeOf(p.data.DefaultValue).Kind() {
	case reflect.String:
		return p.ValidateValueString(s, nil)
	case reflect.Int8:
		return p.ValidateValueInt(strconv.ParseInt(s, 0, 8))
	case reflect.Int16:
		return p.ValidateValueInt(strconv.ParseInt(s, 0, 16))
	case reflect.Int, reflect.Int32:
		return p.ValidateValueInt(strconv.ParseInt(s, 0, 32))
	case reflect.Int64:
		return p.ValidateValueInt(strconv.ParseInt(s, 0, 64))

	case reflect.Uint8:
		return p.ValidateValueUint(strconv.ParseUint(s, 0, 8))
	case reflect.Uint16:
		return p.ValidateValueUint(strconv.ParseUint(s, 0, 16))
	case reflect.Uint, reflect.Uint32:
		return p.ValidateValueUint(strconv.ParseUint(s, 0, 32))
	case reflect.Uint64:
		return p.ValidateValueUint(strconv.ParseUint(s, 0, 64))

	case reflect.Bool:
		return p.ValidateValueBool(strconv.ParseBool(s))
	case reflect.Float32:
		return p.ValidateValueFloat(strconv.ParseFloat(s, 32))
	case reflect.Float64:
		return p.ValidateValueFloat(strconv.ParseFloat(s, 64))
	}
	return p.data.DefaultValue, errors.New("unknown type")
}

func (p *Parameter) ValidateValueString(v string, err error) (interface{}, error) {
	dv := p.data.DefaultValue
	if err != nil {
		return dv, err
	}

	if p.data.MinLength != 0 || p.data.MaxLength != 0 {
		if len(v) < p.data.MinLength {
			return dv, errTooShort
		} else if len(v) > p.data.MaxLength {
			return dv, errTooLong
		}
	}
	if p.data.regex != nil {
		if !p.data.regex.MatchString(v) {
			return dv, errBadPattern
		}
	}

	ret := reflect.New(reflect.TypeOf(dv))
	ret.Elem().SetString(v)

	return ret.Elem().Interface(), err
}

func (p *Parameter) ValidateValueInt(v int64, err error) (interface{}, error) {
	dv := p.data.DefaultValue

	if err != nil {
		return dv, err
	}

	if p.data.MinValue != nil && v < reflect.ValueOf(p.data.MinValue).Int() {
		return dv, errLTMin
	} else if p.data.MaxValue != nil && v > reflect.ValueOf(p.data.MaxValue).Int() {
		return dv, errGTMax
	}

	ret := reflect.New(reflect.TypeOf(dv))
	ret.Elem().SetInt(v)

	return ret.Elem().Interface(), err
}

func (p *Parameter) ValidateValueUint(v uint64, err error) (interface{}, error) {
	dv := p.data.DefaultValue

	if err != nil {
		return dv, err
	}

	if p.data.MinValue != nil && v < reflect.ValueOf(p.data.MinValue).Uint() {
		return dv, errLTMin
	} else if p.data.MaxValue != nil && v > reflect.ValueOf(p.data.MaxValue).Uint() {
		return dv, errGTMax
	}

	ret := reflect.New(reflect.TypeOf(dv))
	ret.Elem().SetUint(v)

	return ret.Elem().Interface(), err
}

func (p *Parameter) ValidateValueBool(v bool, err error) (interface{}, error) {
	dv := p.data.DefaultValue

	if err != nil {
		return dv, err
	}

	ret := reflect.New(reflect.TypeOf(dv))
	ret.Elem().SetBool(v)

	return ret.Elem().Interface(), err
}

func (p *Parameter) ValidateValueFloat(v float64, err error) (interface{}, error) {
	dv := p.data.DefaultValue

	if err != nil {
		return dv, err
	}

	if p.data.MinValue != nil && v < reflect.ValueOf(p.data.MinValue).Float() {
		return dv, errLTMin
	} else if p.data.MaxValue != nil && v > reflect.ValueOf(p.data.MaxValue).Float() {
		return dv, errGTMax
	}

	ret := reflect.New(reflect.TypeOf(dv))
	ret.Elem().SetFloat(v)

	return ret.Elem().Interface(), err
}
