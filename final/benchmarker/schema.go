package main

import (
	"encoding/json"
	"github.com/xeipuuv/gojsonschema"
)

func MustSchema(doc string) *gojsonschema.JsonSchemaDocument {
	var value map[string]interface{}

	err := json.Unmarshal([]byte(doc), &value)
	if err != nil {
		panic(err)
	}

	schema, err := gojsonschema.NewJsonSchemaDocument(value)
	if err != nil {
		panic(err)
	}

	return schema
}

var (
	adSchema = MustSchema(`{
		"type": "object",
		"properties": {
			"slot": { "type": "string" },
			"id": { "type": "string" },
			"title": { "type": "string" },
			"type": {
				"oneOf": [
					{ "type": "string" },
					{ "type": "null" }
				]
			},
			"advertiser": { "type": "string" },
			"destination": { "type": "string" },
			"impressions": { "type": "number", "minimum": 0 },
			"asset": { "type": "string" },
			"counter": { "type": "string" },
			"redirect": { "type": "string" }
		},
		"required": [
			"slot",
			"id",
			"title",
			"type",
			"advertiser",
			"destination",
			"impressions",
			"asset",
			"counter",
			"redirect"
		],
		"additionalProperties": true
	}`)

	myReportSchema = MustSchema(`{
		"type": "object",
		"patternProperties": {
			"^[0-9]+$": {
				"type": "object",
				"properties": {
					"ad": {
						"type": "object",
						"properties": {
							"slot": { "type": "string" },
							"id": { "type": "string" },
							"title": { "type": "string" },
							"type": {
								"oneOf": [
									{ "type": "string" },
									{ "type": "null" }
								]
							},
							"advertiser": { "type": "string" },
							"destination": { "type": "string" }
						},
						"required": [
							"slot",
							"id",
							"title",
							"type",
							"advertiser",
							"destination"
						],
						"additionalProperties": true
					},
					"breakdonw": {
						"type": "object",
						"patternProperties": {
							"^.+$": {
								"type": "object",
								"patternProperties": {
									"^.+$": {
										"type": "integer",
										"minimum": 0
									}
								}
							}
						},
						"required": ["agents", "gender", "generations"]
					},
					"impressions": { "type": "number", "minimum": 0 },
					"clicks": { "type": "number", "minimum": 0 }
				},
				"additionalProperties": true
			}
		},
		"additionalProperties": false
	}`)

	finalReportSchema = MustSchema(`{
		"type": "object",
		"patternProperties": {
			"^[0-9]+$": {
				"type": "object",
				"properties": {
					"ad": {
						"type": "object",
						"properties": {
							"slot": { "type": "string" },
							"id": { "type": "string" },
							"title": { "type": "string" },
							"type": {
								"oneOf": [
									{ "type": "string" },
									{ "type": "null" }
								]
							},
							"advertiser": { "type": "string" },
							"destination": { "type": "string" }
						},
						"required": [
							"slot",
							"id",
							"title",
							"type",
							"advertiser",
							"destination"
						],
						"additionalProperties": true
					},
					"impressions": { "type": "number", "minimum": 0 },
					"clicks": { "type": "number", "minimum": 0 }
				},
				"additionalProperties": true
			}
		},
		"additionalProperties": false
	}`)
)
