package main

import (
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/restfulspec"
	"github.com/tangblue/goapi/spec"

	"./secret"
)

func main() {
	port := ":8080"
	baseURL := "http://localhost"
	ip, err := externalIP()
	if err == nil {
		baseURL = "http://" + ip
	}
	baseURL = baseURL + port

	auth := NewAuth(secret.AuthKey)
	restful.DefaultContainer.Add(auth.WebService("/login", []string{"authentication"}))

	u := NewUserResource(auth)
	restful.DefaultContainer.Add(u.WebService("/users", []string{"users"}))

	swaggerJson := "/apidocs.json"
	config := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(),
		APIPath:     swaggerJson,
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	swaggerPath := "/apidocs/"
	http.Handle(swaggerPath, http.StripPrefix(swaggerPath, http.FileServer(http.Dir("./swagger-ui/dist"))))

	// Optionally, you may need to enable CORS for the UI to work.
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer}
	restful.DefaultContainer.Filter(cors.Filter)

	swaggerJson = baseURL + swaggerJson
	log.Printf("Get the API: " + swaggerJson)
	log.Printf("Swagger UI : " + baseURL + swaggerPath + "?url=" + swaggerJson)
	log.Fatal(http.ListenAndServe(port, nil))
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "UserService",
			Description: "Resource for managing Users",
			Contact: &spec.ContactInfo{
				Name:  "user",
				Email: "user@example.com",
				URL:   "http://example.com",
			},
			License: &spec.License{
				Name: "MIT",
				URL:  "http://mit.org",
			},
			Version: "1.0.0",
		},
	}
	swo.Tags = []spec.Tag{
		spec.Tag{
			TagProps: spec.TagProps{
				Name:        "authentication",
				Description: "Authentication",
			},
		},
		spec.Tag{
			TagProps: spec.TagProps{
				Name:        "users",
				Description: "Managing users",
			},
		},
	}
	gOAuth2 := spec.OAuth2AccessToken("https://accounts.google.com/o/oauth2/auth", "https://accounts.google.com/o/oauth2/token")
	gOAuth2.AddScope("userinfo.email", "https://www.googleapis.com/auth/userinfo.email")
	swo.SecurityDefinitions = spec.SecurityDefinitions{
		"Basic":         spec.BasicAuth(),
		"Bearer":        spec.APIKeyAuth("Authorization", "head"),
		"google_oauth2": gOAuth2,
	}
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}
