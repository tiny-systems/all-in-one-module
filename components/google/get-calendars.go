package google

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	GetCalendarsComponent    = "google_get_calendars"
	GetCalendarsRequestPort  = "request"
	GetCalendarsResponsePort = "response"
	GetCalendarsErrorPort    = "error"
)

type GetCalendarsContext any

type GetCalendarsSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

type GetCalendars struct {
	settings GetCalendarsSettings
}

type GetCalendarsRequest struct {
	Context GetCalendarsContext `json:"context" title:"Context" configurable:"true" propertyOrder:"1"`
	Config  ClientConfig        `json:"config" title:"Config"  required:"true" description:"Client Config" propertyOrder:"2"`
	Token   Token               `json:"token" required:"true" title:"Auth Token" propertyOrder:"7"`
}

type GetCalendarsResponse struct {
	Request   GetCalendarsRequest           `json:"request"`
	Calendars []*calendar.CalendarListEntry `json:"calendars"`
}

type GetCalendarsError struct {
	Request GetCalendarsRequest `json:"request"`
	Error   string              `json:"error"`
}

func (g *GetCalendars) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        GetCalendarsComponent,
		Description: "Get calendars list",
		Info:        "Gets list of Google calendars",
		Tags:        []string{"google", "calendar", "auth"},
	}
}

func (g *GetCalendars) Handle(ctx context.Context, output module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(GetCalendarsSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		g.settings = in
		return nil
	}

	if port != ExchangeAuthCodeRequestPort {
		return fmt.Errorf("unknown port %s", port)
	}

	in, ok := msg.(GetCalendarsRequest)
	if !ok {
		return fmt.Errorf("invalid input message")
	}

	calendars, err := g.getCalendars(ctx, in)
	if err != nil {
		// check err port
		if g.settings.EnableErrorPort {
			return output(ctx, ExchangeAuthCodeErrorPort, GetCalendarsError{
				Request: in,
				Error:   err.Error(),
			})
		}
		return err
	}

	return output(ctx, GetCalendarsResponsePort, GetCalendarsResponse{
		Request:   in,
		Calendars: calendars,
	})
}

func (c *GetCalendars) getCalendars(ctx context.Context, req GetCalendarsRequest) ([]*calendar.CalendarListEntry, error) {

	config, err := google.ConfigFromJSON([]byte(req.Config.Credentials), req.Config.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	client := config.Client(ctx, &oauth2.Token{
		AccessToken:  req.Token.AccessToken,
		RefreshToken: req.Token.RefreshToken,
		Expiry:       req.Token.Expiry,
		TokenType:    req.Token.TokenType,
	})

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve calendar client: %v", err)
	}

	list, err := srv.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (g *GetCalendars) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: GetCalendarsSettings{},
			Source:        true,
		},
		{
			Source:        true,
			Name:          GetCalendarsRequestPort,
			Label:         "Request",
			Position:      module.Left,
			Configuration: GetCalendarsRequest{},
		},
		{
			Source:        false,
			Name:          GetCalendarsResponsePort,
			Label:         "Response",
			Position:      module.Right,
			Configuration: GetCalendarsResponse{},
		},
	}
	if !g.settings.EnableErrorPort {
		return ports
	}

	return append(ports, module.NodePort{
		Position:      module.Bottom,
		Name:          GetCalendarsErrorPort,
		Label:         "Error",
		Source:        false,
		Configuration: GetCalendarsError{},
	})
}

func (g *GetCalendars) Instance() module.Component {
	return &GetCalendars{}
}

var _ module.Component = (*GetCalendars)(nil)

func init() {
	registry.Register(&GetCalendars{})
}
