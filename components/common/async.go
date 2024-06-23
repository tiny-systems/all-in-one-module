package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"go.opentelemetry.io/otel/trace"
)

const (
	AsyncComponent        = "common_async"
	AsyncInPort    string = "in"
	AsyncOutPort   string = "out"
)

type AsyncContext any

type AsyncInMessage struct {
	Context AsyncContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Arbitrary message to be modified"`
}

type AsyncOutMessage struct {
	Context AsyncContext `json:"context"`
}

type Async struct {
}

func (t *Async) Instance() module.Component {
	return &Async{}
}

func (t *Async) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        AsyncComponent,
		Description: "Async",
		Info:        "Asynchronously Sends a new message after incoming message received",
		Tags:        []string{"SDK"},
	}
}

func (t *Async) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if in, ok := msg.(AsyncInMessage); ok {
		go func() {
			_ = handler(trace.ContextWithSpanContext(context.Background(), trace.SpanContextFromContext(ctx)), AsyncOutPort, AsyncOutMessage{
				Context: in.Context,
			})
		}()
		return nil
	}
	return fmt.Errorf("invalid message")
}

func (t *Async) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:          AsyncInPort,
			Label:         "In",
			Source:        true,
			Configuration: AsyncInMessage{},
			Position:      module.Left,
		},
		{
			Name:          AsyncOutPort,
			Label:         "Out",
			Source:        false,
			Configuration: AsyncOutMessage{},
			Position:      module.Right,
		},
	}
}

var _ module.Component = (*Async)(nil)

func init() {
	registry.Register(&Async{})
}
