package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	DebugComponent        = "debug"
	DebugInPort    string = "in"
)

type DebugContext any

type DebugSettings struct {
	Context DebugContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Debug message" propertyOrder:"1"`
}

type DebugIn struct {
	Context DebugContext `json:"context" configurable:"false" required:"true" title:"Context" propertyOrder:"1" title:"Context"`
}

type DebugControl struct {
	Context DebugContext `json:"context" readonly:"true" required:"true" propertyOrder:"1" title:"Context"`
}

type Debug struct {
	settings DebugSettings
}

func (t *Debug) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        DebugComponent,
		Description: "Debug",
		Info:        "Consumes any data without sending it anywhere.",
		Tags:        []string{"SDK"},
	}
}

func (t *Debug) Handle(ctx context.Context, output module.Handler, port string, msg interface{}) error {

	switch port {
	case module.SettingsPort:
		in, ok := msg.(DebugSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
		return nil
	case DebugInPort:
		if in, ok := msg.(DebugIn); ok {
			t.settings.Context = in.Context
			return output(ctx, module.ReconcilePort, nil)
		}
		return fmt.Errorf("invalid message in")
	}

	return fmt.Errorf("unknown port: %s", port)
}

func (t *Debug) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:          DebugInPort,
			Label:         "In",
			Source:        true,
			Configuration: DebugIn{},
			Position:      module.Left,
		},
		{
			Name:  module.ControlPort,
			Label: "Control",
			Configuration: DebugControl{
				Context: t.settings.Context,
			},
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: t.settings,
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
