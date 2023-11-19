package http

import (
	"context"
	"fmt"
	"github.com/clbanning/mxj/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/swaggest/jsonschema-go"
	"github.com/tiny-systems/main/pkg/ttlmap"
	"github.com/tiny-systems/main/pkg/utils"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	HeaderContentType   = "Content-Type"
	MIMEApplicationJSON = "application/json"
	MIMEApplicationXML  = "application/xml"
	MIMETextXML         = "text/xml"
	MimeTextPlain       = "text/plain"
	MIMETextHTML        = "text/html"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
	MIMEMultipartForm   = "multipart/form-data"
)

const (
	ServerComponent    string = "http_server"
	ServerResponsePort        = "response"
	ServerRequestPort         = "request"
	ServerControlPort         = "control"
	ServerStatusPort          = "status"
)

type Server struct {
	settings         ServerSettings
	contexts         *ttlmap.TTLMap
	addressGetter    module.ListenAddressGetter
	publicListenAddr string
	listenPort       int
}

func (h *Server) HTTPService(getter module.ListenAddressGetter) {
	h.addressGetter = getter
}

type ServerSettings struct {
	WriteTimeout      int  `json:"writeTimeout" required:"true" title:"Write Timeout" description:"Covers the time from the end of the request header read to the end of the response write"`
	EnableControlPort bool `json:"enableControlPort" required:"true" title:"Enable control port" description:"Control port allows control server externally"`
}

type ServerRequest struct {
	RequestID     string     `json:"requestID" required:"true"`
	RequestURI    string     `json:"requestURI" required:"true"`
	RequestParams url.Values `json:"requestParams" required:"true"`
	Host          string     `json:"host" required:"true"`
	Method        string     `json:"method" required:"true" title:"Method" enum:"GET,POST,PATCH,PUT,DELETE" enumTitles:"GET,POST,PATCH,PUT,DELETE"`
	RealIP        string     `json:"realIP"`
	Headers       []Header   `json:"headers,omitempty"`
	Body          any        `json:"body"`
	Scheme        string     `json:"scheme"`
}

type ServerControlRequest struct {
	Start bool `json:"start" required:"true" title:"Server state"`
}

type ServerStatus struct {
	ListenAddr string `json:"listenAddr" readonly:"true" title:"Listen Address"`
}

type ServerResponseBody any

type ServerResponse struct {
	RequestID   string             `json:"requestID" required:"true" title:"Request ID" minLength:"1" description:"To match response with request pass request ID to response port" propertyOrder:"1"`
	StatusCode  int                `json:"statusCode" required:"true" title:"Status Code" description:"HTTP status code for response" minimum:"100" default:"200" maximum:"599" propertyOrder:"2"`
	ContentType ContentType        `json:"contentType" required:"true" propertyOrder:"3"`
	Headers     []Header           `json:"headers"  title:"Response headers" propertyOrder:"4"`
	Body        ServerResponseBody `json:"body" title:"Response body" configurable:"true" propertyOrder:"5"`
}

type ContentType string

func (c ContentType) JSONSchema() (jsonschema.Schema, error) {
	contentType := jsonschema.Schema{}
	contentType.AddType(jsonschema.String)
	contentType.WithTitle("Content Type").
		WithDefault(200).
		WithEnum(MIMEApplicationJSON, MIMEApplicationXML, MIMETextHTML, MimeTextPlain).
		WithDefault(MIMEApplicationJSON).
		WithDescription("Content type of the response").
		WithExtraProperties(map[string]interface{}{
			"propertyOrder": 3,
		})
	return contentType, nil
}

func (h *Server) Instance() module.Component {
	return &Server{
		publicListenAddr: "",
		settings: ServerSettings{
			WriteTimeout: 10,
		},
	}
}

func (h *Server) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ServerComponent,
		Description: "HTTP Server",
		Info:        "Serves HTTP requests. Each HTTP requests creates its representing message on a Request port. To display HTTP response incoming message should find its way to the Response port. Other way HTTP request timeout error will be shown.",
		Tags:        []string{"HTTP", "Server"},
	}
}

