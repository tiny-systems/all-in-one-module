package google

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	GetAuthUrlComponent    = "google_get_auth_url"
	GetAuthUrlRequestPort  = "request"
	GetAuthUrlResponsePort = "response"
	GetAuthUrlErrorPort    = "error"
)

type GetAuthUrlInContext any

type GetAuthUrlInMessage struct {
	Context       GetAuthUrlInContext `json:"context" title:"Context" configurable:"true" propertyOrder:"1"`
	Config        ClientConfig        `json:"config" required:"true" title:"Client credentials" propertyOrder:"2"`
	AccessType    string              `json:"accessType" title:"Type of access" enum:"offline,online" enumTitles:"Offline,Online" required:"true" propertyOrder:"3"`
	ApprovalForce bool                `json:"approvalForce" title:"ApprovalForce" required:"true" propertyOrder:"4"`
}

type GetAuthUrlSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

type GetAuthUrlErrorMessage struct {
	Request GetAuthUrlInMessage `json:"request"`
	Error   string              `json:"error"`
}

type GetAuthUrlOutMessage struct {
	Request GetAuthUrlInMessage `json:"request"`
	AuthUrl string              `json:"authUrl" format:"uri"`
}

type GetAuthUrl struct {
	settings GetAuthUrlSettings
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

	if port != GetAuthUrlRequestPort {
		return fmt.Errorf("unknown port %s", port)
	}
	//
	in, ok := msg.(GetAuthUrlInMessage)
	if !ok {
		return fmt.Errorf("invalid input message")
	}
	url, err := getAuthUrl(ctx, in)

	if err != nil {
		// check err port
		if a.settings.EnableErrorPort {
			return output(ctx, GetAuthUrlErrorPort, GetAuthUrlErrorMessage{
				Request: in,
				Error:   err.Error(),
			})
		}
		return err
	}

	return output(ctx, GetAuthUrlResponsePort, GetAuthUrlOutMessage{
		Request: in,
		AuthUrl: url,
	})
}

func getAuthUrl(_ context.Context, in GetAuthUrlInMessage) (string, error) {

	config, err := google.ConfigFromJSON([]byte(in.Config.Credentials), in.Config.Scopes...)
	if err != nil {
		return "", fmt.Errorf("unable to parse client secret file to config: %v", err)
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
	return config.AuthCodeURL("state-token", opts...), nil
}

func (a *GetAuthUrl) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Source:   true,
			Name:     GetAuthUrlRequestPort,
			Label:    "Request",
			Position: module.Left,
			Configuration: GetAuthUrlInMessage{
				AccessType:    "offline",
				ApprovalForce: true,
			},
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: GetAuthUrlSettings{},
			Source:        true,
		},
		{
			Source:        false,
			Name:          GetAuthUrlResponsePort,
			Label:         "Response",
			Position:      module.Right,
			Configuration: GetAuthUrlOutMessage{},
		},
	}

	if !a.settings.EnableErrorPort {
		return ports
	}

	return append(ports, module.NodePort{
		Position:      module.Bottom,
		Name:          GetAuthUrlErrorPort,
		Label:         "Error",
		Source:        false,
		Configuration: GetAuthUrlErrorMessage{},
	})
}

func (a *GetAuthUrl) Instance() module.Component {
	return &GetAuthUrl{}
}

var _ module.Component = (*GetAuthUrl)(nil)

func init() {
	registry.Register(&GetAuthUrl{})
}
