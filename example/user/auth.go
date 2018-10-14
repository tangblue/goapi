package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/restfulspec"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"./secret"
)

type LoginInfo struct {
	Name     string `json:"name" description:"user name"`
	Password string `json:"password" description:"password"`
}

type JWTToken struct {
	Token string `json:"token" description:"JWT token"`
}

type Auth struct {
	secret string

	paramAuth   *restful.Parameter
	paramOAuth2 *restful.Parameter
	paramCode   *restful.Parameter
	paramData   *restful.Parameter

	errorAuth *restful.ResponseError
}

func NewAuth(secret string) *Auth {
	paramAuth := restful.HeaderParameter("authorization", "JWT in authorization header").
		Regex(`[Bb]earer \w+\.\w+\.\w+`).
		DataType("Bearer ")
	paramAuth.CommonValidations.
		WithMinLength(8).
		WithMaxLength(128)

	paramOAuth2 := restful.QueryParameter("oauth2", "OAuth2")
	paramOAuth2.CommonValidations.WithEnum("google")

	paramCode := restful.QueryParameter("code", "OAuth2 code")
	paramCode.AsRequired()
	paramCode.CommonValidations.
		WithMinLength(8).
		WithMaxLength(128)

	paramData := restful.QueryParameter("data", "qr data")
	paramData.AsRequired()

	return &Auth{
		secret:      secret,
		paramAuth:   paramAuth,
		paramOAuth2: paramOAuth2,
		paramCode:   paramCode,
		paramData:   paramData,
		errorAuth:   restful.NewResponseError(http.StatusUnauthorized, "Not Authorized", nil).SetRefName("Unauthorized"),
	}
}

func (a *Auth) WebService(path string, tags []string) *restful.WebService {
	ws := new(restful.WebService)
	ws.Path(path).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").Doc("login with OAuth2").
		Handler(a.loginOAuth2).
		Params(a.paramOAuth2).
		Return(http.StatusOK, "OK", JWTToken{}).
		Return(http.StatusInternalServerError, "Internal Server Error", nil).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("").Doc("login user").
		Handler(a.loginUser).
		Read(LoginInfo{}).
		Return(http.StatusOK, "OK", JWTToken{}).
		Return(http.StatusInternalServerError, "Internal Server Error", nil).
		Return(http.StatusUnprocessableEntity, "Bad user name or password", nil).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/auth").Doc("oauth2 google").
		Handler(a.oauth2).
		Params(a.paramCode).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/qr").Doc("qr code").
		Handler(a.qr).
		Params(a.paramData).
		Produces("image/png").
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

var conf = &oauth2.Config{
	ClientID:     secret.ClientID,
	ClientSecret: secret.ClientSecret,
	RedirectURL:  "http://127.0.0.1:8080/login/auth",
	Scopes: []string{
		"https://www.googleapis.com/auth/userinfo.email",
	},
	Endpoint: google.Endpoint,
}

func (a *Auth) oauth2(req *restful.Request, resp *restful.Response) {
	var code string
	if err := req.GetParameter(a.paramCode, &code); err != nil {
		log.Println(err)
		return
	}

	tok, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Println(err)
		return
	}

	client := conf.Client(oauth2.NoContext, tok)
	userinfo, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		log.Println(err)
		return
	}
	defer userinfo.Body.Close()
	u := struct {
		Sub           string `json:"sub"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Profile       string `json:"profile"`
		Picture       string `json:"picture"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Gender        string `json:"gender"`
	}{}
	data, _ := ioutil.ReadAll(userinfo.Body)
	if err = json.Unmarshal(data, &u); err != nil {
		log.Println(err)
		return
	}
	log.Printf("user: %#v\n", u)

	resp.WriteEntity(a.createJWTToken(u.Email))
}

func (a *Auth) basicAuthenticate(req *restful.Request, resp *restful.Response, next func(*restful.Request, *restful.Response)) {
	// usr/pwd = admin/admin
	u, p, ok := req.Request.BasicAuth()
	if !ok || u != "admin" || p != "admin" {
		resp.AddHeader("WWW-Authenticate", "Basic realm=Protected Area")
		resp.WriteErrorString(http.StatusUnauthorized, "401: Not Authorized")
		return
	}
	next(req, resp)
}

func (a *Auth) loginOAuth2(req *restful.Request, resp *restful.Response) {
	var vendor string
	if err := req.GetParameter(a.paramOAuth2, &vendor); err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}
	if vendor == "google" {
		http.Redirect(resp.ResponseWriter, req.Request, conf.AuthCodeURL("state"), http.StatusFound)
		return
	}
	resp.WriteError(http.StatusInternalServerError, errors.New("Unknow vender"))
}

func (a *Auth) loginUser(req *restful.Request, resp *restful.Response) {
	li := LoginInfo{}
	if err := req.ReadEntity(&li); err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}
	if len(li.Name) <= 0 {
		resp.WriteError(http.StatusInternalServerError, errors.New(""))
		return
	}
	resp.WriteEntity(a.createJWTToken(li.Name))
}

func (a *Auth) qr(req *restful.Request, resp *restful.Response) {
	var data string
	if err := req.GetParameter(a.paramData, &data); err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}

	png, err := qrcode.Encode(data, qrcode.Medium, 256)
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp.AddHeader("Content-Type", "image/png")
	resp.AddHeader("Content-Length", strconv.Itoa(len(png)))
	resp.Write(png)
}

func (a *Auth) createJWTToken(sub string) JWTToken {
	log.Printf("sub: %v\n", sub)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
	})
	tokenString, _ := token.SignedString([]byte(a.secret))
	return JWTToken{Token: tokenString}
}

func (a *Auth) validateJWTToken(req *restful.Request) *jwt.Token {
	var ah string
	if err := req.GetParameter(a.paramAuth, &ah); err != nil {
		log.Printf("Error in parameter {%s}: %s", a.paramAuth, err)
		return nil
	}
	bt := strings.Fields(ah)
	if len(bt) != 2 || !strings.EqualFold(bt[0], "bearer") {
		return nil
	}

	token, err := jwt.Parse(bt[1], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("There was an error")
		}
		return []byte(a.secret), nil
	})
	if err != nil || !token.Valid {
		return nil
	}

	return token
}

func (a *Auth) JWTAuthenticate(req *restful.Request, resp *restful.Response, next func(*restful.Request, *restful.Response)) {
	token := a.validateJWTToken(req)
	if token == nil {
		resp.WriteErrorString(http.StatusUnauthorized, "401: Not Authorized")
		return
	}

	log.Printf("Claims: %v", token.Claims)
	next(req, resp)
}

func (a *Auth) BasicAuth(b *restful.RouteBuilder) {
	b.Filter(a.basicAuthenticate).
		Security("Basic", []string{}).
		ReturnResponses(a.errorAuth)
}

func (a *Auth) JWTAuth(b *restful.RouteBuilder) {
	b.Filter(a.JWTAuthenticate).
		Security("Bearer", []string{}).
		ReturnResponses(a.errorAuth)
}
