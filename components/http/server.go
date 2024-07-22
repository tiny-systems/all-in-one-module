package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/clbanning/mxj/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/swaggest/jsonschema-go"
	"github.com/tiny-systems/main/pkg/ttlmap"
	"github.com/tiny-systems/main/pkg/utils"
	"github.com/tiny-systems/module/api/v1alpha1"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"go.uber.org/atomic"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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
	ServerStopPort            = "stop"
	ServerStatusPort          = "status"
)

type Server struct {
	//e            *echo.Echo
	settings     ServerSettings
	settingsLock *sync.Mutex
	//
	startSettings ServerStart
	//
	contexts *ttlmap.TTLMap

	publicListenAddrLock *sync.Mutex
	publicListenAddr     []string
	//listenPort           int

	cancelFunc     context.CancelFunc
	cancelFuncLock *sync.Mutex

	runLock *sync.Mutex

	startContext ServerStartContext
	startErr     *atomic.Error
	//
	node v1alpha1.TinyNode

	// k8s client wrapper
	client module.Client
}

func (h *Server) Instance() module.Component {

	return &Server{
		//	e:                    echo.New(),
		publicListenAddr:     []string{},
		publicListenAddrLock: &sync.Mutex{},
		cancelFuncLock:       &sync.Mutex{},
		runLock:              &sync.Mutex{},
		//
		settingsLock: &sync.Mutex{},
		//
		startErr: &atomic.Error{},
		startSettings: ServerStart{
			WriteTimeout: 10,
			ReadTimeout:  60,
			AutoHostName: true,
		},
		settings: ServerSettings{
			EnableStatusPort: false,
			EnableStopPort:   false,
		},
	}
}

type ServerSettings struct {
	EnableStatusPort bool `json:"enableStatusPort" required:"true" title:"Enable status port" description:"Status port notifies when server is up or down"`
	EnableStopPort   bool `json:"enableStopPort" required:"true" title:"Enable stop port" description:"Stop port allows you to stop the server"`
	EnableStartPort  bool `json:"enableStartPort" required:"true" title:"Enable start port" description:"Start port allows you to start the server"`
}

type ServerStartContext any

type ServerStart struct {
	Context      ServerStartContext `json:"context" configurable:"true" title:"Context" description:"Start context" propertyOrder:"1"`
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

type ServerStartControl struct {
	Status string `json:"status" title:"Status" readonly:"true" propertyOrder:"2"`
	Start  bool   `json:"start" format:"button" title:"Start" required:"true" description:"Start HTTP server" propertyOrder:"1"`
}

type ServerStopControl struct {
	Stop       bool     `json:"stop" format:"button" title:"Stop" required:"true" description:"Stop HTTP server" propertyOrder:"1"`
	Status     string   `json:"status" title:"Status" readonly:"true" propertyOrder:"2"`
	ListenAddr []string `json:"listenAddr" title:"Listen Address" readonly:"true" propertyOrder:"3"`
}

type ServerStop struct {
}

type ServerStatus struct {
	Context    ServerStartContext `json:"context" title:"Context" propertyOrder:"1"`
	ListenAddr []string           `json:"listenAddr" title:"Listen Address" readonly:"true" propertyOrder:"2"`
	IsRunning  bool               `json:"isRunning" title:"Is running" readonly:"true" propertyOrder:"3"`
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

func (h *Server) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ServerComponent,
		Description: "HTTP Server",
		Info:        "Serves HTTP requests. Each HTTP requests creates its representing message on a Request port. To display HTTP response incoming message should find its way to the Response port. Other way HTTP request timeout error will be shown.",
		Tags:        []string{"HTTP", "Server"},
	}
}

func (h *Server) stop() error {
	h.cancelFuncLock.Lock()
	defer h.cancelFuncLock.Unlock()
	if h.cancelFunc == nil {
		return nil
	}
	h.cancelFunc()

	return nil
}

