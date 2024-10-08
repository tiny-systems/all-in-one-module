package common

import (
	"context"
	"fmt"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/swaggest/jsonschema-go"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"strings"
)

const (
	MixerOutputPort string = "output"
)

type Mixer struct {
	settings MixerSettings
	//
	inputs cmap.ConcurrentMap[string, interface{}]

	output MixerOutput
}

type MixerInputContext any

type MixerInput struct {
	Context      MixerInputContext `json:"context" configurable:"true" required:"true" title:"Context" description:"Arbitrary message"`
	nameOverride string
}

// Process post-processing schema
func (m MixerInput) Process(s *jsonschema.Schema) {

	d := s.ExtraProperties["$defs"]
	defs, ok := d.(map[string]jsonschema.Schema)
	if !ok {
		return
	}

	defName := getDefinitionName(m.nameOverride)
	defs[defName] = defs["Mixerinputcontext"]

	// rename root schema name
	inputName := fmt.Sprintf("Mixerinput%s", strings.ToTitle(m.nameOverride))

	input, ok := defs["Mixerinput"]
	if !ok {
		return
	}

	defs[inputName] = input
	delete(defs, "Mixerinput")

	ctx, ok := input.Properties["context"]
	if !ok {
		return
	}
	if ctx.TypeObject == nil {
		return
	}

	ref := fmt.Sprintf("#/$defs/%s", inputName)
	s.Ref = &ref

	defRef := fmt.Sprintf("#/$defs/%s", defName)
	ctx.TypeObject.Ref = &defRef

	delete(defs, "Mixerinputcontext")
}

type MixerOutput struct {
	inputNames []string
}

func (m MixerOutput) Process(s *jsonschema.Schema) {
	d := s.ExtraProperties["$defs"]

	defs, ok := d.(map[string]jsonschema.Schema)
	if !ok {
		return
	}

	var output = jsonschema.Schema{}

	output.WithType(jsonschema.Object.Type())
	output.WithExtraPropertiesItem("path", "$")
	//
	//
	for _, input := range m.inputNames {
		defName := getDefinitionName(input)
		propName := getPropName(input)

		def := jsonschema.Schema{}
		def.WithDescription(fmt.Sprintf("Arbitrary message %s", input))
		def.WithExtraPropertiesItem("path", fmt.Sprintf("$.%s", propName))
		defs[defName] = def

		ref := jsonschema.Schema{}
		ref.WithRef(fmt.Sprintf("#/$defs/%s", defName))
		output.WithPropertiesItem(propName, ref.ToSchemaOrBool())
	}

	defs["Mixeroutput"] = output
	return
}

type MixerSettings struct {
	Inputs []string `json:"inputs,omitempty" required:"true" title:"Inputs" minItems:"1" uniqueItems:"true"`
}

func (m *Mixer) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        "mixer",
		Description: "Mixer",
		Info:        "Mixes latest values on ports into single message",
		Tags:        []string{"SDK"},
	}
}

func (m *Mixer) Handle(ctx context.Context, output module.Handler, port string, msg interface{}) error {

	switch {
	case port == module.SettingsPort:
		in, ok := msg.(MixerSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		m.settings = in
		// reset state after new settings
		m.inputs.Clear()
		m.output.inputNames = in.Inputs
		return nil

	case m.hasInput(port):
		in, ok := msg.(MixerInput)
		if !ok {
			return fmt.Errorf("invalid message type: %T", msg)
		}

		m.inputs.Set(getPropName(port), in.Context)

		return m.send(ctx, output)
	default:
		return fmt.Errorf("unknown port: %s", port)
	}
}

func (m *Mixer) send(ctx context.Context, output module.Handler) error {
	return output(ctx, MixerOutputPort, m.inputs)
}

func (m *Mixer) hasInput(name string) bool {
	for _, i := range m.settings.Inputs {
		if i == name {
			return true
		}
	}
	return false
}

func (m *Mixer) Ports() []module.Port {
	//
	ports := []module.Port{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: m.settings,
		},
		{
			Name:          MixerOutputPort,
			Label:         "Output",
			Configuration: m.output,
			Position:      module.Right,
		},
	}

	//
	for _, input := range m.settings.Inputs {
		ports = append(ports, module.Port{
			Name:   input,
			Label:  strings.ToUpper(input),
			Source: true,
			Configuration: MixerInput{
				nameOverride: input,
			},
			Position: module.Left,
		})
	}

	return ports
}

func (m *Mixer) Instance() module.Component {
	return &Mixer{
		settings: MixerSettings{Inputs: []string{"A", "B"}},
		inputs:   cmap.New[interface{}](),
	}
}

func getDefinitionName(input string) string {
	return fmt.Sprintf("MixerContext%s", strings.ToTitle(input))
}

func getPropName(input string) string {
	return fmt.Sprintf("context%s", strings.ToTitle(input))
}

var _ module.Component = (*Mixer)(nil)

func init() {
	registry.Register(&Mixer{})
}
