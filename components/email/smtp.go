package email

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"github.com/wneessen/go-mail"
)

const (
	SendEmailComponent = "send_email"
	PortSuccess        = "success"
	PortError          = "error"
	PortIn             = "in"
)

type SenderSettings struct {
	EnableErrorPort   bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen during mail send, error port will emit an error message"`
	EnableSuccessPort bool `json:"enableSuccessPort" required:"true" title:"Enable Success port"`
}

type Recipient struct {
	Name  string `json:"name" title:"Name" colSpan:"col-span-6"`
	Email string `json:"email" required:"true" title:"Email settings" format:"email" minLength:"1" colSpan:"col-span-6"`
}

type SendEmailContext any

type SendEmail struct {
	Context SendEmailContext `json:"context" configurable:"true" title:"Context" propertyOrder:"1"`
	Email   EmailConfig      `json:"email" required:"true" title:"Email"`
}

type EmailConfig struct {
	ContentType string      `json:"contentType" required:"true" title:"Content type" enum:"text/plain,text/html,application/octet-stream" propertyOrder:"2"`
	From        string      `json:"from" title:"From" propertyOrder:"3"`
	To          []Recipient `json:"to,omitempty" required:"true" description:"List of recipients" title:"To" uniqueItems:"true" minItems:"1" propertyOrder:"4"`

	Body         string             `json:"body" title:"Email body" format:"textarea" propertyOrder:"5"`
	Subject      string             `json:"subject" title:"Subject" propertyOrder:"6"`
	SmtpSettings SmtpServerSettings `json:"smtpSettings" required:"true" title:"SMTP Settings" propertyOrder:"7"`
}

type SmtpServerSettings struct {
	Host     string `json:"host" required:"true" minLength:"1" title:"SMTP Host" propertyOrder:"1"`
	Port     int    `json:"port" required:"true" title:"SMTP Port" propertyOrder:"2"`
	Username string `json:"username" title:"SMTP username" required:"true" propertyOrder:"3"`
	Password string `json:"password" title:"SMTP password" required:"true" propertyOrder:"4"`
	Test     bool   `json:"test" format:"button" title:"Test connection" required:"true" propertyOrder:"5"`
}

type SendMessageSuccess struct {
	Context   SendEmailContext `json:"context"`
	MessageID string           `json:"messageID"`
	Email     EmailConfig      `json:"sent"`
}

type SendMessageError struct {
	Context   SendEmailContext `json:"context"`
	Error     string           `json:"error"`
	Email     EmailConfig      `json:"sent"`
	MessageID string           `json:"messageID"`
}

var SenderDefaultSettings = SenderSettings{}

type SmtpSender struct {
	settings SenderSettings
}

func (t *SmtpSender) Instance() module.Component {
	return &SmtpSender{
		settings: SenderDefaultSettings,
	}
}

func (t *SmtpSender) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        SendEmailComponent,
		Description: "SMTP Email sender",
		Info:        "Sends email using SMTP protocol",
		Tags:        []string{"Email", "SMTP"},
	}
}

func (t *SmtpSender) Handle(ctx context.Context, responseHandler module.Handler, port string, msg interface{}) error {
	if port == module.SettingsPort {
		in, ok := msg.(SenderSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		t.settings = in
		return nil
	}

	sendMsg, ok := msg.(SendEmail)
	if !ok {
		return fmt.Errorf("invalid message")
	}

	messageID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	client, err := mail.NewClient(sendMsg.Email.SmtpSettings.Host, mail.WithPort(sendMsg.Email.SmtpSettings.Port), mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithUsername(sendMsg.Email.SmtpSettings.Username), mail.WithPassword(sendMsg.Email.SmtpSettings.Password))

	if err != nil {
		if t.settings.EnableErrorPort {
			return responseHandler(PortError, SendMessageError{
				Context:   sendMsg.Context,
				Email:     sendMsg.Email,
				Error:     err.Error(),
				MessageID: messageID.String(),
			})
		}
		return err
	}

	err = client.DialWithContext(ctx)
	if err != nil {
		if t.settings.EnableErrorPort {
			return responseHandler(PortError, SendMessageError{
				Context:   sendMsg.Context,
				Email:     sendMsg.Email,
				Error:     err.Error(),
				MessageID: messageID.String(),
			})
		}
		return err
	}

	m := mail.NewMsg()
	_ = m.From(sendMsg.Email.From)
	for _, t := range sendMsg.Email.To {
		_ = m.To(fmt.Sprintf("%s <%s>", t.Name, t.Email))
	}

	m.Subject(sendMsg.Email.Subject)
	m.SetBodyString(mail.ContentType(sendMsg.Email.ContentType), sendMsg.Email.Body)

	defer func() {
		_ = client.Close()
	}()

	err = client.Send(m)
	if err != nil {
		if t.settings.EnableErrorPort {
			return responseHandler(PortError, SendMessageError{
				Context:   sendMsg.Context,
				Email:     sendMsg.Email,
				Error:     err.Error(),
				MessageID: messageID.String(),
			})
		}
		return err
	}

	if err == nil && t.settings.EnableSuccessPort {
		return responseHandler(PortSuccess, SendMessageSuccess{
			Context:   sendMsg.Context,
			Email:     sendMsg.Email,
			MessageID: messageID.String(),
		})
	}
	// send email here
	return err
}

func (t *SmtpSender) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: SenderSettings{},
		},
		{
			Name:   PortIn,
			Label:  "In",
			Source: true,
			Configuration: SendEmail{
				Email: EmailConfig{
					Body:        "Email text",
					ContentType: "text/html",
					To: []Recipient{
						{
							Name:  "John Doe",
							Email: "johndoe@example.com",
						},
					},
					SmtpSettings: SmtpServerSettings{
						Host: "smtp.domain.com",
						Port: 587,
					},
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
			Configuration: SendMessageSuccess{},
		})
	}

	if t.settings.EnableErrorPort {
		ports = append(ports, module.NodePort{
			Position:      module.Bottom,
			Name:          PortError,
			Label:         "Error",
			Source:        false,
			Configuration: SendMessageError{},
		})
	}

	return ports
}

var _ module.Component = (*SmtpSender)(nil)

func init() {
	registry.Register(&SmtpSender{})
}
