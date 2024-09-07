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

type DomainWhoisSuccess struct {
	WhoIs      whoisparser.WhoisInfo     `json:"whoIs"`
	DomainName string                    `json:"domainName" format:"hostname"`
	Context    DomainWhoisRequestContext `json:"context,omitempty"`
}

type DomainWhoisError struct {
	Error      string             `json:"error"`
	ErrorType  string             `json:"errorType" enum:"parseError,fetchError"`
	DomainName string             `json:"domainName" format:"hostname"`
	Request    DomainWhoisRequest `json:"request,omitempty"`
}

type Whois struct {
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

	if port != DomainWhoisRequestPort {
		return fmt.Errorf("port %s is not supported", port)
	}

	in, ok := msg.(DomainWhoisRequest)
	if !ok {
		return fmt.Errorf("invalid message")
	}
	resultRaw, err := whois.Whois(in.DomainName)
	if err != nil {
		return handler(ctx, DomainWhoisErrorPort, DomainWhoisError{
			ErrorType:  ErrFetch.Error(),
			Request:    in,
			DomainName: in.DomainName,
			Error:      err.Error(),
		})
	}
	result, err := whoisparser.Parse(resultRaw)
	if err != nil {
		return handler(ctx, DomainWhoisErrorPort, DomainWhoisError{
			ErrorType:  ErrParse.Error(),
			Request:    in,
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
}

func (t *Whois) Ports() []module.Port {
	return []module.Port{
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
			Name:          DomainWhoisErrorPort,
			Label:         "Error",
			Source:        false,
			Configuration: DomainWhoisError{},
			Position:      module.Right,
		},
	}
}

var _ module.Component = (*Whois)(nil)

func init() {
	registry.Register(&Whois{})
}
