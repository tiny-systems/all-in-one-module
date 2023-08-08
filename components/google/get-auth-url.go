package google

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const GetAuthUrlComponent = "google_get_auth_url"

type GetAuthUrlInContext any

type GetAuthUrlInMessage struct {
	Context       GetAuthUrlInContext `json:"context" title:"Context" configurable:"true" propertyOrder:"1"`
	Config        ClientConfig        `json:"config" required:"true" title:"Client credentials" propertyOrder:"2"`
	AccessType    string              `json:"accessType" title:"Type of access" enum:"offline,online" enumTitles:"Offline,Online" required:"true" propertyOrder:"3"`
	ApprovalForce bool                `json:"approvalForce" title:"ApprovalForce" required:"true" propertyOrder:"4"`
}

type GetAuthUrlOutMessage struct {
	Context GetAuthUrlInContext `json:"context""`
	AuthUrl string              `json:"authUrl" format:"uri"`
}

type GetAuthUrl struct {
}

func (a *GetAuthUrl) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        GetAuthUrlComponent,
		Description: "Get Auth URL",
		Info:        "Gets Auth URL which later may be used fot auth redirect",
		Tags:        []string{"google", "auth"},
	}
}

func (a *GetAuthUrl) Handle(ctx context.Context, output module.Handler, port string, msg interface{}) error {
	if port == "in" {
		in, ok := msg.(GetAuthUrlInMessage)
		if !ok {
			return fmt.Errorf("invalid input message")
		}
		config, err := google.ConfigFromJSON([]byte(in.Config.Credentials), in.Config.Scopes...)
		if err != nil {
			return fmt.Errorf("unable to parse client secret file to config: %v", err)
		}
		var opts []oauth2.AuthCodeOption
		if in.ApprovalForce {
			opts = append(opts, oauth2.ApprovalForce)
		}
		if in.AccessType == "online" {
			opts = append(opts, oauth2.AccessTypeOnline)
		} else {
			opts = append(opts, oauth2.AccessTypeOffline)
		}
		return output("out", GetAuthUrlOutMessage{
			Context: in.Context,
			AuthUrl: config.AuthCodeURL("state-token", opts...),
		})
	}
	return nil
}

func (a *GetAuthUrl) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Source:   true,
			Name:     "in",
			Label:    "In",
			Position: module.Left,
			Message: GetAuthUrlInMessage{
				AccessType:    "offline",
				ApprovalForce: true,
			},
		},
		{
			Source:   false,
			Name:     "out",
			Label:    "Out",
			Position: module.Right,
			Message:  GetAuthUrlOutMessage{},
		},
	}
}

func (a *GetAuthUrl) Instance() module.Component {
	return &GetAuthUrl{}
}

var _ module.Component = (*GetAuthUrl)(nil)

func init() {
	registry.Register(&GetAuthUrl{})
}
