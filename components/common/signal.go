package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	SignalComponent        = "signal"
	StartOutPort    string = "out"
)

type StartContext any

type StartSettings struct {
	Context StartContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to send" propertyOrder:"1"`
	Auto    bool         `json:"auto" title:"Auto send" description:"Send signal automatically" propertyOrder:"2"`
}

type Start struct {
	settings StartSettings
}

type StartControl struct {
	Send    bool         `json:"send" format:"button" title:"Send" required:"true" propertyOrder:"1"`
	Context StartContext `json:"context" propertyOrder:"2" title:"Context"`
}

func (t *Start) Instance() module.Component {
	return &Start{
		settings: StartSettings{},
	}
}

func (t *Start) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        SignalComponent,
		Description: "Signal",
		Info:        "Sends any message when flow starts",
		Tags:        []string{"SDK"},
	}
}

func (t *Start) Handle(ctx context.Context, handle module.Handler, port string, msg interface{}) error {

	switch port {
	case module.ControlPort:
		_ = handle(StartOutPort, msg)

	case module.SettingsPort:
		in, ok := msg.(StartSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
		if t.settings.Auto {
			return handle(StartOutPort, in.Context)
		}
	}
	return nil
}

func (t *Start) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: t.settings,
		},
		{
			Name:          StartOutPort,
			Label:         "Out",
			Source:        false,
			Position:      module.Right,
			Configuration: new(StartContext),
		},
		{
			Name:  module.ControlPort,
			Label: "Control",
			Configuration: StartControl{
				Context: t.settings.Context,
			},
		},
	}
}

var _ module.Component = (*Start)(nil)

func init() {
	registry.Register(&Start{})
}