func (h *Server) setCancelFunc(f func()) {
	h.cancelFuncLock.Lock()
	defer h.cancelFuncLock.Unlock()
	h.cancelFunc = f
}

func (h *Server) isRunning() bool {
	h.cancelFuncLock.Lock()
	defer h.cancelFuncLock.Unlock()

	return h.cancelFunc != nil
}

func (h *Server) start(ctx context.Context, msg ServerStart, handler module.Handler) error {
	//
	if h.client == nil {
		return fmt.Errorf("unable to start, no client available")
	}
	h.startContext = msg.Context

	if err := h.stop(); err != nil {
		return err
	}
	h.runLock.Lock()
	defer h.runLock.Unlock()

	e := echo.New()
	e.HideBanner = false
	e.HidePort = false

	//h.e = e

	serverCtx, cancel := context.WithCancel(ctx)

	h.setCancelFunc(cancel)
	h.contexts = ttlmap.New(ctx, msg.ReadTimeout*2)

	e.Any("*", func(c echo.Context) error {
		id, err := uuid.NewUUID()
		if err != nil {
			return err
		}

		idStr := id.String()
		requestResult := ServerRequest{
			Context:       msg.Context,
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
		defer close(ch)

		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)

			for {
				select {
				case <-c.Request().Context().Done():
					return

				case <-ctx.Done():
					return

				case <-time.Tick(time.Duration(msg.ReadTimeout) * time.Second):
					c.Error(fmt.Errorf("read timeout"))
					return

				case resp := <-ch:
					for _, header := range resp.Headers {
						c.Response().Header().Set(header.Key, header.Value)
					}
					switch resp.ContentType {
					case MIMEApplicationXML:
						c.XML(resp.StatusCode, resp.Body)
					case MIMEApplicationJSON:
						c.JSON(resp.StatusCode, resp.Body)
					case MIMETextHTML:
						c.HTML(resp.StatusCode, fmt.Sprintf("%v", resp.Body))
					case MimeTextPlain:
						c.String(resp.StatusCode, fmt.Sprintf("%v", resp.Body))
					default:
						c.String(resp.StatusCode, fmt.Sprintf("%v", resp.Body))
					}
					return
				}
			}
		}()

		if err = handler(c.Request().Context(), ServerRequestPort, requestResult); err != nil {
			return err
		}
		<-doneCh
		return nil
	})

	e.Server.ReadTimeout = time.Duration(msg.ReadTimeout) * time.Second
	e.Server.WriteTimeout = time.Duration(msg.WriteTimeout) * time.Second

	var (
		listenPort      int
		actualLocalPort int
	)

	if annotationPort, err := strconv.Atoi(h.node.Labels[v1alpha1.SuggestedHttpPortAnnotation]); err == nil {
		listenPort = annotationPort
	}

	var listenAddr = ":0"

	if listenPort > 0 {
		listenAddr = fmt.Sprintf(":%d", listenPort)
	}

	go func() {
		err := e.Start(listenAddr)
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		h.startErr.Store(err)
	}()

	time.Sleep(time.Millisecond * 1500)

	if e.Listener != nil {
		if tcpAddr, ok := e.Listener.Addr().(*net.TCPAddr); ok {
			//

			actualLocalPort = tcpAddr.Port
			//
			exposeCtx, cancel := context.WithTimeout(ctx, time.Second*30)
			defer cancel()

			// upgrade
			// hostname it's a last part of the node name
			var autoHostName string

			if msg.AutoHostName {
				autoHostNameParts := strings.Split(h.node.Name, ".")
				autoHostName = autoHostNameParts[len(autoHostNameParts)-1]
			}

			publicURLs, err := h.client.ExposePort(exposeCtx, autoHostName, msg.Hostnames, tcpAddr.Port)
			if err != nil {
				h.setPublicListerAddr([]string{fmt.Sprintf("http://localhost:%d", tcpAddr.Port)})
			} else {
				h.setPublicListerAddr(publicURLs)
			}
		}
	}

	// send status that we run
	_ = h.sendStatus(ctx, msg.Context, handler)
	// ask to reconcile (redraw the component)
	_ = handler(ctx, module.ReconcilePort, nil)

	<-serverCtx.Done()

	shutdownCtx, shutDownCancel := context.WithTimeout(context.Background(), time.Second*30)
	defer shutDownCancel()

	_ = e.Shutdown(shutdownCtx)
	h.setCancelFunc(nil)

	//
	discloseCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_ = h.client.DisclosePort(discloseCtx, actualLocalPort)

	// send status when we stopped
	_ = h.sendStatus(ctx, msg.Context, handler)
	// ask to reconcile (redraw the component)
	_ = handler(ctx, module.ReconcilePort, nil)

	return h.startErr.Load()
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

	switch port {
	case module.NodePort:
		h.node, _ = msg.(v1alpha1.TinyNode)

	case module.ClientPort:
		h.client, _ = msg.(module.Client)

	case module.ControlPort:
		if msg == nil {
			break
		}

		switch msg.(type) {
		case ServerStartControl:
			return h.start(ctx, h.startSettings, handler)

		case ServerStopControl:
			return h.stop()
		}

	case module.SettingsPort:
		in, ok := msg.(ServerSettings)
		if !ok {
			return fmt.Errorf("invalid settings message")
		}

		h.settingsLock.Lock()
		h.settings = in
		h.settingsLock.Unlock()

	case ServerStartPort:
		in, ok := msg.(ServerStart)
		if !ok {
			return fmt.Errorf("invalid start message")
		}
		// give time to fail
		return h.start(ctx, in, handler)

	case ServerStopPort:
		return h.stop()

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
			return fmt.Errorf("context '%s' not found", in.RequestID)
		}

		if respChannel, ok := ch.(chan ServerResponse); ok {
			respChannel <- in
		}

	default:
		return fmt.Errorf("port %s is not supported", port)
	}

	return nil
}

