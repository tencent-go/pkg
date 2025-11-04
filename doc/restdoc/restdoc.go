package restdoc

import (
	"regexp"
	"sort"
	"strings"

	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/rest/api"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
)

type Group struct {
	Name        string
	Description string
	Endpoints   []Endpoint
}

type Endpoint struct {
	Name                   string
	Permission             string
	Description            string
	Method                 api.Method
	Path                   string
	AuthenticationRequired bool
	AuthorizationRequired  bool
	Query                  *schema.Class
	Param                  *schema.Class // path variables
	Header                 *schema.Class
	Body                   *schema.Type
	Response               *schema.Type
}

func NewGroups(schemaCollection schema.Collection, routes []api.Route, permCollection api.PermissionProvider) []Group {
	f := &factory{
		Collection:         schemaCollection,
		groupNameSeparator: "_",
		groupMap:           make(map[string]*group),
		permCollection:     permCollection,
	}
	for _, route := range routes {
		f.parseRoute(route)
	}
	var res []Group
	{
		groupNames := make([]string, 0, len(f.groupMap))
		for k, g := range f.groupMap {
			if len(g.endpoints) == 0 {
				continue
			}
			groupNames = append(groupNames, k)
		}
		sort.Strings(groupNames)
		for _, key := range groupNames {
			g := f.groupMap[key]
			srv := Group{
				Name:        key,
				Description: g.description,
				Endpoints:   make([]Endpoint, 0, len(g.endpoints)),
			}
			var names []string
			for k := range g.endpoints {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, k := range names {
				srv.Endpoints = append(srv.Endpoints, *g.endpoints[k])
			}
			res = append(res, srv)
		}
	}
	return res
}

type group struct {
	description string
	endpoints   map[string]*Endpoint
}

type factory struct {
	schema.Collection
	groupNameSeparator string
	groupMap           map[string]*group
	permCollection     api.PermissionProvider
}

func (f *factory) parseRoute(route api.Route) {
	var g *group
	{
		groupName := f.getGroupName(route.Ancestors())
		gr, exists := f.groupMap[groupName]
		if !exists {
			gr = &group{
				endpoints:   make(map[string]*Endpoint),
				description: getGroupDescription(route.Ancestors()),
			}
			f.groupMap[groupName] = gr
		}
		g = gr
	}
	var name string
	{
		if n := route.Endpoint().Name(); n != nil {
			name = formatEndpointName(*n)
		} else {
			logrus.Fatalf("api name is empty: %s", route)
		}
		if _, exists := g.endpoints[name]; exists {
			logrus.Fatalf("api name '%s' is duplicated,path %s", name, route.Path())
		}
	}
	end := Endpoint{
		Name:                   name,
		Method:                 route.Endpoint().Method(),
		Path:                   route.Path(),
		AuthenticationRequired: route.RequireAuthentication(),
		AuthorizationRequired:  route.RequireAuthorization(),
	}
	if d := route.Endpoint().Description(); d != nil {
		end.Description = *d
	}
	if f.permCollection != nil && route.RequireAuthorization() {
		p, ok := f.permCollection.GetEndpointPermission(route.Endpoint())
		if ok {
			end.AuthenticationRequired = true
			end.Permission = string(*p)
		}
	}
	iType := route.Endpoint().InputType()
	if t, ok := f.ParseAndGetType(iType, util.TagPath); ok {
		end.Param = t.Class
		if t.Class != nil {
			checkPathParams(end.Path, len(t.Class.Fields))
		}
	}
	if t, ok := f.ParseAndGetType(iType, util.TagQuery); ok {
		end.Query = t.Class
	}
	if t, ok := f.ParseAndGetType(iType, util.TagHeader); ok {
		end.Header = t.Class
	}
	if ct := route.RequestContentType(); ct != "" && route.Endpoint().Method() != api.MethodGet {
		switch route.RequestContentType() {
		case api.ContentTypeApplicationJson:
			if t, ok := f.ParseAndGetType(iType, util.TagJson); ok {
				end.Body = t
			}
		case api.ContentTypeApplicationFormUrlencoded:
			if t, ok := f.ParseAndGetType(iType, util.TagForm); ok {
				end.Body = t
			}
		}
	}

	if ct := route.ResponseContentType(); ct != "" {
		oType := route.Endpoint().OutputType()
		switch ct {
		case api.ContentTypeApplicationJson:
			if t, ok := f.ParseAndGetType(oType, util.TagJson); ok {
				end.Response = t
			}
		case api.ContentTypeApplicationFormUrlencoded:
			if t, ok := f.ParseAndGetType(oType, util.TagForm); ok {
				end.Response = t
			}
		}
	}
	g.endpoints[name] = &end
}

func checkPathParams(path string, expected int) {
	re := regexp.MustCompile(`\{(.+?)\}`)
	actual := len(re.FindAllString(path, -1))
	if expected != actual {
		logrus.Fatalf(
			"path '%s' has %d params, but expected %d",
			path, actual, expected,
		)
	}
}

func (f *factory) getGroupName(ancestors []api.Node) string {
	var namePath []string
	for _, node := range ancestors {
		if name := node.Name(); name != nil && *name != "" {
			namePath = append(namePath, *name)
		}
	}
	if len(namePath) == 0 {
		return "root"
	}
	return strings.Join(namePath, f.groupNameSeparator)
}

func getGroupDescription(ancestors []api.Node) string {
	for i := len(ancestors) - 1; i >= 0; i-- {
		node := ancestors[i]
		if name := node.Name(); name != nil && *name != "" && node.Description() != nil {
			return *node.Description()
		}
	}
	return ""
}

func formatEndpointName(name string) string {
	if name == "" {
		return ""
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return name
	}
	result := strings.ToLower(parts[0])
	for _, part := range parts[1:] {
		if part != "" {
			result += strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	return result
}
