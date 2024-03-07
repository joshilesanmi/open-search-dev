package main

import (
	"context"
	"log"
	"os"

	"github.com/joshilesanmi/open-search-dev/search"
	"github.com/joshilesanmi/open-search-dev/search/opensearch"
	"github.com/rs/zerolog"
)

var indexConfig = map[string]interface{}{
	"settings": map[string]interface{}{
		"index": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 1,
		},
	},
	"mappings": map[string]interface{}{
		"dynamic_templates": []interface{}{
			map[string]interface{}{
				"boolean_fields": map[string]interface{}{
					"match":   "field_*_boolean",
					"mapping": map[string]interface{}{"type": "boolean"},
				},
			},
			map[string]interface{}{
				"int_fields": map[string]interface{}{
					"match":   "field_*_int",
					"mapping": map[string]interface{}{"type": "integer"},
				},
			},
			map[string]interface{}{
				"string_fields": map[string]interface{}{
					"match":   "field_*_string",
					"mapping": map[string]interface{}{"type": "text"},
				},
			},
			map[string]interface{}{
				"date_fields": map[string]interface{}{
					"match":   "field_*_datetime",
					"mapping": map[string]interface{}{"type": "date"},
				},
			},
			map[string]interface{}{
				"string_list_fields": map[string]interface{}{
					"match":   "field_*_string_list",
					"mapping": map[string]interface{}{"type": "keyword"},
				},
			},
		},
		"properties": map[string]interface{}{
			"id":                 map[string]interface{}{"type": "keyword"},
			"instance_id":        map[string]interface{}{"type": "keyword"},
			"name":               map[string]interface{}{"type": "text"},
			"assigned_sales_rep": map[string]interface{}{"type": "keyword"},
			"created_at":         map[string]interface{}{"type": "date"},
			"updated_at":         map[string]interface{}{"type": "date"},
			"custom_fields": map[string]interface{}{
				"type":    "object",
				"dynamic": true,
			},
		},
	},
}

func main() {
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()

	endpoint := "http://neodxp-opensearch-dev.justrelate.io"
	ctx := context.Background()

	client, err := opensearch.NewOpenSearch(endpoint, logger)
	if err != nil {
		log.Fatal(err)
	}

	err = client.CreateIndex(ctx, "neodxp-dev", indexConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = client.PutDocument(ctx, "instance-id", "neodxp-dev", "person", "person-id-1", search.Document{
		"entity_type":        "person",
		"name":               "John Doe",
		"assigned_sales_rep": "eyan@test.com",
		"field_1_string":     "random text",
		"field_2_string":     "another random text",
	})
	if err != nil {
		log.Fatal(err)
	}

	err = client.PutDocument(ctx, "instance-id", "neodxp-dev", "person", "person-id-2", search.Document{
		"entity_type":        "person",
		"name":               "Jane Doe",
		"assigned_sales_rep": "eyan@test.com",
		"field_1_string":     "random text",
		"field_2_string":     "another random text",
	})
	if err != nil {
		log.Fatal(err)
	}
}
