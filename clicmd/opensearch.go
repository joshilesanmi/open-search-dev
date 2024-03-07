package clicmd

import (
	"context"
	"os"

	"github.com/joshilesanmi/open-search-dev/search"
	"github.com/joshilesanmi/open-search-dev/search/opensearch"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

func makeOpenSearchClient(endpoint string, logger zerolog.Logger, opts ...opensearch.OpenSearchOption) (search.SearchEngine, error) {
	return opensearch.NewOpenSearch(endpoint, logger, opts...)
}

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

func OpenSearch() *cli.Command {
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()

	createIndex := &cli.Command{
		Name:  "create-index",
		Usage: "create an open search index with its settings",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "index-name",
				Usage:    "index name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "endpoint",
				Usage:    "cluster endpoint (url)",
				Required: true,
			},
		},
		Action: createIndex(logger),
	}

	return &cli.Command{
		Name:  "opensearch",
		Usage: "provides open commands",
		Subcommands: []*cli.Command{
			createIndex,
		},
	}
}

func createIndex(logger zerolog.Logger) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		indexName := c.String("index-name")
		endpoint := c.String("endpoint")

		client, err := makeOpenSearchClient(endpoint, logger)
		if err != nil {
			return err
		}
		return client.CreateIndex(context.Background(), indexName, indexConfig)
	}
}
