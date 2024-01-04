package http

import (
	"context"
	"fmt"
	"github.com/clbanning/mxj/v2"
	"github.com/davecgh/go-spew/spew"
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
	"sync"
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
	ServerStartPort           = "start"
	ServerControlPort         = "control"
	ServerStopPort            = "stop"
	ServerStatusPort          = "status"
)

type Server struct {
	settings     ServerSettings
	settingsLock *sync.Mutex

	contexts      *ttlmap.TTLMap
	addressGetter module.ListenAddressGetter

	publicListenAddrLock *sync.Mutex
	publicListenAddr     []string
	//listenPort           int

	cancelFunc     context.CancelFunc
	cancelFuncLock *sync.Mutex
}

func (h *Server) HTTPService(getter module.ListenAddressGetter) {
	h.addressGetter = getter
}

type ServerSettings struct {
	EnableStatusPort bool `json:"enableStatusPort" required:"true" title:"Enable status port" description:"Status port notifies when server is up or down"`
	EnableStopPort   bool `json:"enableStopPort" required:"true" title:"Enable stop port" description:"Stop port allows you to stop the server"`
}

type ServerStartContext any

type ServerStart struct {
	Context      ServerStartContext `json:"context" configurable:"true" title:"Context" propertyOrder:"1"`
	AutoHostName bool               `json:"autoHostName" title:"Automatically generate hostname" description:"Use cluster auto subdomain setup if any." propertyOrder:"2"`
	Hostnames    []string           `json:"hostnames" title:"Hostnames" required:"false" description:"List of virtual host this server should be bound to." propertyOrder:"3"` //requiredWhen:"['kind', 'equal', 'enum 1']"
	ReadTimeout  int                `json:"readTimeout" required:"true" title:"Read Timeout" description:"Read timeout is the maximum duration for reading the entire request, including the body. A zero or negative value means there will be no timeout." propertyOrder:"4"`
	WriteTimeout int                `json:"writeTimeout" required:"true" title:"Write Timeout" description:"Write timeout is the maximum duration before timing out writes of the response. It is reset whenever a new request's header is read." propertyOrder:"5"`
}

type ServerRequest struct {
	Context       ServerStartContext `json:"context"`
	RequestID     string             `json:"requestID" required:"true"`
	RequestURI    string             `json:"requestURI" required:"true"`
	RequestParams url.Values         `json:"requestParams" required:"true"`
	Host          string             `json:"host" required:"true"`
	Method        string             `json:"method" required:"true" title:"Method" enum:"GET,POST,PATCH,PUT,DELETE" enumTitles:"GET,POST,PATCH,PUT,DELETE"`
	RealIP        string             `json:"realIP"`
	Headers       []Header           `json:"headers,omitempty"`
	Body          any                `json:"body"`
	Scheme        string             `json:"scheme"`
}

type ServerStop struct {
}

