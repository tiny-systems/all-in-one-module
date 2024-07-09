package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	MixerPortA      string = "a"
	MixerPortB      string = "b"
	MixerOutputPort string = "output"
)

type Mixer struct {
	a MixerInputAContext
	b MixerInputBContext
}

type MixerInputAContext any
type MixerInputBContext any

type MixerInputA struct {
	Context MixerInputAContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Arbitrary message"`
}

type MixerInputB struct {
	Context MixerInputBContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Arbitrary message"`
}

type MixerOutput struct {
	ContextA MixerInputAContext `json:"contextA"`
	ContextB MixerInputBContext `json:"contextB"`
}

func (m *Mixer) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        "mixer",
		Description: "Mixer",
		Info:        "Mixes latest values on ports into single message",
		Tags:        []string{"SDK"},
	}
}

func (m *Mixer) Handle(ctx context.Context, output module.Handler, port string, message interface{}) error {

	switch port {
	case MixerPortB:
		m.b = message.(MixerInputBContext)
		return m.send(ctx, output)
	case MixerPortA:
		m.a = message.(MixerInputAContext)
		return m.send(ctx, output)
	default:
		return fmt.Errorf("unknown port: %s", port)
	}
}

func (m *Mixer) send(ctx context.Context, output module.Handler) error {
	return output(ctx, MixerOutputPort, MixerOutput{
		ContextA: m.a,
		ContextB: m.b,
	})
}

func (m *Mixer) Ports() []module.Port {
	return []module.Port{
		{
			Name:          MixerPortA,
			Label:         "A",
			Source:        true,
			Configuration: MixerInputA{},
			Position:      module.Left,
		},
		{
			Name:          MixerPortB,
			Label:         "B",
			Source:        true,
			Configuration: MixerInputB{},
			Position:      module.Left,
		},
		{
			Name:          MixerOutputPort,
			Label:         "Output",
			Configuration: MixerOutput{},
			Position:      module.Right,
		},
	}
}

func (m *Mixer) Instance() module.Component {
	return &Mixer{}
}

var _ module.Component = (*Mixer)(nil)

func init() {
	registry.Register(&Mixer{})
}
