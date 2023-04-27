package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/pkg/module"
	"github.com/tiny-systems/module/registry"
)

const (
	StarterComponent        = "start"
	StarterOutPort   string = "out"
)

type StartContext any

type StartSettings struct {
	Context StartContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be sent during first run or when system will be restarted"`
}

type Start struct {
	settings StartSettings
}

func (t *Start) Instance() module.Component {
	return &Start{
		settings: StartSettings{},
	}
}

func (t *Start) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        StarterComponent,
		Description: "Start",
		Info:        "Sends any message when flow starts",
		Tags:        []string{"SDK"},
	}
}

func (t *Start) Run(ctx context.Context, handle module.Handler) error {
	_ = handle(StarterOutPort, t.settings.Context)
	<-ctx.Done()
	return nil
}

func (t *Start) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
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
			Name:     module.SettingsPort,
			Label:    "Settings",
			Source:   true,
			Settings: true,
			Message:  StartSettings{},
		},
		{
			Name:     StarterOutPort,
			Label:    "Out",
			Source:   false,
			Position: module.Right,
			Message:  new(StartContext),
		},
	}
}

var _ module.Component = (*Start)(nil)

func init() {
	registry.Register(&Start{})
}
