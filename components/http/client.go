package http

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"io"
	"net/http"
	url2 "net/url"
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
	Method  string            `json:"method" required:"true" title:"Method" enum:"GET,POST,PATCH,PUT,DELETE" enumTitles:"GET,POST,PATCH,PUT,DELETE" colSpan:"col-span-3" propertyOrder:"1"`
	URL     string            `json:"url" required:"true" title:"URL" format:"uri" propertyOrder:"2"`
	Headers []Header          `json:"headers" required:"true" title:"Headers" propertyOrder:"3"`
	Body    ClientRequestBody `json:"body" configurable:"true" title:"Request Body" propertyOrder:"4"`
}

type Response struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"statusCode"`
	Body       interface{} `json:"body"`
}

type ClientResponse struct {
	Request  ClientRequest `json:"request"`
	Response string        `json:"response"`
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

	if port != "request" {
		return fmt.Errorf("invalid port")
	}

	in, ok := message.(ClientRequest)
	if !ok {
		return fmt.Errorf("invalid message")
	}

	url, err := url2.Parse(in.URL)
	if err != nil {
		return err
	}

	c := http.Client{}

	r := &http.Request{
		URL:    url,
		Method: in.Method,
	}

	resp, err := c.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return handler("response", ClientResponse{
		Response: string(b),
		Request:  in,
	})
}

func (h Client) Ports() []module.NodePort {
	return []module.NodePort{
		{
			Name:   "request",
			Label:  "Request",
			Source: true,
			Configuration: ClientRequest{
				Request: Request{
					Method:  "get",
					Headers: make([]Header, 0),
					URL:     "http://example.com",
				},
			},
			Position: module.Left,
		},
		{
			Name:          "response",
			Label:         "Response",
			Position:      module.Left,
			Configuration: ClientResponse{},
		},
	}
}

var _ module.Component = (*Client)(nil)

func init() {
	registry.Register(&Client{})
}
