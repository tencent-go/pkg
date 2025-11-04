package rpcdoc

import (
	"path"
	"strings"

	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/rpc"
	"github.com/tencent-go/pkg/util"
)

type Group struct {
	Name        string
	Description string
	Methods     []Method
}

type Method struct {
	Name         string
	Path         string
	Description  string
	RequestType  *schema.Type
	ResponseType *schema.Type
}

func NewGroup(schemaCollection schema.Collection, rpcMethodGroups []rpc.Group) []Group {
	var groups []Group
	for _, rpcGroup := range rpcMethodGroups {
		g := Group{
			Name:        strings.ReplaceAll(strings.Trim(rpcGroup.Path, "/"), "/", "-"),
			Description: rpcGroup.Description,
		}
		for _, route := range rpcGroup.Routes {
			m := Method{
				Name:        strings.ReplaceAll(strings.Trim(route.Path(), "/"), "/", "-"),
				Path:        path.Join("/", rpcGroup.Path, route.Path()),
				Description: route.Description(),
			}
			if t, ok := schemaCollection.ParseAndGetType(route.InputType(), util.TagJson); ok {
				m.RequestType = t
			}
			if t, ok := schemaCollection.ParseAndGetType(route.OutputType(), util.TagJson); ok {
				m.ResponseType = t
			}
			g.Methods = append(g.Methods, m)
		}
		groups = append(groups, g)
	}
	return groups
}
