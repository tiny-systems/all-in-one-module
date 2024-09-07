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
	"time"
)

const (
	CalendarGetEventsComponent    = "google_calendar_get_events"
	CalendarGetEventsRequestPort  = "request"
	CalendarGetEventsResponsePort = "response"
	CalendarGetEventsErrorPort    = "error"
)

type CalendarGetEventsContext any

type CalendarGetEventsRequest struct {
	Context     CalendarGetEventsContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be send further"`
	CalendarId  string                   `json:"calendarId" required:"true" default:"primary" minLength:"1" title:"Calendar ID"`
	ShowDeleted bool                     `json:"showDeleted" required:"true" title:"Show deleted events" default:"true"`
	StartDate   time.Time                `json:"startDate" title:"Start date"`
	EndDate     time.Time                `json:"endDate" title:"End date"`
	SyncToken   string                   `json:"syncToken" title:"Sync Token"`
	PageToken   string                   `json:"pageToken" title:"Page Token"`
	Token       Token                    `json:"token" required:"true" title:"Auth Token"`
	Config      ClientConfig             `json:"config" required:"true" title:"Client credentials"`
}

type ClientConfig struct {
	Credentials string   `json:"credentials" required:"true" format:"textarea" title:"Credentials" description:"Google client credentials.json file content"`
	Scopes      []string `json:"scopes" title:"Scopes" required:"true"`
}

type CalendarGetEventsError struct {
	Request CalendarGetEventsRequest `json:"request"`
	Error   string                   `json:"error"`
}

type CalendarGetEventResponse struct {
	Request CalendarGetEventsRequest `json:"request"`
	Results calendar.Events          `json:"results"`
}

type CalendarGetEvents struct {
	settings CalendarGetEventsSettings
}

type CalendarGetEventsSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" default:"false" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

func (c *CalendarGetEvents) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        CalendarGetEventsComponent,
		Description: "Calendar Get Events",
		Info:        "Calendar Get Events",
		Tags:        []string{"Google", "Calendar"},
	}
}

func (c *CalendarGetEvents) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(CalendarGetEventsSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		c.settings = in
		return nil
	}

	if port != CalendarGetEventsRequestPort {
		return fmt.Errorf("unknown port %s", CalendarGetEventsRequestPort)
	}

	req, ok := msg.(CalendarGetEventsRequest)
	if !ok {
		return fmt.Errorf("invalid message")
	}
	events, err := c.getEvents(ctx, req)
	if err != nil {
		if !c.settings.EnableErrorPort {
			return err
		}
		return handler(ctx, CalendarGetEventsErrorPort, CalendarGetEventsError{
			Request: req,
			Error:   err.Error(),
		})
	}

	return handler(ctx, CalendarGetEventsResponsePort, CalendarGetEventResponse{
		Request: req,
		Results: *events,
	})
}

func (c *CalendarGetEvents) getEvents(ctx context.Context, req CalendarGetEventsRequest) (*calendar.Events, error) {

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

	call := srv.Events.List(req.CalendarId).ShowDeleted(req.ShowDeleted).SingleEvents(true)

	if !req.StartDate.IsZero() {
		call.TimeMin(req.StartDate.Format(time.RFC3339))
	}
	if !req.EndDate.IsZero() {
		call.TimeMax(req.EndDate.Format(time.RFC3339))
	}

	if req.PageToken != "" {
		call.PageToken(req.PageToken)
	}
	if req.SyncToken != "" {
		call.SyncToken(req.SyncToken)
	}

	call.MaxResults(100).OrderBy("startTime")

	events, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve user's events: %v", err)
	}

	return events, nil
}

func (c *CalendarGetEvents) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: CalendarGetEventsSettings{},
			Source:        true,
		},
		{
			Name:  CalendarGetEventsRequestPort,
			Label: "Request",
			Configuration: CalendarGetEventsRequest{
				Config: ClientConfig{
					Scopes: []string{"https://www.googleapis.com/auth/calendar.events.readonly"},
				},
				CalendarId: "SomeID",
				Token: Token{
					TokenType: "Bearer",
				},
			},
			Source:   true,
			Position: module.Left,
		},
		{
			Name:          CalendarGetEventsResponsePort,
			Label:         "Response",
			Source:        false,
			Position:      module.Right,
			Configuration: CalendarGetEventResponse{},
		}}

	if !c.settings.EnableErrorPort {
		return ports
	}

	return append(ports, module.Port{
		Position:      module.Bottom,
		Name:          CalendarGetEventsErrorPort,
		Label:         "Error",
		Source:        false,
		Configuration: CalendarGetEventsError{},
	})
}

func (c *CalendarGetEvents) Instance() module.Component {
	return &CalendarGetEvents{}
}

var _ module.Component = (*CalendarGetEvents)(nil)

func init() {
	registry.Register(&CalendarGetEvents{})
}
