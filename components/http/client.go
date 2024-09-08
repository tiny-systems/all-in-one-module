package http

import (
	"bytes"
	"context"
	"fmt"
	"github.com/clbanning/mxj/v2"
	"github.com/spyzhov/ajson"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"io"
	"net/http"
	"strings"
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
	Request ClientRequestRequest `json:"request" title:"Request" required:"true" description:"HTTP Request"`
	//
}

type ClientRequestRequest struct {
	Method  string `json:"method" required:"true" title:"Method" enum:"GET,POST,PATCH,PUT,DELETE" enumTitles:"GET,POST,PATCH,PUT,DELETE" colSpan:"col-span-6"`
	Timeout int    `json:"timeout" required:"true" title:"Request Timeout" colSpan:"col-span-6"`

	URL         string      `json:"url" required:"true" title:"URL" format:"uri"`
	ContentType ContentType `json:"contentType" required:"true"`
	Headers     []Header    `json:"headers" required:"true" title:"Headers"`
	Body        any         `json:"body" configurable:"true" title:"Request Body"`
}

type ClientResponse struct {
	Context  ClientRequestContext   `json:"context" configurable:"true" required:"true" title:"Context" description:"Message to be sent further"`
	Request  ClientRequestRequest   `json:"request" title:"Request" required:"true" description:"HTTP Request"`
	Response ClientResponseResponse `json:"response" title:"Response" required:"true" description:"HTTP Response"`
}

type ClientResponseResponse struct {
	Headers    []Header `json:"headers" required:"true" title:"Headers"`
	Status     string   `json:"status"`
	StatusCode int      `json:"statusCode"`
	Body       any      `json:"response" required:"true" title:"Body"`
}

type ClientError struct {
	Context ClientRequestContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Message to be sent further"`
	Request ClientRequestRequest `json:"request"`
	Error   string               `json:"response"`
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

		ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(in.Request.Timeout))
		defer cancel()

		var requestBody []byte

		switch in.Request.ContentType {
		case MIMEApplicationXML:

		case MIMEApplicationJSON:

		case MIMETextHTML:

		case MimeTextPlain:

		case MIMEApplicationForm:

		case MIMEMultipartForm:

		}

		r, err := http.NewRequestWithContext(ctx, in.Request.Method, in.Request.URL, bytes.NewReader(requestBody))
		if err != nil {
			return err
		}

		c := http.Client{}
		resp, err := c.Do(r)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		cType := resp.Header.Get(HeaderContentType)

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var result interface{}

		switch {
		case strings.HasPrefix(cType, MIMEApplicationJSON):
			root, err := ajson.Unmarshal(b)

			if err != nil {
				if !h.settings.EnableErrorPort {
					return err
				}
				return handler(ctx, ClientErrorPort, ClientError{
					Request: in.Request,
					Error:   err.Error(),
				})
			}

			result, err = root.Unpack()
			if err != nil {
				if !h.settings.EnableErrorPort {
					return err
				}
				return handler(ctx, ClientErrorPort, ClientError{
					Request: in.Request,
					Error:   err.Error(),
				})
			}

		case strings.HasPrefix(cType, MIMEApplicationXML), strings.HasPrefix(cType, MIMETextXML):

			mxj.SetAttrPrefix("")
			m, err := mxj.NewMapXml(b, false)
			if err != nil {
				if !h.settings.EnableErrorPort {
					return err
				}
				return handler(ctx, ClientErrorPort, ClientError{
					Request: in.Request,
					Error:   err.Error(),
				})
			}

			result = m.Old()

		default:
			builder := strings.Builder{}
			builder.Write(b)
			result = builder.String()
		}

		var headers []Header
		for k, v := range resp.Header {
			for _, vv := range v {
				headers = append(headers, Header{
					Key:   k,
					Value: vv,
				})
			}
		}

		return handler(ctx, ClientResponsePort, ClientResponse{
			Request: in.Request,
			Response: ClientResponseResponse{
				Body:       result,
				Headers:    headers,
				Status:     resp.Status,
				StatusCode: resp.StatusCode,
			},
			Context: in.Context,
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
				Request: ClientRequestRequest{
					Method:      http.MethodGet,
					Headers:     make([]Header, 0),
					URL:         "http://example.com",
					Timeout:     10,
					ContentType: "application/json",
				},
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
