package api

import (
	"regexp"
	"sort"
	"strings"

	"github.com/tencent-go/pkg/errx"
)

type Permission string

const PermissionSeparator = "."

func (p Permission) Match(permissions ...Permission) bool {
	if p == "" {
		return true
	}
	for _, permission := range permissions {
		if permission == p || permission == "*" {
			return true
		}
		if len(permission) < len(p) && strings.HasPrefix(string(p), string(permission)) && p[len(permission)] == PermissionSeparator[0] {
			return true
		}
	}
	return false
}

type PermissionProvider interface {
	GetResources() []Resource
	GetEndpointPermission(api Endpoint) (*Permission, bool)
}

type Resource struct {
	Permission  Permission `json:"permission"`
	Description string     `json:"description,omitempty"`
	Children    []Resource `json:"children,omitempty"`
}

func NewPermissionProvider(routes ...Route) (PermissionProvider, errx.Error) {
	return newProvider(routes...)
}

type permProvider struct {
	permissions map[Endpoint]Permission
	resources   []Resource
}

func (c *permProvider) GetResources() []Resource {
	return c.resources
}

func (c *permProvider) GetEndpointPermission(api Endpoint) (*Permission, bool) {
	permission, exists := c.permissions[api]
	if !exists {
		return nil, false
	}
	return &permission, true
}

type permNode struct {
	description string
	permission  Permission
	children    map[string]*permNode
}

func newProvider(routes ...Route) (*permProvider, errx.Error) {
	//apiAncestors := map[api.Endpoint][]api.Node{}
	//parseApiAncestors(rootNode, apiAncestors)
	rootTrie := &permNode{}
	permissions := map[Endpoint]Permission{}

	for _, route := range routes {
		if !route.RequireAuthorization() {
			continue
		}
		var name string
		if n := route.Endpoint().Name(); n != nil {
			name = camelToSnake(*n)
		}
		if name == "" {
			continue
		}
		currentTrie := rootTrie
		var namePath []string

		for _, node := range route.Ancestors() {
			var nodeName string
			if n := node.Name(); n != nil {
				nodeName = camelToSnake(*n)
			}
			if nodeName == "" {
				continue
			}
			namePath = append(namePath, nodeName)
			var nodeDescription string
			if description := node.Description(); description != nil {
				nodeDescription = *description
			}
			if currentTrie.children == nil {
				currentTrie.children = map[string]*permNode{}
			}
			children := currentTrie.children
			if children[nodeName] == nil {
				children[nodeName] = &permNode{
					description: nodeDescription,
					permission:  Permission(strings.Join(namePath, PermissionSeparator)),
				}
			} else {
				if nodeDescription != "" {
					children[nodeName].description = nodeDescription
				}
			}
			currentTrie = children[nodeName]
		}
		permissions[route.Endpoint()] = currentTrie.permission
	}

	return &permProvider{
		permissions: permissions,
		resources:   convertTrieToResources(rootTrie.children),
	}, nil
}

func convertTrieToResources(permNodes map[string]*permNode) []Resource {
	var resources []Resource
	var nodeNames []string

	for nodeName := range permNodes {
		nodeNames = append(nodeNames, nodeName)
	}
	sort.Strings(nodeNames)

	for _, nodeName := range nodeNames {
		node := permNodes[nodeName]
		resource := Resource{
			Permission:  node.permission,
			Description: node.description,
		}
		if len(node.children) > 0 {
			resource.Children = convertTrieToResources(node.children)
		}
		resources = append(resources, resource)
	}
	return resources
}

var camelToSnakeRegex = regexp.MustCompile("([a-z0-9])([A-Z])")

func camelToSnake(s string) string {
	snake := camelToSnakeRegex.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}
