package slack

import (
	"context"
	"fmt"
	"github.com/slack-go/slack"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	SendSlackChannelComponent = "send_slack_channel"
	PortResponse              = "response"
	PortError                 = "error"
	PortRequest               = "request"
)

type ChannelSenderSettings struct {
	EnableSuccessPort bool `json:"enableSuccessPort" required:"true" title:"Enable Success port" description:""`
	EnableErrorPort   bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen during mail send, error port will emit an error message"`
}

type SendSlackChannelContext any

type Message struct {
	ChannelID  string `json:"channelID" required:"true" minLength:"1" title:"ChannelID" description:""`
	SlackToken string `json:"slackToken" required:"true" minLength:"1" title:"Slack token" description:"Bot User OAuth Token"`
	Text       string `json:"text" required:"true" minLength:"1" title:"Message text" format:"textarea"`
}

type SendChannelRequest struct {
	Context SendSlackChannelContext `json:"context" configurable:"true" title:"Context"`
	Message Message                 `json:"slack_message" required:"true" title:"Slack Message"`
}

type SendSlackChannelSuccess struct {
	Request SendChannelRequest `json:"request"`
	Sent    Message            `json:"sent"`
}

type SendSlackChannelError struct {
	Request SendChannelRequest `json:"request"`
	Error   string             `json:"error"`
}

var SenderDefaultSettings = ChannelSenderSettings{}

type ChannelSender struct {
	settings ChannelSenderSettings
}

func (t *ChannelSender) Instance() module.Component {
	return &ChannelSender{
		settings: SenderDefaultSettings,
	}
}

func (t *ChannelSender) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        SendSlackChannelComponent,
		Description: "Slack channel sender",
		Info:        "Sends messages to slack channel",
		Tags:        []string{"Slack", "IM"},
	}
}

func (t *ChannelSender) Handle(ctx context.Context, responseHandler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(ChannelSenderSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
		return nil
	}

	in, ok := msg.(SendChannelRequest)
	if !ok {
		return fmt.Errorf("invalid message")
	}

	client := slack.New(in.Message.SlackToken)
	_, _, _, err := client.SendMessageContext(ctx, in.Message.ChannelID, slack.MsgOptionText(in.Message.Text, true))

	if err != nil {
		if !t.settings.EnableErrorPort {
			return err
		}
		return responseHandler(ctx, PortError, SendSlackChannelError{
			Request: in,
			Error:   err.Error(),
		})
	}

	if err == nil && t.settings.EnableSuccessPort {
		return responseHandler(ctx, PortResponse, SendSlackChannelSuccess{
			Request: in,
			Sent:    in.Message,
		})
	}
	// send email here
	return err
}

func (t *ChannelSender) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: ChannelSenderSettings{},
		},
		{
			Name:   PortRequest,
			Label:  "Request",
			Source: true,
			Configuration: SendChannelRequest{
				Message: Message{
					Text: "Message to send",
				},
			},
			Position: module.Left,
		},
	}
	if t.settings.EnableSuccessPort {
		ports = append(ports, module.Port{
			Position:      module.Right,
			Name:          PortResponse,
			Label:         "Response",
			Source:        false,
			Configuration: SendSlackChannelSuccess{},
		})
	}

	if !t.settings.EnableErrorPort {
		return ports
	}
	return append(ports, module.Port{
		Position:      module.Bottom,
		Name:          PortRequest,
		Label:         "Error",
		Source:        false,
		Configuration: SendSlackChannelError{},
	})
}

var _ module.Component = (*ChannelSender)(nil)

func init() {
	registry.Register(&ChannelSender{})
}
