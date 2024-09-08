package network

import (
	"context"
	"fmt"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/pkg/errors"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	DomainWhoisComponent           = "domain_whois"
	DomainWhoisResponsePort string = "response"
	DomainWhoisErrorPort    string = "error"
	DomainWhoisRequestPort  string = "request"
)

var (
	ErrFetch = errors.New("fetchError")
	ErrParse = errors.New("parseError")
)

type DomainWhoisRequestContext any

type DomainWhoisRequest struct {
	Context    DomainWhoisRequestContext `json:"context,omitempty" configurable:"true" title:"Context" description:"Arbitrary message to be send further"`
	DomainName string                    `json:"domainName" required:"true" title:"Domain name to check" format:"hostname"`
}

type WhoisSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

type DomainWhoisSuccess struct {
	Context    DomainWhoisRequestContext `json:"context,omitempty"`
	WhoIs      whoisparser.WhoisInfo     `json:"whoIs"`
	DomainName string                    `json:"domainName" format:"hostname"`
}

type DomainWhoisError struct {
	Context    DomainWhoisRequestContext `json:"context,omitempty"`
	Error      string                    `json:"error"`
	ErrorType  string                    `json:"errorType" enum:"parseError,fetchError"`
	DomainName string                    `json:"domainName" format:"hostname"`
}

type Whois struct {
	settings WhoisSettings
}

func (t *Whois) Instance() module.Component {
	return &Whois{}
}

func (t *Whois) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        DomainWhoisComponent,
		Description: "Whois checker",
		Info:        "Fetches and returns a fully-parsed whois record",
		Tags:        []string{"Network"},
	}
}

func (t *Whois) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	switch port {
	case module.SettingsPort:
		// compile template
		in, ok := msg.(WhoisSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
		return nil

	case DomainWhoisRequestPort:

		in, ok := msg.(DomainWhoisRequest)
		if !ok {
			return fmt.Errorf("invalid message")
		}

		resultRaw, err := whois.Whois(in.DomainName)
		if err != nil {
			if !t.settings.EnableErrorPort {
				return err
			}

			return handler(ctx, DomainWhoisErrorPort, DomainWhoisError{
				ErrorType:  ErrFetch.Error(),
				Context:    in.Context,
				DomainName: in.DomainName,
				Error:      err.Error(),
			})
		}
		result, err := whoisparser.Parse(resultRaw)
		if err != nil {

			if !t.settings.EnableErrorPort {
				return err
			}

			return handler(ctx, DomainWhoisErrorPort, DomainWhoisError{
				ErrorType:  ErrParse.Error(),
				Context:    in.Context,
				DomainName: in.DomainName,
				Error:      err.Error(),
			})
		}

		resp := DomainWhoisSuccess{
			WhoIs:      result,
			DomainName: in.DomainName,
			Context:    in.Context,
		}
		return handler(ctx, DomainWhoisResponsePort, resp)

	default:
		return fmt.Errorf("port %s is not supoprted", port)
	}
}

func (t *Whois) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:   DomainWhoisRequestPort,
			Label:  "Request",
			Source: true,
			Configuration: DomainWhoisRequest{
				DomainName: "example.com",
			},
			Position: module.Left,
		},
		{
			Name:          DomainWhoisResponsePort,
			Label:         "Response",
			Source:        false,
			Configuration: DomainWhoisSuccess{},
			Position:      module.Right,
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: t.settings,
			Source:        true,
		},
	}
	if t.settings.EnableErrorPort {
		ports = append(ports, module.Port{
			Name:          DomainWhoisErrorPort,
			Label:         "Error",
			Source:        false,
			Configuration: DomainWhoisError{},
			Position:      module.Bottom,
		})
	}
	return ports
}

var _ module.Component = (*Whois)(nil)

func init() {
	registry.Register(&Whois{})
}
