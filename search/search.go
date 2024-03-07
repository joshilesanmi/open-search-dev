package search

import (
	"context"
)

// Query represents a search query with a string value used to perform search operations within the search engine.
type Query struct {
	Value string
}

// IndexOption is a function type that applies configuration options to an IndexOptions instance.
type IndexOption func(*IndexOptions)

// IndexOptions defines configuration options for indexing operations.
// This struct can include various settings that affect how documents are indexed.
type IndexOptions struct {
	Refresh bool // If true, the index is refreshed immediately after the operation, making the changes searchable.
}

// WithIndexRefresh returns an IndexOption that sets the Refresh flag in IndexOptions.
// This can be used to ensure that documents added or updated are immediately searchable.
func WithIndexRefresh(refresh bool) IndexOption {
	return func(opts *IndexOptions) {
		opts.Refresh = refresh
	}
}

// SearchEngine defines an interface for interacting with a search engine.
type SearchEngine interface {
	// CreateIndex initializes a new index with a given name and configuration.
	CreateIndex(ctx context.Context, indexName string, config map[string]interface{}) error

	// DeleteIndex removes an index by its name.
	DeleteIndex(ctx context.Context, indexName string) error

	// PutDocument adds or updates a document within a specific instance and index.
	PutDocument(ctx context.Context, instanceID, indexName, entityName, entityID string, document Document, opts ...IndexOption) error

	// DeleteDocument removes a document from a specific instance and index.
	DeleteDocument(ctx context.Context, instanceID, indexName, entityName, entityID string) error

	// FindDocument retrieves a single document from a specific instance and index.
	FindDocument(ctx context.Context, instanceID, indexName, entityName, entityID string) (Document, error)

	// Search performs a search operation within a specific instance based on the provided query.
	Search(ctx context.Context, instanceID string, query Query) ([]Document, error)
}
