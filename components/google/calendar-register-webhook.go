package google

import (
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const CalendarRegisterWebhookComponent = "google_calendar_register_webhook"

type CalendarRegisterChannel struct {
	ID          string `json:"id" required:"true" title:"ID" description:"A UUID or similar unique string that identifies this channel."`
	Type        string `json:"type" required:"true" title:"Type" enum:"web_hook" enumTitles:"Webhook" description:"The type of delivery mechanism used for this channel. Valid values are \"web_hook\" (or \"webhook\"). Both values refer to a channel where Http requests are used to deliver messages."`
	Address     string `json:"address" required:"true" title:"Address" description:"The address where notifications are delivered for this channel."`
	Expiration  int64  `json:"expiration" title:"Expiration" description:"Date and time of notification channel expiration, expressed as a Unix timestamp, in milliseconds."`
	ResourceId  string `json:"resourceId" title:"ResourceID" description:"An opaque ID that identifies the resource being watched on this channel. Stable across different API versions."`
	ResourceUri string `json:"resourceUri" title:"ResourceURI" description:"A version-specific identifier for the watched resource."`
	Token       string `json:"token" title:"Auth Token" description:"An arbitrary string delivered to the target address with each notification delivered over this channel."`
}

type CalendarRegisterWebhookSettings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If request may fail, error port will emit an error message"`
}

type CalendarRegisterWebhookContext any

type CalendarRegisterWebhookRequest struct {
	Context  CalendarRegisterWebhookContext         `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be send further" propertyOrder:"1"`
	Calendar CalendarRegisterWebhookRequestCalendar `json:"calendar" required:"true" title:"Calendar" propertyOrder:"2"`
	Channel  CalendarRegisterChannel                `json:"channel" required:"true" title:"Channel" propertyOrder:"3"`
	Token    Token                                  `json:"token" required:"true" title:"Token" propertyOrder:"4"`
}

type CalendarRegisterWebhookRequestCalendar struct {
	ID string `json:"id" required:"true" title:"Calendar ID" description:"Google Calendar ID to be watched"`
}

type CalendarRegisterWebhookSuccess struct {
	Context CalendarRegisterWebhookContext `json:"context"`
	Request CalendarRegisterWebhookRequest `json:"request"`
}

type CalendarRegisterWebhookError struct {
	Context CalendarRegisterWebhookContext `json:"context"`
	Request CalendarRegisterWebhookRequest `json:"request"`
	Error   string                         `json:"error"`
}

type CalendarRegisterWebhook struct {
	settings CalendarRegisterWebhookSettings
}

func (h *CalendarRegisterWebhook) Instance() module.Component {
	return &CalendarRegisterWebhook{
		settings: CalendarRegisterWebhookSettings{},
	}
}

func (h *CalendarRegisterWebhook) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        CalendarRegisterWebhookComponent,
		Description: "Register Google Calendar Webhook",
		Info:        "Register calendar webhook",
		Tags:        []string{"Google", "Calendar"},
	}
}

func (h *CalendarRegisterWebhook) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(CalendarRegisterWebhookSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		h.settings = in
		return nil
	}

	if port != "request" {
		return fmt.Errorf("unknown port %s", port)
	}

	return nil
}

func (h *CalendarRegisterWebhook) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: CalendarRegisterWebhookSettings{},
			Source:        true,
		},
		{
			Name:  "request",
			Label: "Request",
			Configuration: CalendarRegisterWebhookRequest{
				Channel: CalendarRegisterChannel{
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
			Name:          "success",
			Label:         "Success",
			Source:        false,
			Position:      module.Right,
			Configuration: CalendarRegisterWebhookSuccess{},
		},
	}
	if h.settings.EnableErrorPort {
		ports = append(ports, module.NodePort{
			Position:      module.Bottom,
			Name:          "error",
			Label:         "Error",
			Source:        false,
			Configuration: CalendarRegisterWebhookError{},
		})
	}

	return ports
}

var _ module.Component = (*CalendarRegisterWebhook)(nil)

func init() {
	registry.Register(&CalendarRegisterWebhook{})
}
