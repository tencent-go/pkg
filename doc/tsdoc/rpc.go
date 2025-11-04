package tsdoc

import (
	"bytes"
	"path"
	"text/template"

	"github.com/tencent-go/pkg/doc/rpcdoc"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
)

func NewRpcApiFiles(groups []rpcdoc.Group, parentDir ...string) []util.DataFile {
	if len(groups) == 0 {
		return nil
	}
	var providerTmp = `
export type RpcCaller = (path: string, params: any) => Promise<any>;

let provider: RpcCaller | undefined;

export function setRpcCaller(p: RpcCaller): void {
  provider = p;
}

export function callRpcMethod(path: string, params?: any): Promise<any> {
  if (!provider) {
    throw new Error(
      'Rpc provider is not initialized. Please ensure that the provider is properly configured before making requests.'
    );
  }
  return provider(path, params);
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
import { callRpcMethod } from './_provider';
{{range .Items}}
{{if .Description -}}
// {{.Description}}
{{end -}}
export function {{.FuncName}}({{if .Params}}data: {{.Params}}{{end}}):Promise<{{.ReturnType}}> {
    return callRpcMethod('{{.Path}}'{{if .Params}}, data{{end}});
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
		g := &rpcGroup{}
		g.parseGroup(group)
		var buf bytes.Buffer
		if err = t.Execute(&buf, g); err != nil {
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

type rpcGroup struct {
	Items       []rpcMethod
	GroupName   string
	Description string
}

func (g *rpcGroup) parseGroup(group rpcdoc.Group) {
	g.Description = group.Description
	g.GroupName = convertName(group.Name) + "Rpc"
	g.Items = make([]rpcMethod, 0, len(group.Methods))
	for _, method := range group.Methods {
		m := rpcMethod{}
		m.parse(method)
		g.Items = append(g.Items, m)
	}
}

type rpcMethod struct {
	FuncName    string
	Params      string
	ReturnType  string
	Description string
	ResourceID  string
	Path        string
}

func (m *rpcMethod) parse(method rpcdoc.Method) {
	m.FuncName = convertName(method.Name)
	if method.RequestType != nil {
		if t := parseType(*method.RequestType, nil); t != "" && t != "null" {
			m.Params = t
		}
	}
	m.ReturnType = "void"
	if method.ResponseType != nil {
		if t := parseType(*method.ResponseType, nil); t != "" && t != "null" {
			m.ReturnType = t
		}
	}
	m.Description = method.Description
	m.Path = method.Path
}
