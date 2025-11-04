package api

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tencent-go/pkg/errx"
)

func DefaultGroup() GroupBuilder {
	return NewGroup().
		WithRequireAuthentication(true).
		WithRequireWrapOutput(true).
		WithRequestContentType(ContentTypeApplicationJson).
		WithResponseContentType(ContentTypeApplicationJson)
}

func PrintRoutes(routes []Route) {
	if len(routes) == 0 {
		fmt.Println("\nRoutes: (empty)")
		return
	}
	maxPathLen := 0
	maxMethodLen := 0
	maxNameLen := 0
	for _, r := range routes {
		pathLen := len(r.Path())
		if pathLen > maxPathLen {
			maxPathLen = pathLen
		}
		methodLen := len(string(r.Endpoint().Method()))
		if methodLen > maxMethodLen {
			maxMethodLen = methodLen
		}

		if n := r.Endpoint().Name(); n != nil {
			nameLen := len(*n)
			if nameLen > maxNameLen {
				maxNameLen = nameLen
			}
		}
	}
	content := strings.Builder{}
	content.WriteString("\nRoutes:\n")
	content.WriteString(strings.Repeat("-", maxNameLen+maxPathLen+maxMethodLen+10))
	content.WriteString("\n")
	for _, r := range routes {
		if n := r.Endpoint().Name(); n != nil {
			content.WriteString(fmt.Sprintf("%-*s", maxNameLen+2, *n))
		} else {
			content.WriteString(fmt.Sprintf("%-*s", maxNameLen+2, ""))
		}
		path := r.Path()
		content.WriteString(fmt.Sprintf("%-*s", maxPathLen+2, path))
		method := string(r.Endpoint().Method())
		content.WriteString(fmt.Sprintf("%-*s", maxMethodLen+2, method))
		content.WriteString("\n")
	}
	content.WriteString(strings.Repeat("-", maxNameLen+maxPathLen+maxMethodLen+10))
	content.WriteString("\n")
	fmt.Println(content.String())
}

type group struct {
	node
	children   []Node
	methodTrie map[Method]*pathTrieNode
	routes     []Route
}

func (g *group) copy() *group {
	children := make([]Node, len(g.children))
	copy(children, g.children)
	c := *g
	c.children = children
	return &c
}

func (g *group) Name() *string {
	if g.name != nil {
		return g.name
	}
	if g.path == nil || *g.path == "" {
		return nil
	}
	fields := strings.FieldsFunc(*g.path, func(r rune) bool {
		return r == '-' || r == '_' || r == '/' || r == '.'
	})
	var parts []string
	for _, f := range fields {
		if f == "" {
			continue
		}
		if strings.HasPrefix(f, "{") {
			continue
		}
		parts = append(parts, strings.ToLower(f))
	}
	name := strings.Join(parts, "-")
	return &name
}

func (g *group) Children() []Node {
	return g.children
}

func (g *group) WithName(name string) GroupBuilder {
	c := g.copy()
	c.name = &name
	c.resetTrie()
	return c
}

func (g *group) WithPath(path string) GroupBuilder {
	c := g.copy()
	c.path = &path
	c.resetTrie()
	return c
}

func (g *group) WithDescription(description string) GroupBuilder {
	c := g.copy()
	c.description = &description
	c.resetTrie()
	return c
}

func (g *group) WithRequestContentType(contentType ContentType) GroupBuilder {
	c := g.copy()
	c.requestContentType = &contentType
	c.resetTrie()
	return c
}

func (g *group) WithResponseContentType(contentType ContentType) GroupBuilder {
	c := g.copy()
	c.responseContentType = &contentType
	c.resetTrie()
	return c
}

func (g *group) WithRequireAuthentication(required bool) GroupBuilder {
	c := g.copy()
	c.requireAuthentication = &required
	c.resetTrie()
	return c
}

func (g *group) WithRequireAuthorization(required bool) GroupBuilder {
	c := g.copy()
	c.requireAuthorization = &required
	c.resetTrie()
	return c
}

func (g *group) WithRequireWrapOutput(required bool) GroupBuilder {
	c := g.copy()
	c.requireWrapOutput = &required
	c.resetTrie()
	return c
}

func (g *group) WithChildren(children ...Node) GroupBuilder {
	c := g.copy()
	s := make([]Node, 0, len(children))
	for _, child := range children {
		if child != nil {
			s = append(s, child)
		}
	}
	c.children = append(c.children, s...)
	c.resetTrie()
	return c
}

func (g *group) resetTrie() {
	g.methodTrie = nil
	g.routes = nil
	if len(g.children) == 0 {
		return
	}
	var routes []route
	parseRoutes(g, &routes, []Node{}, []string{})
	if len(routes) == 0 {
		return
	}
	g.methodTrie = make(map[Method]*pathTrieNode)
	for i := range routes {
		r := &routes[i]
		g.routes = append(g.routes, r)
		method := r.endpoint.Method()
		if g.methodTrie[method] == nil {
			g.methodTrie[method] = &pathTrieNode{}
		}
		current := g.methodTrie[method]
		for _, p := range r.pathChain {
			if p == "" {
				continue
			}
			key := p
			paramKey := ""
			if strings.HasPrefix(p, "{") {
				key = "$"
				paramKey = p[1 : len(p)-1]
			}
			if current.children == nil {
				current.children = make(map[string]*pathTrieNode)
			}
			if current.children[key] == nil {
				current.children[key] = &pathTrieNode{
					paramKey: paramKey,
				}
			} else if current.children[key].paramKey != paramKey {
				panic(errx.Newf("path '%s' parameter conflict: original '%s' conflicts with new '%s'", r.path, current.children[key].paramKey, paramKey))
			}
			current = current.children[key]
		}
		if current.route != nil {
			panic(errx.Newf("path '%s' conflict definition", r.path))
		}
		current.route = r
	}
	sortRoutes(g.routes)
}

