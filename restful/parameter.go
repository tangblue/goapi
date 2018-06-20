package restful

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"

	"github.com/tangblue/goapi/spec"
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
	spec.Parameter
	Model   interface{}
	regex   *regexp.Regexp
	RefName string
}

func (p *Parameter) String() string {
	path := p.Name
	if p.Pattern != "" {
		path += string(':') + p.Pattern
	}

	return path
}

func QueryParameter(name, description string) *Parameter {
	return &Parameter{
		Parameter: *spec.QueryParam(name).WithDescription(description),
		Model:     "",
	}
}

func HeaderParameter(name, description string) *Parameter {
	return &Parameter{
		Parameter: *spec.HeaderParam(name).WithDescription(description),
		Model:     "",
	}
}

func PathParameter(name, description string) *Parameter {
	return &Parameter{
		Parameter: *spec.PathParam(name).WithDescription(description),
		Model:     "",
	}
}

func BodyParameter(name, description string) *Parameter {
	return &Parameter{
		Parameter: *spec.BodyParam(name, nil).WithDescription(description),
		Model:     "",
	}
}

func FormDataParameter(name, description string) *Parameter {
	return &Parameter{
		Parameter: *spec.FormDataParam(name).WithDescription(description),
		Model:     "",
	}
}

// CollectionFormat sets the collection format for an array type
func (p *Parameter) WithCollectionFormat(format CollectionFormat) *Parameter {
	p.CollectionFormat = format.String()
	return p
}

func (p *Parameter) DataType(model interface{}) *Parameter {
	p.Model = model
	return p
}

func (p *Parameter) Regex(regex string) *Parameter {
	r, err := regexp.Compile(regex)
	if err != nil {
		panic("Bad regex: " + regex)
	}
	p.Pattern = regex
	p.regex = r
	return p
}

func (p *Parameter) SetRefName(refName string) *Parameter {
	p.RefName = refName
	return p
}

var (
	errLTMin      = errors.New("less than minimum")
	errGTMax      = errors.New("great than maximum")
	errTooShort   = errors.New("too short")
	errTooLong    = errors.New("too long")
	errBadPattern = errors.New("bad pattern")
	errBadEnum    = errors.New("bad enum")
)

func (p *Parameter) getValue(s string) (interface{}, error) {
	switch reflect.TypeOf(p.Model).Kind() {
	case reflect.String:
		return p.validateValueString(s, nil)
	case reflect.Int8:
		return p.validateValueInt(strconv.ParseInt(s, 0, 8))
	case reflect.Int16:
		return p.validateValueInt(strconv.ParseInt(s, 0, 16))
	case reflect.Int, reflect.Int32:
		return p.validateValueInt(strconv.ParseInt(s, 0, 32))
	case reflect.Int64:
		return p.validateValueInt(strconv.ParseInt(s, 0, 64))

	case reflect.Uint8:
		return p.validateValueUint(strconv.ParseUint(s, 0, 8))
	case reflect.Uint16:
		return p.validateValueUint(strconv.ParseUint(s, 0, 16))
	case reflect.Uint, reflect.Uint32:
		return p.validateValueUint(strconv.ParseUint(s, 0, 32))
	case reflect.Uint64:
		return p.validateValueUint(strconv.ParseUint(s, 0, 64))

	case reflect.Bool:
		return p.validateValueBool(strconv.ParseBool(s))
	case reflect.Float32:
		return p.validateValueFloat(strconv.ParseFloat(s, 32))
	case reflect.Float64:
		return p.validateValueFloat(strconv.ParseFloat(s, 64))
	}
	return nil, errors.New("unknown type")
}

func (p *Parameter) validateEnum(v interface{}) bool {
	if p.Enum == nil {
		return true
	}

	for _, e := range p.Enum {
		if v == e {
			return true
		}
	}

	return false
}

func (p *Parameter) validateValueString(v string, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	if p.MinLength != nil || p.MaxLength != nil {
		if len(v) < *p.MinLength {
			return nil, errTooShort
		} else if len(v) > *p.MaxLength {
			return nil, errTooLong
		}
	}
	if p.regex != nil {
		if !p.regex.MatchString(v) {
			return nil, errBadPattern
		}
	}

	retElem := reflect.New(reflect.TypeOf(p.Model)).Elem()
	retElem.SetString(v)
	ret := retElem.Interface()
	if !p.validateEnum(ret) {
		return nil, errBadEnum
	}

	return ret, nil
}

func (p *Parameter) validateValueInt(v int64, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	if p.Minimum != nil && v < reflect.ValueOf(p.Minimum).Int() {
		return nil, errLTMin
	} else if p.Maximum != nil && v > reflect.ValueOf(p.Maximum).Int() {
		return nil, errGTMax
	}

	retElem := reflect.New(reflect.TypeOf(p.Model)).Elem()
	retElem.SetInt(v)
	ret := retElem.Interface()
	if !p.validateEnum(ret) {
		return nil, errBadEnum
	}

	return ret, nil
}

func (p *Parameter) validateValueUint(v uint64, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	if p.Minimum != nil && v < reflect.ValueOf(p.Minimum).Uint() {
		return nil, errLTMin
	} else if p.Maximum != nil && v > reflect.ValueOf(p.Maximum).Uint() {
		return nil, errGTMax
	}

	retElem := reflect.New(reflect.TypeOf(p.Model)).Elem()
	retElem.SetUint(v)
	ret := retElem.Interface()
	if !p.validateEnum(ret) {
		return nil, errBadEnum
	}

	return ret, nil
}

func (p *Parameter) validateValueBool(v bool, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	retElem := reflect.New(reflect.TypeOf(p.Model)).Elem()
	retElem.SetBool(v)
	ret := retElem.Interface()
	if !p.validateEnum(ret) {
		return nil, errBadEnum
	}

	return ret, nil
}

func (p *Parameter) validateValueFloat(v float64, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	if p.Minimum != nil && v < reflect.ValueOf(p.Minimum).Float() {
		return nil, errLTMin
	} else if p.Maximum != nil && v > reflect.ValueOf(p.Maximum).Float() {
		return nil, errGTMax
	}

	retElem := reflect.New(reflect.TypeOf(p.Model)).Elem()
	retElem.SetFloat(v)
	ret := retElem.Interface()
	if !p.validateEnum(ret) {
		return nil, errBadEnum
	}

	return ret, nil
}
