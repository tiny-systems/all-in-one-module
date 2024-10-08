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
	Context SignalContext `json:"context" required:"true" configurable:"true" title:"Context" description:"Arbitrary message to send"`
	Auto    bool          `json:"auto" title:"Auto send" required:"true" description:"Send signal automatically"`
}

type Signal struct {
	settings SignalSettings
}

type SignalControl struct {
	Context SignalContext `json:"context" required:"true" title:"Context"`
	Send    bool          `json:"send" format:"button" title:"Send" required:"true"`
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

func (t *Signal) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	switch port {
	case module.ControlPort:
		in, ok := msg.(SignalControl)
		if !ok {
			return fmt.Errorf("invalid input msg")
		}

		t.settings.Context = in.Context
		_ = handler(ctx, module.ReconcilePort, nil)
		_ = handler(ctx, SignalOutPort, in.Context)

	case module.SettingsPort:
		in, ok := msg.(SignalSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in

		if t.settings.Auto {
			return handler(ctx, SignalOutPort, in.Context)
		}
	}
	return nil
}

func (t *Signal) Ports() []module.Port {

	return []module.Port{
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
