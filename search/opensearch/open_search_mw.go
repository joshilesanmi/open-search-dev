package opensearch

import (
	"context"
	"time"

	"github.com/joshilesanmi/open-search-dev/search"
	"github.com/rs/zerolog"
)

// OpenSearchMiddleware describes a SearchEngine middleware.
type OpenSearchMiddleware func(search.SearchEngine) search.SearchEngine

// OpenSearchLoggingMiddleware takes a logger as a dependency and returns a OpenSearchMiddleware.
func OpenSearchLoggingMiddleware(logger zerolog.Logger) OpenSearchMiddleware {
	return func(next search.SearchEngine) search.SearchEngine {
		return opensearchLoggingMiddleware{
			logger: logger.With().Str("search", "OpenSearch").Logger(),
			next:   next,
		}
	}
}

type opensearchLoggingMiddleware struct {
	logger zerolog.Logger
	next   search.SearchEngine
}

var _ search.SearchEngine = &OpenSearch{}

func (mw opensearchLoggingMiddleware) CreateIndex(ctx context.Context, indexName string, config map[string]interface{}) (err error) {
	defer func(begin time.Time) {
		mw.logger.Log().
			Float64("took", float64(time.Since(begin))/1e6).
			Str("method", "CreateIndex").
			Str("params.indexName", indexName).
			Send()
	}(time.Now())
	return mw.next.CreateIndex(ctx, indexName, config)
}

func (mw opensearchLoggingMiddleware) DeleteIndex(ctx context.Context, indexName string) (err error) {
	defer func(begin time.Time) {
		mw.logger.Log().
			Float64("took", float64(time.Since(begin))/1e6).
			Str("method", "DeleteIndex").
			Str("params.indexName", indexName).
			Send()
	}(time.Now())
	return mw.next.DeleteIndex(ctx, indexName)
}

func (mw opensearchLoggingMiddleware) PutDocument(ctx context.Context, instanceID, indexName, entityName, entityID string, document search.Document, refresh ...search.IndexOption) (err error) {
	defer func(begin time.Time) {
		mw.logger.Log().
			Str("method", "PutDocument").
			Str("params.indexName", indexName).
			Str("params.instanceID", instanceID).
			Str("params.indexName", indexName).
			Str("params.entityID", entityID).
			AnErr("err", err).
			Float64("took", float64(time.Since(begin))/1e6).
			Send()
	}(time.Now())
	return mw.next.PutDocument(ctx, instanceID, indexName, entityName, entityID, document, refresh...)
}

func (mw opensearchLoggingMiddleware) FindDocument(ctx context.Context, instanceID, indexName, entityName, entityID string) (_ search.Document, err error) {
	defer func(begin time.Time) {
		mw.logger.Log().
			Str("method", "FindDocument").
			Str("params.indexName", indexName).
			Str("params.instanceID", instanceID).
			Str("params.indexName", indexName).
			Str("params.entityID", entityID).
			AnErr("err", err).
			Float64("took", float64(time.Since(begin))/1e6).
			Send()
	}(time.Now())
	return mw.next.FindDocument(ctx, instanceID, indexName, entityName, entityID)
}

func (mw opensearchLoggingMiddleware) DeleteDocument(ctx context.Context, instanceID, indexName, entityName, entityID string) (err error) {
	defer func(begin time.Time) {
		mw.logger.Log().
			Str("method", "DeleteDocument").
			Str("params.indexName", indexName).
			Str("params.instanceID", instanceID).
			Str("params.indexName", indexName).
			Str("params.entityID", entityID).
			AnErr("err", err).
			Float64("took", float64(time.Since(begin))/1e6).
			Send()
	}(time.Now())
	return mw.next.DeleteDocument(ctx, instanceID, indexName, entityName, entityID)
}

func (mw opensearchLoggingMiddleware) Search(ctx context.Context, instanceID string, query search.Query) (_ []search.Document, err error) {
	defer func(begin time.Time) {
		mw.logger.Log().
			Str("method", "DeleteDocument").
			Str("params.instanceID", instanceID).
			Str("query.value", query.Value).
			AnErr("err", err).
			Float64("took", float64(time.Since(begin))/1e6).
			Send()
	}(time.Now())
	return mw.next.Search(ctx, instanceID, query)
}
