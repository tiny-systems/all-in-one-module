package common

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"sync/atomic"
	"time"
)

const (
	TickerComponent         = "ticker"
	TickerOutPort    string = "out"
	TickerStatusPort string = "status"
)

type TickerContext any

type TickerStatus struct {
	Status string `json:"status" readonly:"true" title:"Status" colSpan:"col-span-6" propertyOrder:"1"`
	Reset  bool   `json:"reset" format:"button" title:"Reset" required:"true" colSpan:"col-span-6" propertyOrder:"2"`
}

type TickerSettings struct {
	Period            int           `json:"period" required:"true" title:"Periodicity (ms)" minimum:"10" default:"1000" propertyOrder:"1"`
	EnableControlPort bool          `json:"enableControlPort" required:"true" title:"Enable control port" description:"Control port allows control ticker externally" propertyOrder:"3"`
	Context           TickerContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be send each period of time"`
}

type Ticker struct {
	counter  int64
	settings TickerSettings
}

func (t *Ticker) Instance() module.Component {
	return &Ticker{
		settings: TickerSettings{
			Period: 1000,
		},
	}
}

type TickerControl struct {
	Start bool `json:"start" required:"true" title:"Ticker state"`
}

func (t *Ticker) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        TickerComponent,
		Description: "Ticker",
		Info:        "Sends messages periodically",
		Tags:        []string{"SDK"},
	}
}

// Emit non a pointer receiver copies Ticker with copy of settings
func (t *Ticker) emit(ctx context.Context, handler module.Handler) error {
	ticker := time.NewTicker(time.Duration(t.settings.Period) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:

			atomic.AddInt64(&t.counter, 1)
			_ = handler(ctx, TickerOutPort, t.settings.Context)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Ticker) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		settings, ok := msg.(TickerSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		if settings.Period < 10 {
			return fmt.Errorf("period should be more than 10 milliseconds")
		}
		t.settings = settings
		return nil
	}

	return fmt.Errorf("invalid message")
}

func (t *Ticker) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:   TickerStatusPort,
			Label:  "Status",
			Source: true,
			Configuration: TickerStatus{
				Status: fmt.Sprintf("All good: %d", t.counter),
			},
		},
		{
			Name:   module.SettingsPort,
			Label:  "Settings",
			Source: true,

			Configuration: TickerSettings{
				Period: 1000,
			},
		},
		{
			Name:          TickerOutPort,
			Label:         "Out",
			Source:        false,
			Position:      module.Right,
			Configuration: new(TickerContext),
		},
	}
	if t.settings.EnableControlPort {
		ports = append(ports, module.NodePort{
			Position:      module.Left,
			Name:          module.ControlPort,
			Label:         "Control",
			Source:        true,
			Configuration: TickerControl{},
		})
	}
	return ports
}

var _ module.Component = (*Ticker)(nil)

func init() {
	registry.Register(&Ticker{})
}
