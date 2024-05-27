package common

import (
	"context"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	DebugComponent        = "debug"
	DebugInPort    string = "in"
)

type DebugContext any

type DebugContextIn struct {
	Context DebugContext `json:"context" title:"Context"`
}

type Debug struct {
}

func (t *Debug) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        DebugComponent,
		Description: "Debug",
		Info:        "Consumes any data without sending it anywhere.",
		Tags:        []string{"SDK"},
	}
}

func (t *Debug) Handle(ctx context.Context, output module.Handler, port string, message interface{}) error {
	return nil
}

func (t *Debug) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:          DebugInPort,
			Label:         "In",
			Source:        true,
			Configuration: DebugContextIn{},
			Position:      module.Left,
		},
	}
}

func (t *Debug) Instance() module.Component {
	return &Debug{}
}

var _ module.Component = (*Debug)(nil)

func init() {
	registry.Register(&Debug{})
}
