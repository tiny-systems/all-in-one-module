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
	EngineComponent    = "template_engine"
	EngineRequestPort  = "request"
	EngineResponsePort = "response"
	EngineErrorPort    = "error"
)

type Context any
type RenderContext any

type Template struct {
	Name    string `json:"name,omitempty" required:"true" title:"File name" Description:"e.g. footer.tmpl"`
	Content string `json:"content,omitempty" required:"true" title:"Template" format:"textarea"`
}

type Settings struct {
	EnableErrorPort bool `json:"enableErrorPort,omitempty" required:"true" title:"Enable Error Port" description:"If error happen during mail send, error port will emit an error message" tab:"Settings"`

	Templates []Template `json:"templates,omitempty" required:"true" title:"Templates" minItems:"1" uniqueItems:"true" tab:"Templates"`
	Partials  []Template `json:"partials,omitempty" required:"true" title:"Partials" description:"All partials being loaded with each template" minItems:"0" uniqueItems:"true" tab:"Partials"`
}

type Error struct {
	Input Input  `json:"input"`
	Error string `json:"error"`
}

type Input struct {
	Context       Context       `json:"context,omitempty" configurable:"true" title:"Context" description:"Arbitrary message to be send alongside with rendered content"`
	RenderContext RenderContext `json:"renderContext,omitempty" configurable:"true" title:"Render context" description:"Data being used to render the template"`
	Template      string        `json:"template,omitempty" required:"true" title:"Template" description:"Template to render"`
}

type Output struct {
	Input   Input  `json:"input"`
	Content string `json:"content"`
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
		Info:        "Renders templates using go's html/template standard package",
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
				return err
			}
			for _, p := range in.Partials {
				_, err = tmpl.New(p.Name).Parse(p.Content)
				if err != nil {

					return err
				}
			}
			ts[t.Name] = tmpl
		}

		h.templateSet = ts
	case EngineRequestPort:

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
			if !h.settings.EnableErrorPort {
				return err
			}
			return handler(ctx, EngineErrorPort, Error{
				Input: in,
				Error: err.Error(),
			})
		}

		err := t.ExecuteTemplate(buf, in.Template, in.RenderContext)
		if err != nil {
			if !h.settings.EnableErrorPort {
				return err
			}
			return handler(ctx, EngineErrorPort, Error{
				Input: in,
				Error: err.Error(),
			})
		}

		return handler(ctx, EngineResponsePort, Output{
			Content: buf.String(),
			Input:   in,
		})

	default:
		return fmt.Errorf("port %s is not supoprted", port)
	}
	return nil
}

func (h *Engine) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:          EngineRequestPort,
			Label:         "Request",
			Position:      module.Left,
			Source:        true,
			Configuration: Input{},
		},
		{
			Name:          EngineResponsePort,
			Position:      module.Right,
			Label:         "Response",
			Configuration: Output{},
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: h.settings,
		},
	}
	if !h.settings.EnableErrorPort {
		return ports
	}
	return append(ports, module.Port{
		Position:      module.Bottom,
		Name:          EngineErrorPort,
		Label:         "Error",
		Source:        false,
		Configuration: Error{},
	})
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
