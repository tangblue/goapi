package restfulspec

import (
	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/spec"
)

type swaggerBuilder struct {
	def   definitionBuilder
	param parameterBuilder
	resp  responseBuilder
}

func (b *swaggerBuilder) buildParameter(restfulParam *restful.Parameter, pattern string) spec.Parameter {
	return b.param.build(restfulParam, pattern, &b.def)
}

func (b *swaggerBuilder) buildResponse(e *restful.ResponseError) spec.Response {
	return b.resp.build(e, &b.def)
}

// NewOpenAPIService returns a new WebService that provides the API documentation of all services
// conform the OpenAPI documentation specifcation.
func NewOpenAPIService(config Config) *restful.WebService {

	ws := new(restful.WebService)
	ws.Path(config.APIPath)
	ws.Produce(restful.MIME_JSON)
	if config.DisableCORS {
		ws.Filter(enableCORS)
	}

	swagger := BuildSwagger(config)
	resource := specResource{swagger: swagger}
	ws.Route(ws.GET("/").Handler(resource.getSwagger))
	return ws
}

// BuildSwagger returns a Swagger object for all services' API endpoints.
func BuildSwagger(config Config) *spec.Swagger {
	// collect paths and model definitions to build Swagger object.
	paths := &spec.Paths{Paths: map[string]spec.PathItem{}}
	sb := &swaggerBuilder{}
	sb.def.Definitions = spec.Definitions{}

	for _, each := range config.WebServices {
		for path, item := range buildPaths(each, config, sb).Paths {
			paths.Paths[path] = item
		}
	}
	swagger := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Paths:       paths,
			Definitions: sb.def.getDefinitions(),
			Parameters:  sb.param.getRefParameters(&sb.def),
			Responses:   sb.resp.getRefResponses(&sb.def),
		},
	}
	if config.PostBuildSwaggerObjectHandler != nil {
		config.PostBuildSwaggerObjectHandler(swagger)
	}
	return swagger
}

func enableCORS(req *restful.Request, resp *restful.Response, next func(*restful.Request, *restful.Response)) {
	if origin := req.HeaderParameter(restful.HEADER_Origin); origin != "" {
		// prevent duplicate header
		if len(resp.Header().Get(restful.HEADER_AccessControlAllowOrigin)) == 0 {
			resp.AddHeader(restful.HEADER_AccessControlAllowOrigin, origin)
		}
	}
	next(req, resp)
}

// specResource is a REST resource to serve the Open-API spec.
type specResource struct {
	swagger *spec.Swagger
}

func (s specResource) getSwagger(req *restful.Request, resp *restful.Response) {
	resp.WriteAsJson(s.swagger)
}