func (h *Server) getControl() interface{} {
	if h.isRunning() {
		return ServerStopControl{
			Status:     "Running",
			ListenAddr: h.getPublicListerAddr(),
		}
	}
	return ServerStartControl{
		Status: "Not running",
	}
}

func (h *Server) Ports() []module.Port {

	h.settingsLock.Lock()
	defer h.settingsLock.Unlock()

	ports := []module.Port{
		{
			Name: module.NodePort, // to receive tiny node
		},
		{
			Name: module.ClientPort, // to receive k8s client
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: h.settings,
			Source:        true,
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
			Name:          module.ControlPort,
			Label:         "Dashboard",
			Configuration: h.getControl(),
		},
	}

	if h.settings.EnableStartPort {

		ports = append(ports, module.Port{
			Name:          ServerStartPort,
			Label:         "Start",
			Source:        true,
			Position:      module.Left,
			Configuration: h.startSettings,
		})
	}

	// programmatically stop server
	if h.settings.EnableStopPort {
		ports = append(ports, module.Port{
			Position:      module.Left,
			Name:          ServerStopPort,
			Label:         "Stop",
			Source:        true,
			Configuration: ServerStop{},
		})
	}

	// programmatically use status in flows

	if h.settings.EnableStatusPort {
		ports = append(ports, module.Port{
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
		IsRunning:  h.isRunning(),
	}
}

func (h *Server) sendStatus(ctx context.Context, start ServerStartContext, handler module.Handler) error {
	if !h.settings.EnableStatusPort {
		return nil
	}
	return handler(ctx, ServerStatusPort, ServerStatus{
		Context:    start,
		ListenAddr: h.getPublicListerAddr(),
		IsRunning:  h.isRunning(),
	})
}

var _ module.Component = (*Server)(nil)

func init() {
	registry.Register(&Server{})
}
