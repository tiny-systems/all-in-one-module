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

const CalendarGetEventsComponent = "google_calendar_get_events"

type CalendarGetEventsContext any

type CalendarGetEventsRequestPort struct {
	Context CalendarGetEventsContext `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be send further" propertyOrder:"1"`
	Request CalendarGetEventsRequest `json:"request" title:"Request" propertyOrder:"2"`
}

type CalendarGetEventsRequest struct {
	CalendarId  string       `json:"calendarId" required:"true" default:"primary" minLength:"1" title:"Calendar ID" propertyOrder:"1"`
	ShowDeleted bool         `json:"showDeleted" required:"true" title:"Show deleted events" default:"true" propertyOrder:"2"`
	StartDate   time.Time    `json:"startDate" title:"Start date" propertyOrder:"3"`
	EndDate     time.Time    `json:"endDate" title:"End date" propertyOrder:"4"`
	SyncToken   string       `json:"syncToken" title:"Sync Token" propertyOrder:"5"`
	Token       Token        `json:"token" required:"true" title:"Auth Token" propertyOrder:"6"`
	Config      ClientConfig `json:"config" required:"true" title:"Client credentials" propertyOrder:"7"`
}

type ClientConfig struct {
	Credentials string   `json:"credentials" required:"true" format:"textarea" title:"Credentials" description:"Google client credentials.json file content"`
	Scopes      []string `json:"scopes" title:"Scopes" required:"true"`
}

type CalendarGetEventsError struct {
	Context CalendarGetEventsContext `json:"context"`
	Request CalendarGetEventsRequest `json:"request"`
	Error   string                   `json:"error"`
}

type CalendarGetEventSuccess struct {
	Context CalendarGetEventsContext `json:"context"`
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

func (c *CalendarGetEvents) Handle(ctx context.Context, output module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(CalendarGetEventsSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		c.settings = in
		return nil
	}

	req, ok := msg.(CalendarGetEventsRequestPort)
	if !ok {
		return fmt.Errorf("invalid message")
	}
	events, err := c.getEvents(ctx, req)

	if err != nil && c.settings.EnableErrorPort {
		_ = output("error", CalendarGetEventsError{
			Context: req.Context,
			Request: req.Request,
			Error:   err.Error(),
		})
		return err
	}

	return output("success", CalendarGetEventSuccess{
		Request: req.Request,
		Context: req.Context,
		Results: *events,
	})

}

func (c *CalendarGetEvents) getEvents(ctx context.Context, req CalendarGetEventsRequestPort) (*calendar.Events, error) {
	config, err := google.ConfigFromJSON([]byte(req.Request.Config.Credentials), calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	client := config.Client(ctx, &oauth2.Token{
		AccessToken:  req.Request.Token.AccessToken,
		RefreshToken: req.Request.Token.RefreshToken,
		Expiry:       req.Request.Token.Expiry,
		TokenType:    req.Request.Token.TokenType,
	})

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve calendar client: %v", err)
	}

	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(req.Request.CalendarId).ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve user's events: %v", err)
	}

	return events, nil
}

func (c *CalendarGetEvents) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:     module.SettingsPort,
			Label:    "Settings",
			Message:  CalendarGetEventsSettings{},
			Source:   true,
			Settings: true,
		},
		{
			Name:  "request",
			Label: "Request",
			Message: CalendarGetEventsRequestPort{
				Request: CalendarGetEventsRequest{
					Config: ClientConfig{
						Scopes: []string{"https://www.googleapis.com/auth/calendar.events.readonly"},
					},
					CalendarId: "SomeID",
					Token: Token{
						TokenType: "Bearer",
					},
				},
			},
			Source:   true,
			Position: module.Left,
		},
		{
			Name:     "success",
			Label:    "Success",
			Source:   false,
			Position: module.Right,
			Message:  CalendarGetEventSuccess{},
		}}

	if c.settings.EnableErrorPort {
		ports = append(ports, module.NodePort{
			Position: module.Bottom,
			Name:     "error",
			Label:    "Error",
			Source:   false,
			Message:  CalendarGetEventsError{},
		})
	}
	return ports
}

func (c *CalendarGetEvents) Instance() module.Component {
	return &CalendarGetEvents{}
}

var _ module.Component = (*CalendarGetEvents)(nil)

func init() {
	registry.Register(&CalendarGetEvents{})
}
