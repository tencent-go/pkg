package tsdoc

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"text/template"
	"unicode"

	"github.com/tencent-go/pkg/doc/restdoc"
	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
)

func NewRestApiFiles(groups []restdoc.Group, parentDir ...string) []util.DataFile {
	if len(parentDir) == 0 {
		return nil
	}
	var providerTmp = `
export interface RequestProps {
  ignoreAuth?: boolean;
  method: 'DELETE' | 'GET' | 'POST' | 'PUT' | 'PATCH' | 'OPTIONS' | 'HEAD';
  url: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data?: any;
  header?: Record<string, string>;
}

export type Request = (props: RequestProps) => Promise<any>;

let provider: Request | undefined;

export function setRequestProvider(p: Request): void {
  provider = p;
}

export function request(props: RequestProps): Promise<any> {
  if (!provider) {
    throw new Error('Request provider is not initialized. Please ensure that the provider is properly configured before making requests.');
  }
  return provider(props);
}
`
	var res = []util.DataFile{
		{
			Name: "_provider.ts",
			Data: []byte(providerTmp),
			Dir:  path.Join(parentDir...),
		},
	}
	var interfaceTmp = `
import { request } from "./_provider";

{{range .Items}}
{{if .Description -}}
// {{.Description}}
{{end -}}
export function {{.FuncName}}({{.Params}}): Promise<{{.ReturnType}}> {
    return request({ {{.RequestParams}} });
}
{{- if .ResourceID}}{{.FuncName}}.permissionKey = '{{.ResourceID}}';{{end}}
{{end}}
{{if .Description}}// {{.Description}}{{end}}
const {{.GroupName}} = {
    {{$isFirst := true }}
{{- range $index, $item := .Items }}
    {{- if not $isFirst }}, {{end}}
    {{- $item.FuncName }}
    {{- $isFirst = false }}
{{- end }}
}

export default {{.GroupName}}
`

	t, err := template.New("typescript_interfaces").Parse(interfaceTmp)
	if err != nil {
		logrus.Fatalf("parse interface template failed: %v", err)
	}

	for _, group := range groups {
		data := &interfaceData{}
		data.parseGroup(group)
		var buf bytes.Buffer
		if err = t.Execute(&buf, data); err != nil {
			logrus.Fatalf("execute interface template failed: %v", err)
		}
		f := util.DataFile{
			Dir:  path.Join(parentDir...),
			Name: group.Name + ".ts",
			Data: buf.Bytes(),
		}
		res = append(res, f)
	}
	return res
}

type interfaceData struct {
	Items       []funcItem
	GroupName   string
	Description string
}

func (g *interfaceData) parseGroup(group restdoc.Group) {
	g.Description = group.Description
	g.GroupName = convertName(group.Name) + "Api"
	g.Items = make([]funcItem, 0, len(group.Endpoints))
	for _, endpoint := range group.Endpoints {
		item := &funcItem{}
		item.parseEndpoint(endpoint)
		g.Items = append(g.Items, *item)
	}
}

type funcItem struct {
	FuncName      string
	RequestParams string
	ReturnType    string
	Params        string
	Description   string
	ResourceID    string
}

func (item *funcItem) parseEndpoint(a restdoc.Endpoint) {
	item.FuncName = convertName(a.Name)
	item.Description = a.Description
	item.ResourceID = a.Permission
	if !a.AuthenticationRequired || !a.AuthorizationRequired {
		item.ResourceID = ""
	}
	//params
	var params []string
	if a.Body != nil {
		params = append(params, "data: "+parseType(*a.Body, nil))
	}
	if a.Param != nil && len(a.Param.Fields) > 0 {
		v := fmt.Sprintf("pathParams: %s.%s", a.Param.Package.Name, a.Param.Name)
		params = append(params, v)
	}
	if a.Query != nil {
		v := fmt.Sprintf("query: %s.%s", a.Query.Package.Name, a.Query.Name)
		params = append(params, v)
	}
	if a.Header != nil {
		v := fmt.Sprintf("header: %s.%s", a.Query.Package.Name, a.Header.Name)
		params = append(params, v)
	}
	item.Params = strings.Join(params, ", ")
	item.ReturnType = "void"
	//generics
	if a.Response != nil {
		if t := parseType(*a.Response, nil); t != "" && t != "null" {
			item.ReturnType = t
		}
	}

	//request params
	reqParams := []string{fmt.Sprintf("method: '%s'", a.Method)}
	{
		url := path.Join("/", a.Path)
		if a.Param != nil && len(a.Param.Fields) > 0 {
			url = strings.ReplaceAll(url, "{", "${pathParams.")
		}
		if a.Query != nil {
			url += "?${new URLSearchParams(query as any)}"
		}
		reqParams = append(reqParams, fmt.Sprintf("url: `%s`", url))
	}
	if !a.AuthenticationRequired {
		reqParams = append(reqParams, "ignoreAuth: true")
	}
	if a.Body != nil {
		reqParams = append(reqParams, "data")
	}

	if a.Header != nil {
		reqParams = append(reqParams, "header")
	}
	item.RequestParams = strings.Join(reqParams, ", ")
}

func getGenerics(resType schema.Type, wrapperType *schema.Class) string {
	t := parseType(resType, nil)
	if wrapperType == nil {
		return fmt.Sprintf("<%s>", t)
	} else {
		return fmt.Sprintf("<%s.%s<%s>>", wrapperType.Package.Name, wrapperType.Name, t)
	}
}

func convertName(s string) string {
	var result []rune
	uppercaseNext := false
	symbols := map[rune]bool{'_': true, '-': true}
	for _, char := range s {
		if symbols[char] {
			uppercaseNext = true
		} else {
			if uppercaseNext {
				result = append(result, unicode.ToUpper(char))
				uppercaseNext = false
			} else {
				result = append(result, char)
			}
		}
	}
	return string(result)
}
