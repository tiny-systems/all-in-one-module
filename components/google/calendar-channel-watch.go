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
	CalendarChannelWatchComponent    = "google_calendar_channel_watch"
	CalendarChannelWatchRequestPort  = "request"
	CalendarChannelWatchResponsePort = "response"
	CalendarChannelWatchErrorPort    = "error"
)

type CalendarWatchChannel struct {
	ID          string `json:"id" required:"true" title:"ID" description:"A UUID or similar unique string that identifies this channel."`
	Type        string `json:"type" required:"true" title:"Type" enum:"web_hook" enumTitles:"Webhook" description:"The type of delivery mechanism used for this channel. Valid values are \"web_hook\" (or \"webhook\"). Both values refer to a channel where Http requests are used to deliver messages."`
	Address     string `json:"address" required:"true" title:"Address" description:"The address where notifications are delivered for this channel."`
	Expiration  int64  `json:"expiration" title:"Expiration" description:"Date and time of notification channel expiration, expressed as a Unix timestamp, in milliseconds."`
	ResourceId  string `json:"resourceId" title:"ResourceID" description:"An opaque ID that identifies the resource being watched on this channel. Stable across different API versions."`
	ResourceUri string `json:"resourceUri" title:"ResourceURI" description:"A version-specific identifier for the watched resource."`
	Token       string `json:"token" title:"Auth Token" description:"An arbitrary string delivered to the target address with each notification delivered over this channel."`
}

type CalendarChannelWatchSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

type CalendarChannelWatchContext any

type CalendarChannelWatchRequest struct {
	Context  CalendarChannelWatchContext         `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be send further" propertyOrder:"1"`
	Calendar CalendarChannelWatchRequestCalendar `json:"calendar" required:"true" title:"Calendar" propertyOrder:"2"`
	Channel  CalendarWatchChannel                `json:"channel" required:"true" title:"Channel" propertyOrder:"3"`
	Token    Token                               `json:"token" required:"true" title:"Token" propertyOrder:"4"`
	Config   ClientConfig                        `json:"config" required:"true" title:"Client credentials" propertyOrder:"5"`
}

type CalendarChannelWatchRequestCalendar struct {
	ID string `json:"id" required:"true" title:"Calendar ID" description:"Google Calendar ID to be watched"`
}

type CalendarChannelWatchChannel struct {
	ID string `json:"id"`
}

type CalendarChannelWatchResponse struct {
	Request CalendarChannelWatchRequest `json:"request"`
	Channel CalendarChannelWatchChannel `json:"channel"`
}

type CalendarChannelWatchError struct {
	Request CalendarChannelWatchRequest `json:"request"`
	Error   string                      `json:"error"`
}

type CalendarChannelWatch struct {
	settings CalendarChannelWatchSettings
}

func (h *CalendarChannelWatch) Instance() module.Component {
	return &CalendarChannelWatch{
		settings: CalendarChannelWatchSettings{},
	}
}

func (h *CalendarChannelWatch) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        CalendarChannelWatchComponent,
		Description: "Watch calendar channel",
		Info:        "Register calendar watcher",
		Tags:        []string{"Google", "Calendar"},
	}
}

func (h *CalendarChannelWatch) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(CalendarChannelWatchSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		h.settings = in
		return nil
	}

	if port != CalendarChannelWatchRequestPort {
		return fmt.Errorf("unknown port %s", port)
	}

	req, ok := msg.(CalendarChannelWatchRequest)
	if !ok {
		return fmt.Errorf("invalid message")
	}

	ch, err := h.watch(ctx, req)
	if err != nil {
		if !h.settings.EnableErrorPort {
			return err
		}
		return handler(ctx, CalendarChannelWatchErrorPort, CalendarChannelWatchError{
			Request: req,
			Error:   err.Error(),
		})
	}

	return handler(ctx, CalendarChannelWatchResponsePort, CalendarChannelWatchResponse{
		Request: req,
		Channel: CalendarChannelWatchChannel{
			ID: ch.Id,
		},
	})
}

func (h *CalendarChannelWatch) watch(ctx context.Context, req CalendarChannelWatchRequest) (*calendar.Channel, error) {
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

	return srv.Events.Watch(req.Calendar.ID, &calendar.Channel{
		Type:       req.Channel.Type,
		Address:    req.Channel.Address,
		Token:      req.Channel.Token,
		Id:         req.Channel.ID,
		Expiration: req.Channel.Expiration,
	}).Do()
}

func (h *CalendarChannelWatch) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: CalendarChannelWatchSettings{},
			Source:        true,
		},
		{
			Name:  CalendarChannelWatchRequestPort,
			Label: "Request",
			Configuration: CalendarChannelWatchRequest{
				Channel: CalendarWatchChannel{
					Type: "web_hook",
				},
				Token: Token{
					TokenType: "Bearer",
				},
			},
			Source:   true,
			Position: module.Left,
		},
		{
			Name:          CalendarChannelWatchResponsePort,
			Label:         "Response",
			Source:        false,
			Position:      module.Right,
			Configuration: CalendarChannelWatchResponse{},
		},
	}
	if !h.settings.EnableErrorPort {
		return ports
	}
	return append(ports, module.Port{
		Name:          CalendarChannelWatchErrorPort,
		Label:         "Error",
		Source:        false,
		Position:      module.Bottom,
		Configuration: CalendarChannelWatchError{},
	})
}

var _ module.Component = (*CalendarChannelWatch)(nil)

func init() {
	registry.Register(&CalendarChannelWatch{})
}
