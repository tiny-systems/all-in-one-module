package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	ModifyComponent        = "common_modify"
	ModifyInPort    string = "in"
	ModifyOutPort   string = "out"
)

type ModifyContext any

type ModifyInMessage struct {
	Context ModifyContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Arbitrary message to be modified"`
}

type ModifyOutMessage struct {
	Context ModifyContext `json:"context"`
}

type Modify struct {
}

func (t *Modify) Instance() module.Component {
	return &Modify{}
}

func (t *Modify) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ModifyComponent,
		Description: "Modify",
		Info:        "Sends a new message after incoming message received",
		Tags:        []string{"SDK"},
	}
}

func (t *Modify) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if in, ok := msg.(ModifyInMessage); ok {
		return handler(ctx, ModifyOutPort, ModifyOutMessage{
			Context: in.Context,
		})
	}
	return fmt.Errorf("invalid message")
}

func (t *Modify) Ports() []module.Port {
	return []module.Port{
		{
			Name:          ModifyInPort,
			Label:         "In",
			Source:        true,
			Configuration: ModifyInMessage{},
			Position:      module.Left,
		},
		{
			Name:          ModifyOutPort,
			Label:         "Out",
			Source:        false,
			Configuration: ModifyOutMessage{},
			Position:      module.Right,
		},
	}
}

var _ module.Component = (*Modify)(nil)

func init() {
	registry.Register(&Modify{})
}
