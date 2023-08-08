package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	ModifyComponent        = "common_modify"
	ModifyOutPort   string = "out"
	ModifyInPort    string = "in"
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
		Info:        "Sends a new message after any incoming message received",
		Tags:        []string{"SDK"},
	}
}

func (t *Modify) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if in, ok := msg.(ModifyInMessage); ok {
		return handler(ModifyOutPort, ModifyOutMessage{
			Context: in.Context,
		})
	}
	return fmt.Errorf("invalid message")
}

func (t *Modify) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:     ModifyInPort,
			Label:    "In",
			Source:   true,
			Message:  ModifyInMessage{},
			Position: module.Left,
		},
		{
			Name:     ModifyOutPort,
			Label:    "Out",
			Source:   false,
			Message:  ModifyOutMessage{},
			Position: module.Right,
		},
	}
}

var _ module.Component = (*Modify)(nil)

func init() {
	registry.Register(&Modify{})
}
