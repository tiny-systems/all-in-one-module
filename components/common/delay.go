package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"time"
)

const (
	DelayComponent        = "delay"
	DelayOutPort   string = "out"
	DelayInPort    string = "in"
)

type DelayContext any

type DelayInMessage struct {
	Context DelayContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be delayed" propertyOrder:"1"`
	Delay   int          `json:"delay" required:"true" title:"Delay (ms)" propertyOrder:"2"`
}

type DelayOutMessage struct {
	Delay   int          `json:"delay"`
	Context DelayContext `json:"context"`
}

type Delay struct {
}

func (t *Delay) Instance() module.Component {
	return &Delay{}
}

func (t *Delay) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        DelayComponent,
		Description: "Delay",
		Info:        "Sleeps before passing incoming messages further",
		Tags:        []string{"SDK"},
	}
}

func (t *Delay) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	in, ok := msg.(DelayInMessage)
	if !ok {
		return fmt.Errorf("invalid message")
	}
	if in.Delay <= 0 {
		return fmt.Errorf("invalid delay")
	}

	time.Sleep(time.Millisecond * time.Duration(in.Delay))
	_ = handler(DelayOutPort, DelayOutMessage{
		Context: in.Context,
		Delay:   in.Delay,
	})
	return nil
}

func (t *Delay) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:   DelayInPort,
			Label:  "In",
			Source: true,
			Configuration: DelayInMessage{
				Delay: 1000,
			},
			Position: module.Left,
		},
		{
			Name:          DelayOutPort,
			Label:         "Out",
			Source:        false,
			Configuration: DelayOutMessage{},
			Position:      module.Right,
		},
	}
}

var _ module.Component = (*Delay)(nil)

func init() {
	registry.Register(&Delay{})
}
