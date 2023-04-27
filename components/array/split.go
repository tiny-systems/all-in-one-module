package array

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tiny-systems/module/pkg/module"
	"github.com/tiny-systems/module/registry"
)

const (
	SplitComponent        = "split"
	SplitOutPort   string = "out"
	SplitInPort    string = "in"
)

type SplitContext any
type SplitItemContext any

type SplitInMessage struct {
	Context SplitContext       `json:"context" title:"Context" configurable:"true"  description:"Message to be send further with each item"  configurable:"true" propertyOrder:"1"`
	Array   []SplitItemContext `json:"array,omitempty" title:"Array" default:"null" description:"Array of items to be split" required:"true" propertyOrder:"2"`
}

type SplitOutMessage struct {
	Context SplitContext     `json:"context"`
	Item    SplitItemContext `json:"item"`
}

type Split struct {
}

func (t *Split) Instance() module.Component {
	return &Split{}
}

func (t *Split) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        SplitComponent,
		Description: "Array split",
		Info:        "Splits any array into chunks and send further as separate messages",
		Tags:        []string{"SDK", "ARRAY"},
	}
}

func (t *Split) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if in, ok := msg.(SplitInMessage); ok {
		for _, item := range in.Array {
			if err := handler(SplitOutPort, SplitOutMessage{
				Context: in.Context,
				Item:    item,
			}); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	return fmt.Errorf("invalid message")
}

func (t *Split) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:     SplitInPort,
			Label:    "In",
			Source:   true,
			Message:  SplitInMessage{},
			Position: module.Left,
		},
		{
			Name:     SplitOutPort,
			Label:    "Out",
			Source:   false,
			Message:  SplitOutMessage{},
			Position: module.Right,
		},
	}
}

func init() {
	registry.Register(&Split{})
}
