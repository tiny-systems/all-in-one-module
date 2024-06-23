package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	SignalComponent        = "signal"
	SignalOutPort   string = "out"
)

type SignalContext any

type SignalSettings struct {
	Context SignalContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to send" propertyOrder:"1"`
	Auto    bool          `json:"auto" title:"Auto send" required:"true" description:"Send signal automatically" propertyOrder:"2"`
}

type Signal struct {
	settings SignalSettings
}

type SignalControl struct {
	Send    bool          `json:"send" format:"button" title:"Send" required:"true" propertyOrder:"1"`
	Context SignalContext `json:"context" propertyOrder:"2" title:"Context"`
}

func (t *Signal) Instance() module.Component {
	return &Signal{
		settings: SignalSettings{},
	}
}

func (t *Signal) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        SignalComponent,
		Description: "Signal",
		Info:        "Sends any message when flow starts",
		Tags:        []string{"SDK"},
	}
}

func (t *Signal) Handle(ctx context.Context, handle module.Handler, port string, msg interface{}) error {

	switch port {
	case module.ControlPort:
		in, ok := msg.(SignalControl)
		if !ok {
			return fmt.Errorf("invalid input msg")
		}
		_ = handle(ctx, SignalOutPort, in.Context)

	case module.SettingsPort:
		in, ok := msg.(SignalSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
		if t.settings.Auto {
			return handle(ctx, SignalOutPort, in.Context)
		}
	}
	return nil
}

func (t *Signal) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: t.settings,
		},
		{
			Name:          SignalOutPort,
			Label:         "Out",
			Source:        false,
			Position:      module.Right,
			Configuration: new(SignalContext),
		},
		{
			Name:  module.ControlPort,
			Label: "Control",
			Configuration: SignalControl{
				Context: t.settings.Context,
			},
		},
	}
}

var _ module.Component = (*Signal)(nil)

func init() {
	registry.Register(&Signal{})
}