type ServerError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ServerStatus struct {
	Context    ServerStartContext `json:"context" title:"Context" propertyOrder:"1"`
	ListenAddr []string           `json:"listenAddr" title:"Listen Address" readonly:"true" propertyOrder:"2"`
	Error      *ServerError       `json:"error" title:"Error" readonly:"true" propertyOrder:"3"`
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
		publicListenAddr:     []string{},
		publicListenAddrLock: &sync.Mutex{},
		cancelFuncLock:       &sync.Mutex{},
		settingsLock:         &sync.Mutex{},
		settings: ServerSettings{
			EnableStatusPort: false,
			EnableStopPort:   false,
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

func (h *Server) stop(ctx context.Context, msg ServerStop, handler module.Handler) error {
	h.cancelFuncLock.Lock()
	defer h.cancelFuncLock.Unlock()
	if h.cancelFunc != nil {
		h.cancelFunc()
	}
	return nil
}

func (h *Server) start(ctx context.Context, msg ServerStart, handler module.Handler) error {

	fmt.Println("START", msg)
	ctx, cancel := context.WithCancel(ctx)

	h.cancelFuncLock.Lock()
	h.cancelFunc = cancel
	h.cancelFuncLock.Unlock()

	h.contexts = ttlmap.New(ctx, msg.ReadTimeout+msg.ReadTimeout)
	e := echo.New()

	e.HideBanner = false
	e.HidePort = false

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
			case <-time.Tick(time.Duration(msg.ReadTimeout) * time.Second):
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

	e.Server.ReadTimeout = time.Duration(msg.ReadTimeout) * time.Second
	e.Server.WriteTimeout = time.Duration(msg.WriteTimeout) * time.Second

	var (
		ch         = make(chan struct{}, 0)
		upgrade    module.AddressUpgrade
		err        error
		listenPort int
	)

	if h.addressGetter != nil {
		listenPort, upgrade = h.addressGetter()
	}

	var listenAddr = ":0"

	if listenPort > 0 {
		listenAddr = fmt.Sprintf(":%d", listenPort)
	}

	go func() {
		err = e.Start(listenAddr)
	}()

	go func() {
		defer close(ch)
		time.Sleep(time.Second)
		if e.Listener != nil {
			if tcpAddr, ok := e.Listener.Addr().(*net.TCPAddr); ok {
				hostnames, err := upgrade(ctx, msg.AutoHostName, msg.Hostnames, tcpAddr.Port)
				if err != nil {
					h.setPublicListerAddr([]string{fmt.Sprintf("ERROR: %s", err.Error())})
					return
				}
				h.setPublicListerAddr(hostnames)
			}
		}
		<-ctx.Done()
	}()

	<-ch
	return err
}

func (h *Server) setPublicListerAddr(addr []string) {
	h.publicListenAddrLock.Lock()
	defer h.publicListenAddrLock.Unlock()
	h.publicListenAddr = addr
}

func (h *Server) getPublicListerAddr() []string {
	h.publicListenAddrLock.Lock()
	defer h.publicListenAddrLock.Unlock()
	return h.publicListenAddr
}

func (h *Server) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	spew.Dump(port, msg)

	switch port {
	case module.SettingsPort:
		in, ok := msg.(ServerSettings)
		if !ok {
			return fmt.Errorf("invalid settings message")
		}

		h.settingsLock.Lock()
		defer h.settingsLock.Unlock()

		h.settings = in
		return nil

	case ServerStartPort:
		in, ok := msg.(ServerStart)
		if !ok {
			return fmt.Errorf("invalid start message")
		}
		return h.start(ctx, in, handler)

	case ServerStopPort:
		in, ok := msg.(ServerStop)
		if !ok {
			return fmt.Errorf("invalid stop message")
		}
		return h.stop(ctx, in, handler)

	case ServerResponsePort:
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
	return fmt.Errorf("port %s is not supported", port)
}

func (h *Server) Ports() []module.NodePort {

	h.settingsLock.Lock()
	defer h.settingsLock.Unlock()

	ports := []module.NodePort{
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
		{
			Name:     ServerStartPort,
			Label:    "Start",
			Source:   true,
			Position: module.Left,
			Configuration: ServerStart{
				WriteTimeout: 10,
				ReadTimeout:  60,
				AutoHostName: true,
			},
		},
		{
			Name:          ServerControlPort,
			Label:         "Status",
			Source:        true,
			Control:       true,
			Configuration: h.getStatus(),
		},
	}

	// programmatically stop server
	if h.settings.EnableStopPort {
		ports = append(ports, module.NodePort{
			Position:      module.Left,
			Name:          ServerStopPort,
			Label:         "Stop",
			Source:        true,
			Configuration: ServerStop{},
		})
	}

	// programmatically use status in flows
	if h.settings.EnableStatusPort {
		ports = append(ports, module.NodePort{
			Position:      module.Bottom,
			Name:          ServerStatusPort,
			Label:         "Status",
			Configuration: h.getStatus(),
		})
	}

	return ports
}

func (h *Server) getStatus() ServerStatus {
	return ServerStatus{
		ListenAddr: h.getPublicListerAddr(),
		Error:      nil,
	}
}

var _ module.Component = (*Server)(nil)

// var _ module.Emitter = (*Server)(nil)
var _ module.HTTPService = (*Server)(nil)

func init() {
	registry.Register(&Server{})
}
