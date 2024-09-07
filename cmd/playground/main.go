package main

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/tiny-systems/main/components/http"
	"github.com/tiny-systems/module/pkg/schema"
)

func main() {
	testRouter()
}

func testRouter() {

	settingsSchema, err := do(http.ClientRequest{})
	if err != nil {
		log.Fatal().Err(err).Msg("settings")
	}

	fmt.Println(string(settingsSchema))
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
	return marshaled, err
}
