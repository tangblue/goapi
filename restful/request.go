package restful

// Copyright 2013 Ernest Micklei. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

import (
	"compress/zlib"
	"errors"
	"net/http"
)

var defaultRequestContentType string

// Request is a wrapper for a http Request that provides convenience methods
type Request struct {
	Request           *http.Request
	pathParameters    map[string]string
	attributes        map[string]interface{} // for storing request-scoped values
	selectedRoutePath string                 // root path + route path that matched the request, e.g. /meetings/{id}/attendees
}

func NewRequest(httpRequest *http.Request) *Request {
	return &Request{
		Request:        httpRequest,
		pathParameters: map[string]string{},
		attributes:     map[string]interface{}{},
	} // empty parameters, attributes
}

// If ContentType is missing or */* is given then fall back to this type, otherwise
// a "Unable to unmarshal content of type:" response is returned.
// Valid values are restful.MIME_JSON and restful.MIME_XML
// Example:
// 	restful.DefaultRequestContentType(restful.MIME_JSON)
func DefaultRequestContentType(mime string) {
	defaultRequestContentType = mime
}

// GetParameter accesses the parameter value by Parameter
func (r *Request) GetParameter(p *Parameter) (interface{}, error) {
	var v string
	var ok bool

	err := r.Request.ParseForm()
	if err != nil {
		return p.getDefaultValue(), err
	}

	name := p.getName()
	switch p.getKind() {
	case PathParameterKind:
		v, ok = r.pathParameters[name]
	case QueryParameterKind, FormParameterKind:
		va, found := r.Request.Form[name]
		if found {
			v = va[0]
			ok = true
		}
	case BodyParameterKind:
		va, found := r.Request.PostForm[name]
		if found {
			v = va[0]
			ok = true
		}
	case HeaderParameterKind:
		v, ok = r.Request.Header.Get(name), true
	}

	if !ok {
		if v := p.getDefaultValue(); p.isRequired() {
			return v, errors.New("not available")
		} else {
			return v, nil
		}
	}

	return p.getValue(v)
}

// HeaderParameter returns the HTTP Header value of a Header name or empty if missing
func (r *Request) HeaderParameter(name string) string {
	return r.Request.Header.Get(name)
}

// ReadEntity checks the Accept header and reads the content into the entityPointer.
func (r *Request) ReadEntity(entityPointer interface{}) (err error) {
	contentType := r.Request.Header.Get(HEADER_ContentType)
	contentEncoding := r.Request.Header.Get(HEADER_ContentEncoding)

	// check if the request body needs decompression
	if ENCODING_GZIP == contentEncoding {
		gzipReader := currentCompressorProvider.AcquireGzipReader()
		defer currentCompressorProvider.ReleaseGzipReader(gzipReader)
		gzipReader.Reset(r.Request.Body)
		r.Request.Body = gzipReader
	} else if ENCODING_DEFLATE == contentEncoding {
		zlibReader, err := zlib.NewReader(r.Request.Body)
		if err != nil {
			return err
		}
		r.Request.Body = zlibReader
	}

	// lookup the EntityReader, use defaultRequestContentType if needed and provided
	entityReader, ok := entityAccessRegistry.accessorAt(contentType)
	if !ok {
		if len(defaultRequestContentType) != 0 {
			entityReader, ok = entityAccessRegistry.accessorAt(defaultRequestContentType)
		}
		if !ok {
			return NewError(http.StatusBadRequest, "Unable to unmarshal content of type:"+contentType)
		}
	}
	return entityReader.Read(r, entityPointer)
}

// SetAttribute adds or replaces the attribute with the given value.
func (r *Request) SetAttribute(name string, value interface{}) {
	r.attributes[name] = value
}

// Attribute returns the value associated to the given name. Returns nil if absent.
func (r Request) Attribute(name string) interface{} {
	return r.attributes[name]
}

// SelectedRoutePath root path + route path that matched the request, e.g. /meetings/{id}/attendees
func (r Request) SelectedRoutePath() string {
	return r.selectedRoutePath
}
