package template

import (
	"bytes"
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"html/template"
	"time"
)

const (
	EngineComponent = "template-engine"
	EngineInPort    = "input"
	EngineOutPort   = "output"
	EngineErrorPort = "error"
)

type Context any
type RenderContext any

type Template struct {
	Name    string `json:"name" required:"true" title:"File name" Description:"e.g. footer.tmpl" propertyOrder:"1"`
	Content string `json:"content" required:"true" title:"Template" format:"textarea" propertyOrder:"2"`
}

type Settings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen during mail send, error port will emit an error message" propertyOrder:"1" tab:"Settings"`

	Templates []Template `json:"templates,omitempty" required:"true" title:"Templates" minItems:"1" uniqueItems:"true" propertyOrder:"1" tab:"Templates"`
	Partials  []Template `json:"partials,omitempty" required:"true" title:"Partials" description:"All partials being loaded with each template" minItems:"0" uniqueItems:"true" propertyOrder:"1" tab:"Partials"`
}

type Error struct {
	Context Context `json:"context"`
	Error   string  `json:"error"`
}

type Input struct {
	Context       Context       `json:"context" configurable:"true" required:"true" title:"Context" description:"Arbitrary message to be send alongside with rendered content" propertyOrder:"1"`
	RenderContext RenderContext `json:"renderContext" configurable:"true" required:"true" title:"Render context" description:"Data being used to render the template" propertyOrder:"2"`
	Template      string        `json:"template" required:"true" title:"Template" description:"Template to render" propertyOrder:"3"`
}

type Output struct {
	Context       Context       `json:"context"`
	RenderContext RenderContext `json:"renderContext"`
	Content       string        `json:"content"`
}

type Engine struct {
	templateSet map[string]*template.Template
	settings    Settings
}

var defaultEngineSettings = Settings{
	Templates: []Template{
		{
			Name: "home.html",
			Content: `{{template "layout.html" .}}
{{define "title"}}Welcome.{{end}}
{{define "content"}}
Welcome
{{end}}`,
		},
		{
			Name: "page1.html",
			Content: `{{template "layout.html" .}}
{{define "title"}} Page one.{{end}}
{{define "content"}}
I'm page 1
{{end}}`,
		},
		{
			Name: "page2.html",
			Content: `{{template "layout.html" .}}
{{define "title"}} Page 2 title {{end}}
{{define "content"}}
I'm page 2
{{end}}`,
		},
	},
	Partials: []Template{
		{
			Name: "layout.html",
			Content: `<!DOCTYPE html>
<html lang="en">
<head>
<title>{{block "title" .}}{{end}}</title>
</head>
<body>
{{block "nav" . }}{{end}}
<div style="padding:20px">
{{block "content" .}}{{end}}
</div>
{{block "footer" .}}{{end}}
</body>
</html>`,
		},
		{
			Name: "footer.html",
			Content: `{{define "footer"}}
<hr/>
<div style="text-align:center">
 <p>&copy; {{now.UTC.Year}}</p>
 <p>{{builtWith}}</p>
</div>
{{end}}`,
		},
		{
			Name: "nav.html",
			Content: `{{define "nav"}}
<div>
 <a href="/">Home page</a>
 <a href="/page1">Page1</a>
 <a href="/page2">Page2</a>
</div>
{{end}}`,
		},
	},
}

func (h *Engine) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        EngineComponent,
		Description: "Template engine",
		Info:        "Renders templates using html/template standard package",
		Tags:        []string{"html", "template", "engine"},
	}
}

func (h *Engine) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	switch port {
	case module.SettingsPort:
		// compile template
		in, ok := msg.(Settings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}

		h.settings = in
		ts := map[string]*template.Template{}

		funcMap := template.FuncMap{
			"now": time.Now,
			"builtWith": func() template.HTML {
				return `<a href="https://tinysystems.io?from=builtwith" target="_blank">Built with Tiny Systems</a>`
			},
		}

		for _, t := range in.Templates {
			tmpl, err := template.New(t.Name).Funcs(funcMap).Parse(t.Content)
			if err != nil {
				h.error(ctx, err, in, handler)
				return err
			}
			for _, p := range in.Partials {
				_, err = tmpl.New(p.Name).Parse(p.Content)
				if err != nil {
					h.error(ctx, err, in, handler)
					return err
				}
			}
			ts[t.Name] = tmpl
		}

		h.templateSet = ts
	case EngineInPort:

		in, ok := msg.(Input)
		if !ok {
			return fmt.Errorf("invalid input")
		}
		if h.templateSet == nil {
			return fmt.Errorf("template set not loaded")
		}

		buf := &bytes.Buffer{}
		t, ok := h.templateSet[in.Template]
		if !ok {
			err := fmt.Errorf("template not found")
			h.error(ctx, err, in.Context, handler)
			return err
		}

		if err := t.ExecuteTemplate(buf, in.Template, in.RenderContext); err != nil {
			if h.settings.EnableErrorPort {
				_ = handler(ctx, EngineErrorPort, Error{
					Context: in.Context,
					Error:   err.Error(),
				})
			}
			return err
		}

		return handler(ctx, EngineOutPort, Output{
			Content:       buf.String(),
			RenderContext: in.RenderContext,
			Context:       in.Context,
		})

	default:
		return fmt.Errorf("port %s is not supoprted", port)
	}
	return nil
}

func (h *Engine) error(ctx context.Context, err error, contextMsg Context, handler module.Handler) {
	if h.settings.EnableErrorPort {
		_ = handler(ctx, EngineErrorPort, Error{
			Context: contextMsg,
			Error:   err.Error(),
		})
	}
}
func (h *Engine) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:          EngineInPort,
			Label:         "Input",
			Position:      module.Left,
			Source:        true,
			Configuration: Input{},
		},
		{
			Name:          EngineOutPort,
			Position:      module.Right,
			Label:         "Output",
			Configuration: Output{},
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: h.settings,
		},
	}
	if h.settings.EnableErrorPort {
		ports = append(ports, module.NodePort{
			Position:      module.Bottom,
			Name:          EngineErrorPort,
			Label:         "Error",
			Source:        false,
			Configuration: Error{},
		})
	}
	return ports
}

func (h *Engine) Instance() module.Component {
	return &Engine{
		settings: defaultEngineSettings,
	}
}

var _ module.Component = (*Engine)(nil)

func init() {
	registry.Register(&Engine{})
}