func (h *Server) Emit(ctx context.Context, handler module.Handler) error {
	h.contexts = ttlmap.New(ctx, h.settings.WriteTimeout)
	e := echo.New()

	//	e.HideBanner = true
	//	e.HidePort = true

	e.Any("*", func(c echo.Context) error {
		id, err := uuid.NewUUID()
		if err != nil {
			return err
		}

		idStr := id.String()
		requestResult := ServerRequest{
			RequestID:     idStr,
			Host:          c.Request().Host,
			Method:        c.Request().Method,
			RequestURI:    c.Request().RequestURI,
			RequestParams: c.QueryParams(),
			RealIP:        c.RealIP(),
			Scheme:        c.Scheme(),
			Headers:       make([]Header, 0),
		}
		req := c.Request()

		keys := make([]string, 0, len(req.Header))
		for k := range req.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range req.Header[k] {
				requestResult.Headers = append(requestResult.Headers, Header{
					Key:   k,
					Value: v,
				})
			}
		}

		cType := req.Header.Get(HeaderContentType)
		switch {
		case strings.HasPrefix(cType, MIMEApplicationJSON):
			if err = c.Echo().JSONSerializer.Deserialize(c, &requestResult.Body); err != nil {
				switch err.(type) {
				case *echo.HTTPError:
					return err
				default:
					return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
				}
			}
		case strings.HasPrefix(cType, MIMEApplicationXML), strings.HasPrefix(cType, MIMETextXML):
			mxj.SetAttrPrefix("")
			m, err := mxj.NewMapXmlReader(req.Body, false)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			requestResult.Body = m.Old()

		case strings.HasPrefix(cType, MIMEApplicationForm), strings.HasPrefix(cType, MIMEMultipartForm):
			params, err := c.FormParams()
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			requestResult.Body = params
		default:
			body, _ := io.ReadAll(req.Body)
			requestResult.Body = utils.BytesToString(body)
		}

		ch := make(chan ServerResponse)
		h.contexts.Put(idStr, ch)

		if err = handler(ServerRequestPort, requestResult); err != nil {
			return err
		}

		for {
			select {
			case <-time.Tick(time.Duration(h.settings.WriteTimeout) * time.Second):
				err = fmt.Errorf("response timeout")
				c.Error(err)
				return err
			case resp := <-ch:
				for _, h := range resp.Headers {
					c.Response().Header().Set(h.Key, h.Value)
				}
				switch resp.ContentType {
				case MIMEApplicationXML:
					return c.XML(resp.StatusCode, resp.Body)
				case MIMEApplicationJSON:
					return c.JSON(resp.StatusCode, resp.Body)
				case MIMETextHTML:
					return c.HTML(resp.StatusCode, fmt.Sprintf("%v", resp.Body))
				case MimeTextPlain:
					return c.String(resp.StatusCode, fmt.Sprintf("%v", resp.Body))
				default:
					return c.String(resp.StatusCode, fmt.Sprintf("%v", resp.Body))
				}
			}
		}
	})

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
	}()
	e.Server.WriteTimeout = time.Duration(h.settings.WriteTimeout) * time.Second

	var (
		ch      = make(chan struct{}, 0)
		upgrade module.AddressUpgrade
		err     error
	)

	if h.addressGetter != nil {
		h.listenPort, upgrade = h.addressGetter()
	}

	var listenAddr = ":0"

	if h.listenPort > 0 {
		listenAddr = fmt.Sprintf(":%d", h.listenPort)
	}

	go func() {
		err = e.Start(listenAddr)
	}()

	go func() {
		defer close(ch)
		time.Sleep(time.Second)
		if e.Listener != nil {
			if tcpAddr, ok := e.Listener.Addr().(*net.TCPAddr); ok {
				h.publicListenAddr, err = upgrade(tcpAddr.Port)
				if !strings.HasPrefix(h.publicListenAddr, "https://") {
					h.publicListenAddr = fmt.Sprintf("https://%s", h.publicListenAddr)
				}
				if err != nil {
					h.publicListenAddr = fmt.Sprintf("ERROR: %s", err.Error())
				}
			}
		}
		<-ctx.Done()
	}()

	<-ch
	return err
}

func (h *Server) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(ServerSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		h.settings = in
		return nil
	}

	if port == ServerControlPort {
		return nil
	}

	in, ok := msg.(ServerResponse)
	if !ok {
		return fmt.Errorf("invalid response message")
	}

	if h.contexts == nil {
		return fmt.Errorf("unknown request ID %s", in.RequestID)
	}

	ch := h.contexts.Get(in.RequestID)
	if ch == nil {
		return fmt.Errorf("context not found %s", in.RequestID)
	}

	if respChannel, ok := ch.(chan ServerResponse); ok {
		respChannel <- in
	}
	return nil
}

func (h *Server) Ports() []module.NodePort {

	ports := []module.NodePort{
		{
			Name:   module.StatusPort,
			Label:  "Status",
			Source: true,
			Status: true,
			Configuration: ServerStatus{
				ListenAddr: h.publicListenAddr,
			},
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: h.settings,
			Source:        true,
			Settings:      true,
		},
		{
			Name:          ServerRequestPort,
			Label:         "Request",
			Configuration: ServerRequest{},
			Position:      module.Right,
		},
		{
			Name:     ServerResponsePort,
			Label:    "Response",
			Source:   true,
			Position: module.Right,
			Configuration: ServerResponse{
				StatusCode: 200,
			},
		},
	}

	if h.settings.EnableControlPort {
		ports = append(ports, module.NodePort{
			Position:      module.Left,
			Name:          ServerControlPort,
			Label:         "Control",
			Source:        true,
			Configuration: ServerControlRequest{},
		})
	}

	return ports
}

var _ module.Component = (*Server)(nil)
var _ module.Emitter = (*Server)(nil)
var _ module.HTTPService = (*Server)(nil)

func init() {
	registry.Register(&Server{})
}
