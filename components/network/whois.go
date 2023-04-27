package network

import (
	"context"
	"fmt"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/pkg/errors"
	"github.com/tiny-systems/module/pkg/module"
	"github.com/tiny-systems/module/registry"
)

const (
	DomainWhoisComponent          = "domain_whois"
	DomainWhoisSuccessPort string = "success"
	DomainWhoisErrorPort   string = "error"
	DomainWhoisInPort      string = "in"
)

var (
	ErrFetch = errors.New("fetchError")
	ErrParse = errors.New("parseError")
)

type DomainWhoisRequestContext any

type DomainWhoisRequest struct {
	Context    DomainWhoisRequestContext `json:"context,omitempty" configurable:"true" title:"Context" description:"Arbitrary message to be send further" propertyOrder:"1"`
	DomainName string                    `json:"domainName" required:"true" title:"Domain name to check" format:"hostname" propertyOrder:"2"`
}

type DomainWhoisSuccess struct {
	WhoIs      whoisparser.WhoisInfo     `json:"whoIs"`
	DomainName string                    `json:"domainName" format:"hostname"`
	Context    DomainWhoisRequestContext `json:"context,omitempty"`
}

type DomainWhoisError struct {
	Error      string                    `json:"error"`
	ErrorType  string                    `json:"errorType" enum:"parseError,fetchError"`
	DomainName string                    `json:"domainName" format:"hostname"`
	Context    DomainWhoisRequestContext `json:"context,omitempty"`
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
	in, ok := msg.(DomainWhoisRequest)
	if !ok {
		return fmt.Errorf("invalid message")
	}
	resultRaw, err := whois.Whois(in.DomainName)
	if err != nil {
		return handler(DomainWhoisErrorPort, DomainWhoisError{
			ErrorType:  ErrFetch.Error(),
			Context:    in.Context,
			DomainName: in.DomainName,
			Error:      err.Error(),
		})
	}
	result, err := whoisparser.Parse(resultRaw)
	if err != nil {
		return handler(DomainWhoisErrorPort, DomainWhoisError{
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
	return handler(DomainWhoisSuccessPort, resp)
}

func (t *Whois) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:   DomainWhoisInPort,
			Label:  "In",
			Source: true,
			Message: DomainWhoisRequest{
				DomainName: "example.com",
			},
			Position: module.Left,
		},
		{
			Name:     DomainWhoisSuccessPort,
			Label:    "Success",
			Source:   false,
			Message:  DomainWhoisSuccess{},
			Position: module.Right,
		},
		{
			Name:     DomainWhoisErrorPort,
			Label:    "Error",
			Source:   false,
			Message:  DomainWhoisError{},
			Position: module.Right,
		},
	}
}

var _ module.Component = (*Whois)(nil)

func init() {
	registry.Register(&Whois{})
}
