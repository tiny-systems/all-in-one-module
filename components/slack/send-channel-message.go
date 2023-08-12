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
	PortSuccess               = "success"
	PortError                 = "error"
	PortIn                    = "in"
)

type ChannelSenderSettings struct {
	EnableSuccessPort bool `json:"enableSuccessPort" required:"true" title:"Enable Success port" description:"" propertyOrder:"1"`
	EnableErrorPort   bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen during mail send, error port will emit an error message" propertyOrder:"2"`
}

type SendSlackChannelContext any

type Message struct {
	ChannelID  string `json:"channelID" required:"true" minLength:"1" title:"ChannelID" description:"" propertyOrder:"1"`
	SlackToken string `json:"slackToken" required:"true" minLength:"1" title:"Slack token" description:"Bot User OAuth Token" propertyOrder:"2"`
	Text       string `json:"text" required:"true" minLength:"1" title:"Message text" format:"textarea" propertyOrder:"3"`
}

type SendChannelRequest struct {
	Context SendSlackChannelContext `json:"context" configurable:"true" title:"Context" propertyOrder:"1"`
	Message Message                 `json:"slack_message" required:"true" title:"Slack Message" propertyOrder:"2"`
}

type SendSlackChannelSuccess struct {
	Context SendSlackChannelContext `json:"context"`
	Sent    Message                 `json:"sent"`
}

type SendSlackChannelError struct {
	Context SendSlackChannelContext `json:"context"`
	Error   string                  `json:"error"`
	Send    Message                 `json:"sent"`
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
		if t.settings.EnableErrorPort {
			return responseHandler(PortError, SendSlackChannelError{
				Context: in.Context,
				Send:    in.Message,
				Error:   err.Error(),
			})
		}
		return err
	}

	if err == nil && t.settings.EnableSuccessPort {
		return responseHandler(PortSuccess, SendSlackChannelSuccess{
			Context: in.Context,
			Sent:    in.Message,
		})
	}
	// send email here
	return err
}

func (t *ChannelSender) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Settings:      true,
			Configuration: ChannelSenderSettings{},
		},
		{
			Name:   PortIn,
			Label:  "In",
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
		ports = append(ports, module.NodePort{
			Position:      module.Right,
			Name:          PortSuccess,
			Label:         "Success",
			Source:        false,
			Configuration: SendSlackChannelSuccess{},
		})
	}

	if t.settings.EnableErrorPort {
		ports = append(ports, module.NodePort{
			Position:      module.Bottom,
			Name:          PortError,
			Label:         "Error",
			Source:        false,
			Configuration: SendSlackChannelError{},
		})
	}

	return ports
}

var _ module.Component = (*ChannelSender)(nil)

func init() {
	registry.Register(&ChannelSender{})
}
