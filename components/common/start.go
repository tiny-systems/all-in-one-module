package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	StartComponent          = "start"
	StartOutPort     string = "out"
	StartControlPort string = "control"
)

type StartContext any

type StartSettings struct {
	Context StartContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be sent during first run or when system will be restarted"`
}

type Start struct {
	settings StartSettings
}

type StartControl struct {
	Send    bool         `json:"send" format:"button" title:"Send" required:"true" propertyOrder:"1"`
	Context StartContext `json:"context"`
}

func (t *Start) Instance() module.Component {
	return &Start{
		settings: StartSettings{},
	}
}

func (t *Start) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        StartComponent,
		Description: "Signal",
		Info:        "Sends any message when flow starts",
		Tags:        []string{"SDK"},
	}
}

func (t *Start) Handle(ctx context.Context, handle module.Handler, port string, msg interface{}) error {

	switch port {
	case StartControlPort:
		_ = handle(StartOutPort, msg)

	case module.SettingsPort:
		in, ok := msg.(StartSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
	}
	return nil
}

func (t *Start) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Settings:      true,
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
			Name:    StartControlPort,
			Label:   "Control",
			Control: true,
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
