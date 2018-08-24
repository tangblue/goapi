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

func (p *Parameter) getValue(s []string, out interface{}) error {
	t := reflect.TypeOf(out).Elem()
	v := reflect.ValueOf(out).Elem()

	switch t.Kind() {
	case reflect.Slice:
		l := len(s)
		if v.Len() < l {
			v.Set(reflect.MakeSlice(t, l, l))
		}
		fallthrough
	case reflect.Array:
		l := len(s)
		if v.Len() < l {
			l = v.Len()
		}
		for i := 0; i < l; i++ {
			if err := p.getElemValue(s[i], v.Index(i)); err != nil {
				return err
			}
		}
	default:
		return p.getElemValue(s[0], v)
	}

	return nil
}

func (p *Parameter) getElemValue(s string, out reflect.Value) error {
	switch out.Type().Kind() {
	case reflect.String:
		return p.validateValueString(s, out)

	case reflect.Int8:
		return p.validateValueInt(s, 8, out)
	case reflect.Int16:
		return p.validateValueInt(s, 16, out)
	case reflect.Int, reflect.Int32:
		return p.validateValueInt(s, 32, out)
	case reflect.Int64:
		return p.validateValueInt(s, 64, out)

	case reflect.Uint8:
		return p.validateValueUint(s, 8, out)
	case reflect.Uint16:
		return p.validateValueUint(s, 16, out)
	case reflect.Uint, reflect.Uint32:
		return p.validateValueUint(s, 32, out)
	case reflect.Uint64:
		return p.validateValueUint(s, 64, out)

	case reflect.Bool:
		return p.validateValueBool(s, out)

	case reflect.Float32:
		return p.validateValueFloat(s, 32, out)
	case reflect.Float64:
		return p.validateValueFloat(s, 64, out)
	}

	return errors.New("unknown type")
}

func (p *Parameter) validateEnum(v reflect.Value) error {
	if p.Enum == nil {
		return nil
	}

	vi := v.Interface()
	for _, e := range p.Enum {
		if vi == e {
			return nil
		}
	}

	return errBadEnum
}

func (p *Parameter) validateValueString(v string, out reflect.Value) error {
	if p.MinLength != nil && len(v) < *p.MinLength {
		return errTooShort
	} else if p.MaxLength != nil && len(v) > *p.MaxLength {
		return errTooLong
	} else if p.regex != nil && !p.regex.MatchString(v) {
		return errBadPattern
	}

	out.SetString(v)

	return p.validateEnum(out)
}

func (p *Parameter) validateValueInt(s string, bits int, out reflect.Value) error {
	if v, err := strconv.ParseInt(s, 0, bits); err != nil {
		return err
	} else if p.Minimum != nil && v < reflect.ValueOf(p.Minimum).Int() {
		return errLTMin
	} else if p.Maximum != nil && v > reflect.ValueOf(p.Maximum).Int() {
		return errGTMax
	} else {
		out.SetInt(v)
	}

	return p.validateEnum(out)
}

func (p *Parameter) validateValueUint(s string, bits int, out reflect.Value) error {
	if v, err := strconv.ParseUint(s, 0, bits); err != nil {
		return err
	} else if p.Minimum != nil && v < reflect.ValueOf(p.Minimum).Uint() {
		return errLTMin
	} else if p.Maximum != nil && v > reflect.ValueOf(p.Maximum).Uint() {
		return errGTMax
	} else {
		out.SetUint(v)
	}

	return p.validateEnum(out)
}

func (p *Parameter) validateValueBool(s string, out reflect.Value) error {
	if v, err := strconv.ParseBool(s); err != nil {
		return err
	} else {
		out.SetBool(v)
	}

	return p.validateEnum(out)
}

func (p *Parameter) validateValueFloat(s string, bits int, out reflect.Value) error {
	if v, err := strconv.ParseFloat(s, bits); err != nil {
		return err
	} else if p.Minimum != nil && v < reflect.ValueOf(p.Minimum).Float() {
		return errLTMin
	} else if p.Maximum != nil && v > reflect.ValueOf(p.Maximum).Float() {
		return errGTMax
	} else {
		out.SetFloat(v)
	}

	return p.validateEnum(out)
}
