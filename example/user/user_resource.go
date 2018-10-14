package main

import (
	"log"
	"net/http"

	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/restfulspec"
)

type UID int
type User struct {
	ID   UID    `json:"id" description:"identifier of the user" default:"1"`
	Name string `json:"name" description:"name of the user" default:"john"`
	Age  int    `json:"age" description:"age of the user" default:"21"`
}

type UserResource struct {
	auth *Auth

	paramUID          *restful.Parameter
	errorBadUserID    *restful.ResponseError
	errorUserNotFound *restful.ResponseError
	// normally one would use DAO (data access object)
	users map[UID]User
}

func NewUserResource(auth *Auth) *UserResource {
	paramUID := restful.PathParameter("userID", "identifier of the user").
		SetRefName("userID").DataType(UID(0))
	paramUID.CommonValidations.
		WithMinimum(UID(0), false).
		WithMaximum(UID(10), false)

	return &UserResource{
		auth: auth,

		paramUID:          paramUID,
		errorBadUserID:    restful.NewResponseError(http.StatusBadRequest, "User ID is invalid.", nil).SetRefName("BadUserID"),
		errorUserNotFound: restful.NewResponseError(http.StatusNotFound, "Not Found", nil).SetRefName("UserNotFound"),
		users:             map[UID]User{},
	}
}

func (u *UserResource) WebService(path string, tags []string) *restful.WebService {
	printPath := func(req *restful.Request, resp *restful.Response, next func(*restful.Request, *restful.Response)) {
		log.Printf("Path: %v", req.Request.URL.Path)
		next(req, resp)
	}
	tagUsers := func(b *restful.RouteBuilder) {
		b.Metadata(restfulspec.KeyOpenAPITags, tags)
	}

	ws := new(restful.WebService)
	ws.Path(path).
		Consumes(restful.MIME_JSON, restful.MIME_XML).
		Produces(restful.MIME_JSON, restful.MIME_XML).
		Filter(printPath)

	resp := restful.NewResponseError(200, "OK", []User{}).Header("x-google-x", "desc", UID(0))
	ws.Route(ws.GET("/").Doc("get all users").
		Handler(u.findAllUsers).
		ReturnResponses(resp).
		Do(tagUsers, u.auth.BasicAuth))

	ws.Route(ws.PUT("").Doc("create a user").
		Handler(u.createUser).
		Read(User{}).
		Return(http.StatusCreated, "Created", User{}).
		Do(tagUsers, u.auth.JWTAuth))

	ws.Route(ws.GET("/{%s}", u.paramUID).Doc("get a user").
		Handler(u.findUser).
		ReturnResponses(u.errorBadUserID, u.errorUserNotFound).
		Return(http.StatusOK, "OK", User{}).
		Do(tagUsers))

	ws.Route(ws.PUT("/{%s}", u.paramUID).Doc("update a user").
		Handler(u.updateUser).
		Read(User{}).
		ReturnResponses(u.errorBadUserID, u.errorUserNotFound).
		Return(http.StatusOK, "OK", User{}).
		Do(tagUsers, u.auth.JWTAuth))

	ws.Route(ws.DELETE("/{%s}", u.paramUID).Doc("delete a user").
		Handler(u.removeUser).
		ReturnResponses(u.errorBadUserID, u.errorUserNotFound).
		Return(http.StatusNoContent, "No Content", nil).
		Do(tagUsers, u.auth.JWTAuth))

	return ws
}

func (u *UserResource) findAllUsers(req *restful.Request, resp *restful.Response) {
	list := []User{}
	for _, each := range u.users {
		list = append(list, each)
	}
	resp.WriteEntity(list)
}

func (u *UserResource) findUser(req *restful.Request, resp *restful.Response) {
	var id UID
	err := req.GetParameter(u.paramUID, &id)
	if err != nil {
		resp.WriteErrorResponse(u.errorBadUserID)
		return
	}

	if usr, ok := u.users[id]; !ok {
		resp.WriteErrorResponse(u.errorUserNotFound)
	} else {
		resp.WriteEntity(usr)
	}
}

func (u *UserResource) updateUser(req *restful.Request, resp *restful.Response) {
	var id UID
	err := req.GetParameter(u.paramUID, &id)
	if err != nil {
		resp.WriteErrorResponse(u.errorBadUserID)
		return
	}

	usr, ok := u.users[id]
	if !ok {
		resp.WriteErrorResponse(u.errorUserNotFound)
		return
	}

	if err := req.ReadEntity(&usr); err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}

	usr.ID = id
	u.users[id] = usr
	resp.WriteEntity(usr)
}

func (u *UserResource) createUser(req *restful.Request, resp *restful.Response) {
	usr := User{}
	if err := req.ReadEntity(&usr); err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}
	u.users[usr.ID] = usr
	resp.WriteHeaderAndEntity(http.StatusCreated, usr)
}

func (u *UserResource) removeUser(req *restful.Request, resp *restful.Response) {
	var id UID
	err := req.GetParameter(u.paramUID, &id)
	if err != nil {
		resp.WriteErrorResponse(u.errorBadUserID)
		return
	}

	delete(u.users, id)
	resp.WriteHeader(http.StatusNoContent)
}
