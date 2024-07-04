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
	ExchangeAutCodeComponent     = "google_exchange_auth_code"
	ExchangeAuthCodeRequestPort  = "request"
	ExchangeAuthCodeResponsePort = "response"
	ExchangeAuthCodeErrorPort    = "error"
)

type ExchangeAuthCodeInContext any

type ExchangeAuthCodeInMessage struct {
	Context  ExchangeAuthCodeInContext `json:"context" title:"Context" configurable:"true" propertyOrder:"1"`
	Config   ClientConfig              `json:"config" title:"Config"  required:"true" description:"Client Config" propertyOrder:"2"`
	AuthCode string                    `json:"authCode" required:"true" title:"Authorisation code" propertyOrder:"3"`
}

type ExchangeAuthCodeSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

type ExchangeAuthCodeOutMessage struct {
	Context ExchangeAuthCodeInContext `json:"context" title:"Context" propertyOrder:"1"`
	Token   Token                     `json:"token" propertyOrder:"2"`
}

type ExchangeAuthCodeError struct {
	Request ExchangeAuthCodeInMessage `json:"request"`
	Error   string                    `json:"error"`
}

///

type ExchangeAuthCode struct {
	settings ExchangeAuthCodeSettings
}

func (a *ExchangeAuthCode) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ExchangeAutCodeComponent,
		Description: "Exchange Auth Code",
		Info:        "Exchanges Auth code to Auth token",
		Tags:        []string{"google", "auth"},
	}
}

func (a *ExchangeAuthCode) exchange(ctx context.Context, in ExchangeAuthCodeInMessage) (*oauth2.Token, error) {

	config, err := google.ConfigFromJSON([]byte(in.Config.Credentials), in.Config.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	return config.Exchange(ctx, in.AuthCode)
}

func (a *ExchangeAuthCode) Handle(ctx context.Context, output module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(ExchangeAuthCodeSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		a.settings = in
		return nil
	}

	if port != ExchangeAuthCodeRequestPort {
		return fmt.Errorf("unknown port %s", port)
	}

	in, ok := msg.(ExchangeAuthCodeInMessage)
	if !ok {
		return fmt.Errorf("invalid input message")
	}

	token, err := a.exchange(ctx, in)
	if err != nil {
		// check err port
		if !a.settings.EnableErrorPort {
			return err
		}
		return output(ctx, ExchangeAuthCodeErrorPort, ExchangeAuthCodeError{
			Request: in,
			Error:   err.Error(),
		})
	}

	return output(ctx, ExchangeAuthCodeResponsePort, ExchangeAuthCodeOutMessage{
		Context: in.Context,
		Token: Token{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    token.TokenType,
			Expiry:       token.Expiry,
		},
	})

}

func (a *ExchangeAuthCode) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: ExchangeAuthCodeSettings{},
			Source:        true,
		},
		{
			Source:        true,
			Name:          ExchangeAuthCodeRequestPort,
			Label:         "Request",
			Position:      module.Left,
			Configuration: ExchangeAuthCodeInMessage{},
		},
		{
			Source:        false,
			Name:          ExchangeAuthCodeResponsePort,
			Label:         "Response",
			Position:      module.Right,
			Configuration: ExchangeAuthCodeOutMessage{},
		},
	}

	if !a.settings.EnableErrorPort {
		return ports
	}

	return append(ports, module.Port{
		Position:      module.Bottom,
		Name:          "error",
		Label:         "Error",
		Source:        false,
		Configuration: ExchangeAuthCodeError{},
	})
}

func (a *ExchangeAuthCode) Instance() module.Component {
	return &ExchangeAuthCode{}
}

var _ module.Component = (*ExchangeAuthCode)(nil)

func init() {
	registry.Register(&ExchangeAuthCode{})
}
