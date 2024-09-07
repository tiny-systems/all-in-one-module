package http

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"io"
	"net/http"
	"time"
)

const (
	ClientComponent    = "http_client"
	ClientRequestPort  = "request"
	ClientResponsePort = "response"
	ClientErrorPort    = "error"
)

type Header struct {
	Key   string `json:"key" required:"true" title:"Key" colSpan:"col-span-6"`
	Value string `json:"value" required:"true" title:"Value" colSpan:"col-span-6"`
}

type ClientRequestContext any

type ClientRequestSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

type ClientRequest struct {
	Context ClientRequestContext `json:"context,omitempty" configurable:"true" title:"Context" description:"Message to be sent further"`
	Method  string               `json:"method" required:"true" title:"Method" enum:"GET,POST,PATCH,PUT,DELETE" enumTitles:"GET,POST,PATCH,PUT,DELETE" colSpan:"col-span-6"`
	Timeout int                  `json:"timeout" required:"true" title:"Request Timeout" colSpan:"col-span-6"`

	URL     string   `json:"url" required:"true" title:"URL" format:"uri"`
	Headers []Header `json:"headers" required:"true" title:"Headers"`
	Body    string   `json:"body" format:"textarea" title:"Request Body"`
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

type ClientError struct {
	Request ClientRequest `json:"request"`
	Error   string        `json:"response"`
}

type Client struct {
	settings ClientRequestSettings
}

func (h *Client) Instance() module.Component {
	return &Client{}
}

func (h *Client) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ClientComponent,
		Description: "HTTP Client",
		Info:        "Performs HTTP requests.",
		Tags:        []string{"HTTP", "Client"},
	}
}

func (h *Client) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	switch port {
	case module.SettingsPort:
		// compile template
		in, ok := msg.(ClientRequestSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		h.settings = in

		return nil

	case ClientRequestPort:
		in, ok := msg.(ClientRequest)
		if !ok {
			return fmt.Errorf("invalid message")
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(in.Timeout))
		defer cancel()

		r, err := http.NewRequestWithContext(ctx, in.Method, in.URL, nil)
		if err != nil {
			return err
		}

		c := http.Client{}

		resp, err := c.Do(r)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return handler(ctx, ClientResponsePort, ClientResponse{
			Response: string(b),
			Request:  in,
		})

	default:
		return fmt.Errorf("port %s is not supoprted", port)
	}

}

func (h *Client) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:   ClientRequestPort,
			Label:  "Request",
			Source: true,
			Configuration: ClientRequest{
				Method:  http.MethodGet,
				Headers: make([]Header, 0),
				URL:     "http://example.com",
				Timeout: 10,
			},
			Position: module.Left,
		},

		{
			Name:          ClientResponsePort,
			Label:         "Response",
			Position:      module.Right,
			Configuration: ClientResponse{},
		},

		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: h.settings,
			Source:        true,
		},
	}

	if !h.settings.EnableErrorPort {
		return ports
	}

	return append(ports, module.Port{
		Name:          ClientErrorPort,
		Label:         "Error",
		Source:        false,
		Position:      module.Bottom,
		Configuration: ClientError{},
	})
}

var _ module.Component = (*Client)(nil)

func init() {
	registry.Register(&Client{})
}
