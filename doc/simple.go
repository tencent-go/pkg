package doc

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/tencent-go/pkg/doc/openapi"
	"github.com/tencent-go/pkg/doc/restdoc"
	"github.com/tencent-go/pkg/doc/rpcdoc"
	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/doc/tsdoc"
	"github.com/tencent-go/pkg/doc/wsdoc"
	"github.com/tencent-go/pkg/rest/api"
	"github.com/tencent-go/pkg/rpc"
	"github.com/tencent-go/pkg/util"
	"github.com/tencent-go/pkg/wsx"
)

type Config struct {
	Rest      []api.Route
	Rpc       []rpc.Group
	Websocket []wsx.EventChannel
}

func NewSimpleHttpHandler(config Config) http.Handler {
	mux := http.NewServeMux()
	sc := schema.NewCollection()
	var tsFiles []util.DataFile
	swaggerTemp, e := template.New("swagger").Parse(swaggerHtml)
	if e != nil {
		panic(e)
	}
	if rest := config.Rest; len(rest) > 0 {
		model := restdoc.NewGroups(sc, rest, nil)
		tsFiles = append(tsFiles, tsdoc.NewRestApiFiles(model, "restapi")...)
		swagger := openapi.NewDefault()
		swagger.ExternalDocs = &openapi.ExternalDocumentation{
			Description: "Typescript",
			URL:         "../ts.zip",
		}
		swagger.Parse(model)
		swagger.Info.Title = "Restful API"
		jsonFile, e := json.Marshal(swagger)
		if e != nil {
			panic(e)
		}
		var swaggerData []byte
		{
			buf := &bytes.Buffer{}
			if e = swaggerTemp.Execute(buf, swagger.Info.Title); e != nil {
				panic(e)
			}
			swaggerData = buf.Bytes()
		}
		mux.HandleFunc("/rest", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "rest/index.html")
			w.WriteHeader(http.StatusFound)
		})
		mux.HandleFunc("/rest/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "index.html")
			w.WriteHeader(http.StatusFound)
		})
		mux.HandleFunc("/rest/index.html", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(swaggerData)
		})
		mux.HandleFunc("/rest/doc.json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jsonFile)
		})
	}
	if rpcGroups := config.Rpc; len(rpcGroups) > 0 {
		model := rpcdoc.NewGroup(sc, rpcGroups)
		tsFiles = append(tsFiles, tsdoc.NewRpcApiFiles(model, "rpc")...)
		swagger := openapi.NewDefault()
		swagger.ExternalDocs = &openapi.ExternalDocumentation{
			Description: "Typescript",
			URL:         "../ts.zip",
		}
		var formated []restdoc.Group
		for _, group := range model {
			g := restdoc.Group{
				Name:        group.Name,
				Description: group.Description,
			}
			for _, method := range group.Methods {
				end := restdoc.Endpoint{
					Name:        method.Name,
					Path:        method.Path,
					Description: method.Description,
					Method:      api.MethodPost,
					Body:        method.RequestType,
					Response:    method.ResponseType,
				}
				g.Endpoints = append(g.Endpoints, end)
			}
			formated = append(formated, g)
		}
		swagger.Parse(formated)
		swagger.Info.Title = "RPC API"
		jsonFile, e := json.Marshal(swagger)
		if e != nil {
			panic(e)
		}
		var swaggerData []byte
		{
			buf := &bytes.Buffer{}
			if e = swaggerTemp.Execute(buf, swagger.Info.Title); e != nil {
				panic(e)
			}
			swaggerData = buf.Bytes()
		}
		mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "rpc/index.html")
			w.WriteHeader(http.StatusFound)
		})
		mux.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "index.html")
			w.WriteHeader(http.StatusFound)
		})
		mux.HandleFunc("/rpc/index.html", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(swaggerData)
		})
		mux.HandleFunc("/rpc/doc.json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jsonFile)
		})
	}
	if ws := config.Websocket; len(ws) > 0 {
		model := wsdoc.NewGroups(sc, ws)
		tsFiles = append(tsFiles, tsdoc.NewEventFile(model, nil, "websocket")...)
	}
	tsFiles = append(tsFiles, tsdoc.NewDictionariesFile(sc.Packages()))
	tsFiles = append(tsFiles, tsdoc.NewSchemaFiles(sc.Packages(), "schema")...)
	tsZipFile, err := util.ZipFilesBytes(tsFiles)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("/ts.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tsZipFile)
	})
	return mux
}

var swaggerHtml = `<!DOCTYPE html>
<html>
  <head>
    <title>{{.}}</title>
    <link rel="icon" href="https://cdn.jsdelivr.net/gh/twitter/twemoji/2/72x72/1f600.png" type="image/png">
    <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.18.1/swagger-ui.css">
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.18.1/swagger-ui-bundle.js"></script>
    <script>
      const ui = SwaggerUIBundle({
        url: "doc.json",
        dom_id: '#swagger-ui',
        persistAuthorization: true,
      });
    </script>
  </body>
</html>`
