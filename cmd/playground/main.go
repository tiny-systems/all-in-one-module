package main

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/spyzhov/ajson"
	"github.com/tiny-systems/main/components/common"
	"github.com/tiny-systems/module/pkg/schema"
)

func testDebug() {

	fmt.Println("----SETTINGS------")

	settingsSchema, err := do(common.DebugSettings{})

	if err != nil {
		log.Fatal().Err(err).Msg("settings")
	}

	fmt.Println("---CONTROL--")
	controlSchema, err := do(common.DebugControl{
		Context: map[string]interface{}{
			"field": 1,
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("control")
	}

	fmt.Println("-----DEFINITIONS-----")

	fmt.Println("-----CONTROL-----")
	fmt.Println(string(controlSchema))

	fmt.Println()

	fmt.Println("-----CONTROL UPDATED-----")

	defs := make(map[string]*ajson.Node)

	err = schema.ParseSchema(settingsSchema, defs)
	if err != nil {
		log.Fatal().Err(err).Msg("parse")
	}

	updatedSchema, err := schema.UpdateWithConfigurableDefinitions(controlSchema, defs)
	if err != nil {
		log.Fatal().Err(err).Msg("update")
	}
	fmt.Println(string(updatedSchema))
	fmt.Println()
}

func main() {
	testDebug()
}

func testRouter() {

	fmt.Println("----SETTINGS------")

	settingsSchema, err := do(common.RouterSettings{})
	if err != nil {
		log.Fatal().Err(err).Msg("settings")
	}
	edgeSchema := `{"$ref":"#/$defs/Routerinmessage","$defs":{"Condition":{"required":["route","condition"],"properties":{"condition":{"title":"Condition","type":"boolean","propertyOrder":2},"route":{"$ref":"#/$defs/Routename","title":"Route","propertyOrder":1}},"type":"object","path":"$.conditions[0]"},"Routename":{"title":"Route","default":"A","enum":["A","B"],"type":"string","path":"$.conditions[0].route"},"Routercontext":{"title":"Context","description":"Arbitrary message to be routed","configurable":true,"path":"$.context","propertyOrder":1,"configure":false,"type":"object","properties":{"field_0_1":{"type":"string"}},"required":["field_0_1"]},"Routerinmessage":{"required":["context","conditions"],"properties":{"conditions":{"title":"Conditions","items":{"$ref":"#/$defs/Condition"},"minItems":1,"uniqueItems":true,"type":"array","propertyOrder":2},"context":{"$ref":"#/$defs/Routercontext"}},"type":"object","path":"$"}}}`

	conf := make(map[string]*ajson.Node)

	err = schema.ParseSchema(settingsSchema, conf)
	if err != nil {
		log.Fatal().Err(err).Msg("parse settingsSchema schema")
	}

	err = schema.ParseSchema([]byte(edgeSchema), conf)
	if err != nil {
		log.Fatal().Err(err).Msg("parse edgeSchema schema")
	}

	updated, err := schema.UpdateWithConfigurableDefinitions([]byte(`{"$ref":"#/$defs/Routerinmessage","$defs":{"Condition":{"required":["route","condition"],"properties":{"condition":{"title":"Condition","type":"boolean","propertyOrder":2},"route":{"$ref":"#/$defs/Routename","title":"Route","propertyOrder":1}},"type":"object","path":"$.conditions[0]"},"Routename":{"title":"Route","default":"A","enum":["A","B"],"type":"string","path":"$.conditions[0].route"},"Routercontext":{"title":"Context","description":"Arbitrary message to be routed","configurable":true,"path":"$.context","propertyOrder":1},"Routerinmessage":{"required":["context","conditions"],"properties":{"conditions":{"title":"Conditions","items":{"$ref":"#/$defs/Condition"},"minItems":1,"uniqueItems":true,"type":"array","propertyOrder":2},"context":{"$ref":"#/$defs/Routercontext"}},"type":"object","path":"$"}}}`), conf)
	if err != nil {
		log.Fatal().Err(err).Msg("update schema")
	}

	fmt.Println(string(updated))
}

func do(s interface{}) ([]byte, error) {

	schema, err := schema.CreateSchema(s)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create schema")
	}

	marshaled, err := json.MarshalIndent(schema, "", "   ")
	if err != nil {
		log.Fatal().Err(err).Msg("marshal error")
		return nil, err
	}
	fmt.Println(string(marshaled))
	return marshaled, err
}
