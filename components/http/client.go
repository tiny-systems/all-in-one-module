package http

import (
	"context"
	"github.com/tiny-systems/module/pkg/module"
	"github.com/tiny-systems/module/registry"
)

const ClientComponent = "http_client"

type Header struct {
	Key   string `json:"key" required:"true" title:"Key" colSpan:"col-span-6"`
	Value string `json:"value" required:"true" title:"Value" colSpan:"col-span-6"`
}

type ClientRequestContext any
type ClientRequestBody any

type ClientRequest struct {
	Context ClientRequestContext `json:"context" configurable:"true" title:"Context" description:"Message to be sent further" propertyOrder:"1"`
	Request `json:"request" title:"HTTP request" required:"true" propertyOrder:"2"`
}

type Request struct {
	Method  string            `json:"method" required:"true" title:"Method" enum:"get,post,patch,put,delete" enumTitles:"GET,POST,PATCH,PUT,DELETE" colSpan:"col-span-3" propertyOrder:"1"`
	URL     string            `json:"url" required:"true" title:"URL" format:"uri" propertyOrder:"2"`
	Headers []Header          `json:"headers" required:"true" title:"Headers" propertyOrder:"3"`
	Body    ClientRequestBody `json:"body" configurable:"true" title:"Request Body" propertyOrder:"4"`
}

type ClientResponse struct {
	Request ClientRequest `json:"request"`
}

type Client struct {
}

func (h Client) Instance() module.Component {
	return Client{}
}

func (h Client) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ClientComponent,
		Description: "HTTP Client",
		Info:        "Performs HTTP requests.",
		Tags:        []string{"HTTP", "Client"},
	}
}

func (h Client) Handle(ctx context.Context, handler module.Handler, port string, message interface{}) error {
	return nil
}

func (h Client) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:   "request",
			Label:  "Request",
			Source: true,
			Message: ClientRequest{
				Request: Request{
					Method:  "get",
					Headers: make([]Header, 0),
					URL:     "https://example.com",
				},
			},
			Position: module.Left,
		},
		{
			Name:     "response",
			Label:    "Response",
			Position: module.Left,
			Message:  ClientResponse{},
		},
	}
}

var _ module.Component = (*Client)(nil)

func init() {
	registry.Register(&Client{})
}
