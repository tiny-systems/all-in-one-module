package google

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"golang.org/x/oauth2/google"
)

const ExchangeAutCodeComponent = "google_exchange_auth_code"

type ExchangeAuthCodeInContext any

type ExchangeAuthCodeInMessage struct {
	Context  ExchangeAuthCodeInContext `json:"context" title:"Context" configurable:"true" propertyOrder:"1"`
	Config   ClientConfig              `json:"config" title:"Config"  required:"true" description:"Client Config" propertyOrder:"2"`
	AuthCode string                    `json:"authCode" required:"true" title:"Authorisation code" propertyOrder:"3"`
}

type ExchangeAuthCodeOutMessage struct {
	Context ExchangeAuthCodeInContext `json:"context" title:"Context" propertyOrder:"1"`
	Token   Token                     `json:"token" propertyOrder:"2"`
}

type ExchangeAuthCode struct {
}

func (a *ExchangeAuthCode) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ExchangeAutCodeComponent,
		Description: "Exchange Auth Code",
		Info:        "Exchanges Auth code to Auth token",
		Tags:        []string{"google", "auth"},
	}
}

func (a *ExchangeAuthCode) Handle(ctx context.Context, output module.Handler, port string, msg interface{}) error {
	if port == "in" {
		in, ok := msg.(ExchangeAuthCodeInMessage)
		if !ok {
			return fmt.Errorf("invalid input message")
		}
		config, err := google.ConfigFromJSON([]byte(in.Config.Credentials), in.Config.Scopes...)
		if err != nil {
			return fmt.Errorf("unable to parse client secret file to config: %v", err)
		}
		token, err := config.Exchange(ctx, in.AuthCode)
		if err != nil {
			return err
		}

		return output(ctx, "out", ExchangeAuthCodeOutMessage{
			Context: in.Context,
			Token: Token{
				AccessToken:  token.AccessToken,
				RefreshToken: token.RefreshToken,
				TokenType:    token.TokenType,
				Expiry:       token.Expiry,
			},
		})
	}
	return nil
}

func (a *ExchangeAuthCode) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Source:        true,
			Name:          "in",
			Label:         "In",
			Position:      module.Left,
			Configuration: ExchangeAuthCodeInMessage{},
		},
		{
			Source:        false,
			Name:          "out",
			Label:         "Out",
			Position:      module.Right,
			Configuration: ExchangeAuthCodeOutMessage{},
		},
	}
}

func (a *ExchangeAuthCode) Instance() module.Component {
	return &ExchangeAuthCode{}
}

var _ module.Component = (*ExchangeAuthCode)(nil)

func init() {
	registry.Register(&ExchangeAuthCode{})
}
