package common

import (
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/swaggest/jsonschema-go"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"strings"
)

const (
	RouterComponent   = "router"
	RouterInPort      = "input"
	RouterDefaultPort = "default"
)

// RouteName special type which can carry its value and possible options for enum values
type RouteName struct {
	Value   string
	Options []string
}

// MarshalJSON treat like underlying Value string
func (r *RouteName) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Value)
}

// UnmarshalJSON treat like underlying Value string
func (r *RouteName) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Value)
}

func (r RouteName) JSONSchema() (jsonschema.Schema, error) {
	name := jsonschema.Schema{}
	name.AddType(jsonschema.String)
	name.WithTitle("Route name")
	name.WithDefault(r.Value)
	enums := make([]interface{}, len(r.Options))
	for k, v := range r.Options {
		enums[k] = v
	}
	name.WithEnum(enums...)
	return name, nil
}

type Condition struct {
	RouteName RouteName `json:"route" title:"Route" required:"true" propertyOrder:"1"`
	Condition bool      `json:"condition,omitempty" required:"true" title:"Condition" propertyOrder:"2"`
}

type RouterSettings struct {
	Routes []string `json:"routes,omitempty" required:"true" title:"Routes" minItems:"1" uniqueItems:"true"`
}

type RouterContext any

type RouterOutMessage struct {
	Route   string        `json:"route" required:"true" title:"Selected route" default:"A"`
	Context RouterContext `json:"context"`
}

type RouterInMessage struct {
	Context    RouterContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Arbitrary message to be routed" propertyOrder:"1"`
	Conditions []Condition   `json:"conditions,omitempty" required:"true" title:"Conditions" minItems:"1" uniqueItems:"true" propertyOrder:"2"`
}

type Router struct {
	settings RouterSettings
}

var defaultRouterSettings = RouterSettings{
	Routes: []string{"A", "B"},
}

func (t *Router) Instance() module.Component {
	return &Router{
		settings: defaultRouterSettings,
	}
}

func (t *Router) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        RouterComponent,
		Description: "Router",
		Info:        "Routes incoming messages depends on message itself",
		Tags:        []string{"SDK"},
	}
}

func (t *Router) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(RouterSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
		return nil
	}

	if in, ok := msg.(RouterInMessage); ok {
		for _, condition := range in.Conditions {
			if condition.Condition {
				return handler(getPortNameFromRoute(condition.RouteName.Value), RouterOutMessage{
					Context: in.Context,
					Route:   condition.RouteName.Value,
				})
			}
		}
		return handler(RouterDefaultPort, RouterOutMessage{
			Context: in.Context,
			Route:   RouterDefaultPort,
		})
	}
	return fmt.Errorf("invalid message")
}

// Ports drop settings, make it port payload
func (t *Router) Ports() []module.NodePort {

	inMessage := RouterInMessage{
		Conditions: []Condition{{
			RouteName: RouteName{Value: "A", Options: t.settings.Routes},
			Condition: true,
		}},
	}

	ports := []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Settings:      true,
			Configuration: t.settings,
		},
		{
			Position:      module.Left,
			Name:          RouterInPort,
			Label:         "IN",
			Source:        true,
			Configuration: inMessage,
		},
		{
			Position: module.Bottom,
			Name:     RouterDefaultPort,
			Label:    "DEFAULT",
			Source:   false,
			Configuration: RouterOutMessage{
				Context: inMessage.Context,
				Route:   RouterDefaultPort,
			},
		},
	}
	for _, r := range t.settings.Routes {
		ports = append(ports, module.NodePort{
			Position:      module.Right,
			Name:          getPortNameFromRoute(r),
			Label:         strings.ToTitle(r),
			Source:        false,
			Configuration: RouterOutMessage{},
		})
	}
	return ports
}

func getPortNameFromRoute(route string) string {
	return fmt.Sprintf("out_%s", strings.ToLower(route))
}

var _ module.Component = (*Router)(nil)
var _ jsonschema.Exposer = (*RouteName)(nil)

func init() {
	registry.Register(&Router{})
}