func sortRoutes(routes []Route) {
	methodOrder := map[Method]int{
		MethodGet:    0,
		MethodPost:   1,
		MethodPatch:  2,
		MethodPut:    3,
		MethodDelete: 4,
	}
	slices.SortFunc(routes, func(r Route, r2 Route) int {
		// 首先按path排序
		if r.Path() < r2.Path() {
			return -1
		}
		if r.Path() > r2.Path() {
			return 1
		}
		// path相同的情况下，按method排序

		method1 := r.Endpoint().Method()
		method2 := r2.Endpoint().Method()

		order1 := methodOrder[method1]
		order2 := methodOrder[method2]

		if order1 < order2 {
			return -1
		}
		if order1 > order2 {
			return 1
		}

		return 0
	})
}

func parseRoutes(node Node, routes *[]route, ancestors []Node, pathChain []string) {
	newPathChain := make([]string, len(pathChain))
	copy(newPathChain, pathChain)
	newAncestors := make([]Node, len(ancestors), len(ancestors)+1)
	copy(newAncestors, ancestors)
	newAncestors = append(newAncestors, node)
	if p := node.Path(); p != nil && *p != "" {
		chain := strings.Split(*p, "/")
		for _, s := range chain {
			if s == "" {
				continue
			}
			newPathChain = append(newPathChain, s)
		}
	}
	if a, ok := node.(Endpoint); ok {
		r := route{
			endpoint:  a,
			path:      "/" + strings.Join(newPathChain, "/"),
			pathChain: newPathChain,
			ancestors: ancestors,
		}
		for _, c := range newAncestors {
			if v := c.RequestContentType(); v != nil {
				r.requestContentType = *v
			}
			if v := c.ResponseContentType(); v != nil {
				r.responseContentType = *v
			}
			if v := c.RequireAuthorization(); v != nil {
				r.requireAuthorization = *v
			}
			if v := c.RequireAuthentication(); v != nil {
				r.requireAuthentication = *v
			}
			if v := c.RequireWrapOutput(); v != nil {
				r.requireWrapOutput = *v
			}
		}
		*routes = append(*routes, r)
		return
	}
	if g, ok := node.(Group); ok {
		for _, child := range g.Children() {
			parseRoutes(child, routes, newAncestors, newPathChain)
		}
	}
}

func (g *group) Match(method Method, path string) (MatchedRoute, bool) {
	if g.methodTrie == nil {
		return nil, false
	}
	if g.methodTrie[method] == nil {
		return nil, false
	}
	pathSegments := strings.Split(path, "/")
	currentNode := g.methodTrie[method]
	pathParams := make(map[string]string)
	for _, segment := range pathSegments {
		if segment == "" {
			continue
		}
		if exactMatch, exists := currentNode.children[segment]; exists {
			currentNode = exactMatch
			continue
		}
		if paramNode, exists := currentNode.children["$"]; exists {
			currentNode = paramNode
			pathParams[paramNode.paramKey] = segment
			continue
		}
		if wildcardNode, exists := currentNode.children["*"]; exists && wildcardNode.route != nil {
			return &matchedRoute{route: wildcardNode.route, pathParams: pathParams}, true
		}
		return nil, false
	}
	if currentNode.route != nil {
		return &matchedRoute{route: currentNode.route, pathParams: pathParams}, true
	}
	return nil, false
}

func (g *group) Routes() []Route {
	return g.routes
}

type pathTrieNode struct {
	paramKey string
	route    *route
	children map[string]*pathTrieNode
}

type route struct {
	endpoint              Endpoint
	path                  string
	pathChain             []string
	ancestors             []Node
	requestContentType    ContentType
	responseContentType   ContentType
	requireAuthentication bool
	requireAuthorization  bool
	requireWrapOutput     bool
}

func (r *route) Endpoint() Endpoint {
	return r.endpoint
}

func (r *route) RequestContentType() ContentType {
	return r.requestContentType
}

func (r *route) ResponseContentType() ContentType {
	return r.responseContentType
}

func (r *route) RequireAuthentication() bool {
	return r.requireAuthentication
}

func (r *route) RequireAuthorization() bool {
	return r.requireAuthorization
}

func (r *route) RequireWrapOutput() bool {
	return r.requireWrapOutput
}

func (r *route) Path() string {
	return r.path
}

func (r *route) PathChain() []string {
	return r.pathChain
}

func (r *route) Ancestors() []Node {
	return r.ancestors
}

type matchedRoute struct {
	*route
	pathParams map[string]string
}

func (m *matchedRoute) PathParams() map[string]string {
	return m.pathParams
}
