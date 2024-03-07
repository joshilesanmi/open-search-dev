package opensearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/joshilesanmi/open-search-dev/search"
	opensearch "github.com/opensearch-project/opensearch-go/v2"
	opensearchapi "github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/rs/zerolog"
)

// OpenSearch holds the configuration for interacting with OpenSearch clusters.
// It holds references to primary and secondary OpenSearch clients, allowing operations to
// be performed against two separate clusters
type OpenSearch struct {
	primaryClient   *opensearch.Client
	secondaryClient *opensearch.Client
}

// OpenSearchOption defines a function signature for configuring options on an OpenSearch instance.
type OpenSearchOption func(*OpenSearch) error

// Ensures the OpenSearch struct correctly implements the SearchEngine interface.
var _ search.SearchEngine = &OpenSearch{}

// ErrDocumentNotFound is an error that indicates a requested document could not be found in the search index.
var ErrDocumentNotFound = errors.New("document not found")

// ErrDocumentMismatch is an error indicating that there is a mismatch between the expected and actual document.
var ErrDocumentMismatch = errors.New("document mismatch")

// NewOpenSearch initializes and returns a new OpenSearch instance configured with a primary client
// and the option to add a secondary client. The initial configuration sets up the primary client as default.
// Additional configurations can be applied through OpenSearchOption. It also incorporates AWS X-Ray for tracing
// and logging for monitoring and debugging purposes.
func NewOpenSearch(endpoint string, logger zerolog.Logger, opts ...OpenSearchOption) (search.SearchEngine, error) {
	// Wrap the HTTP transport with X-Ray
	xrayTransport := xray.RoundTripper(&http.Transport{
		TLSClientConfig: &tls.Config{},
	})

	client, err := opensearch.NewClient(opensearch.Config{
		Transport: xrayTransport,
		Addresses: []string{endpoint},
	})
	if err != nil {
		return nil, err
	}

	os := &OpenSearch{
		primaryClient: client,
	}

	for _, opt := range opts {
		err := opt(os)
		if err != nil {
			return nil, err
		}
	}

	return OpenSearchLoggingMiddleware(logger)(os), nil
}

// WithSecondaryEndpoint configures an OpenSearch instance to use a secondary endpoint.
func WithSecondaryEndpoint(endpoint string) OpenSearchOption {
	return func(os *OpenSearch) error {
		xrayTransport := xray.RoundTripper(&http.Transport{
			TLSClientConfig: &tls.Config{},
		})
		client, err := opensearch.NewClient(opensearch.Config{
			Transport: xrayTransport,
			Addresses: []string{endpoint},
		})
		if err != nil {
			return err
		}
		os.secondaryClient = client
		return nil
	}
}

// CreateIndex creates an index with the specified name and configuration on both the primary and,
// if configured, the secondary OpenSearch clients.
func (os *OpenSearch) CreateIndex(ctx context.Context, indexName string, config map[string]interface{}) error {
	configByte, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal index config %v", err)
	}

	if err := os.ensureIndex(ctx, os.primaryClient, indexName, configByte); err != nil {
		return fmt.Errorf("primary client: %v", err)
	}

	if os.secondaryClient != nil {
		if err := os.ensureIndex(ctx, os.secondaryClient, indexName, configByte); err != nil {
			return fmt.Errorf("secondary client: %v", err)
		}
	}

	return nil
}

// PutDocument handles the insertion or update of a document within a specified OpenSearch index. It adds to
// the document metadata (instanceID, entityName, and entityID) and generates a unique ID for it. The function
// allows extra index options like refresh. Initially stored in the primary OpenSearch cluster, the document
// is also be stored to a secondary cluster, if it is configured.
func (os *OpenSearch) PutDocument(ctx context.Context, instanceID, indexName, entityName, entityID string, document search.Document, opts ...search.IndexOption) error {
	// Add necessary metadata to the document before insertion.
	d, err := document.AddDocumentMetaData(instanceID, entityName, entityID)
	if err != nil {
		return fmt.Errorf("missing document meta data %v", err)
	}

	docByte, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal document %v", err)
	}

	// Generate a unique ID for the document using instanceID, entityName, and entityID.
	documentID := search.GenerateDocumentID(instanceID, entityName, entityID)

	options := &search.IndexOptions{Refresh: false}
	for _, opt := range opts {
		opt(options)
	}

	refresh := strconv.FormatBool(options.Refresh)

	// Store the document in the index on the primary client.
	if err = os.putDocument(ctx, os.primaryClient, indexName, documentID, docByte, refresh); err != nil {
		return fmt.Errorf("primary client: %v", err)
	}

	// If a secondary client is configured, store the document there as well.
	if os.secondaryClient != nil {
		if err := os.putDocument(ctx, os.secondaryClient, indexName, documentID, docByte, refresh); err != nil {
			return fmt.Errorf("secondary client: %v", err)
		}
	}

	return nil
}

// FindDocument searches for a document within an index based on the provided documentID. It attempts to retrieve
// the document from the primary OpenSearch client and, if a secondary client is configured, verifies the document's
// consistency across both clients.
func (os *OpenSearch) FindDocument(ctx context.Context, instanceID, indexName, entityName, entityID string) (search.Document, error) {
	documentID := search.GenerateDocumentID(instanceID, entityName, entityID)

	pryDoc, err := os.findDocument(ctx, os.primaryClient, indexName, documentID)
	if err != nil {
		return nil, fmt.Errorf("primary client: %w", err)
	}

	if os.secondaryClient != nil {
		secDoc, err := os.findDocument(ctx, os.secondaryClient, indexName, documentID)
		if err != nil {
			return nil, fmt.Errorf("secondary client: %w", err)
		}

		if !compareDocuments(pryDoc, secDoc) {
			return nil, fmt.Errorf("documents mismatch for id %q: %w", entityID, ErrDocumentMismatch)
		}
	}

	return pryDoc, nil
}

// DeleteDocument removes a document from the specified index in both the primary and, if configured, the secondary
// OpenSearch clients.
func (os *OpenSearch) DeleteDocument(ctx context.Context, instanceID, indexName, entityName, entityID string) error {
	documentID := search.GenerateDocumentID(instanceID, entityName, entityID)

	if err := os.deleteDocument(ctx, os.primaryClient, indexName, documentID); err != nil {
		return fmt.Errorf("primary client: %v", err)
	}

	if os.secondaryClient != nil {
		if err := os.deleteDocument(ctx, os.secondaryClient, indexName, documentID); err != nil {
			return fmt.Errorf("secondary client: %v", err)
		}
	}

	return nil
}

// DeleteIndex removes an entire index from both the primary and, if configured, the secondary OpenSearch clients.
func (os *OpenSearch) DeleteIndex(ctx context.Context, indexName string) error {
	if err := os.deleteIndex(ctx, os.primaryClient, indexName); err != nil {
		return fmt.Errorf("primary client: %v", err)
	}

	if os.secondaryClient != nil {
		if err := os.deleteIndex(ctx, os.secondaryClient, indexName); err != nil {
			return fmt.Errorf("secondary client: %v", err)
		}
	}

	return nil
}

// Search performs a search operation across documents in an index based on a given query and instance ID.
// This method constructs a search query that includes both a search term and a filter for the instance ID,
// ensuring that only documents relevant to the specified instance and matching the search criteria are returned.
func (os *OpenSearch) Search(ctx context.Context, instanceID string, query search.Query) ([]search.Document, error) {
	searchQuery := os.constructSearchQuery(instanceID, query)

	q, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %v", err)
	}

	searchReq := opensearchapi.SearchRequest{
		Body: bytes.NewReader(q),
	}

	resp, err := os.executeReadRequest(ctx, os.primaryClient, searchReq)
	if err != nil {
		return nil, err
	}

	return os.extractDocumentsFromSearchResponse(resp)
}

// ensureIndex checks if an index exists, and creates it if not.
func (os *OpenSearch) ensureIndex(ctx context.Context, client *opensearch.Client, indexName string, body []byte) error {
	exists, err := os.indexExists(ctx, client, indexName)
	if err != nil {
		return fmt.Errorf("failed to check if index exist: %v", err)
	}
	if !exists {
		if err := os.createIndex(ctx, client, indexName, body); err != nil {
			return fmt.Errorf("failed to create index: %v", err)
		}
	}
	return nil
}

// indexExists checks if an index exists in OpenSearch.
func (os *OpenSearch) indexExists(ctx context.Context, client *opensearch.Client, indexName string) (bool, error) {
	req := opensearchapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	resp, err := os.executeReadRequest(ctx, client, req)
	if err != nil {
		return false, err
	}

	// A 200 OK response means the index exists.
	// A 404 Not Found response means the index does not exist.
	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// For any other response, return an error.
	return false, fmt.Errorf("unexpected response status checking index exists: %d", resp.StatusCode)
}

// createIndex sends a request to create an index with the specified name and configuration body on the OpenSearch
// cluster using the provided client.
func (os *OpenSearch) createIndex(ctx context.Context, client *opensearch.Client, indexName string, body []byte) error {
	req := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  bytes.NewReader(body),
	}

	return os.executeRequest(ctx, client, &req)
}

// putDocument sends a request to index or update a document in the specified index using the provided OpenSearch client.
// It allows for immediate refresh of the index based on the refresh parameter to make the document searchable right.
func (os *OpenSearch) putDocument(ctx context.Context, client *opensearch.Client, indexName, documentID string, body []byte, refresh string) error {
	req := opensearchapi.IndexRequest{
		Index:      indexName,
		DocumentID: documentID,
		Body:       bytes.NewReader(body),
		Refresh:    refresh,
	}

	return os.executeRequest(ctx, client, &req)
}

// findDocument retrieves a document by its ID from the specified index using the provided OpenSearch client.
func (os *OpenSearch) findDocument(ctx context.Context, client *opensearch.Client, indexName, documentID string) (search.Document, error) {
	req := opensearchapi.GetRequest{
		Index:      indexName,
		DocumentID: documentID,
	}

	resp, err := os.executeReadRequest(ctx, client, req)
	if err != nil {
		return nil, err
	}

	var r struct {
		Source search.Document `json:"_source"`
	}

	err = decodeResponse(resp, &r)
	if err != nil {
		return nil, err
	}

	return r.Source, nil
}

func (os *OpenSearch) deleteDocument(ctx context.Context, client *opensearch.Client, indexName, documentID string) error {
	req := opensearchapi.DeleteRequest{
		Index:      indexName,
		DocumentID: documentID,
	}

	return os.executeRequest(ctx, client, &req)
}

// deleteIndex sends a request to delete an index from the OpenSearch cluster using the specified client.
func (os *OpenSearch) deleteIndex(ctx context.Context, client *opensearch.Client, indexName string) error {
	req := opensearchapi.IndicesDeleteRequest{
		Index: []string{indexName},
	}

	return os.executeRequest(ctx, client, &req)
}

// executeRequest performs a generic OpenSearch API request using the provided client and request parameters.
// It is a utility function designed to handle the execution of various OpenSearch requests.
func (os *OpenSearch) executeRequest(ctx context.Context, client *opensearch.Client, req opensearchapi.Request) error {
	resp, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("request failed: %s", resp.String())
	}

	return nil
}

// executeReadRequest performs a generic request using the provided OpenSearch client and request parameters,
// specifically tailored for read operations such as document retrieval or search.
func (os *OpenSearch) executeReadRequest(ctx context.Context, client *opensearch.Client, req opensearchapi.Request) (*opensearchapi.Response, error) {
	resp, err := req.Do(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	return resp, nil
}

// constructSearchQuery builds the search query.
func (os *OpenSearch) constructSearchQuery(instanceID string, query search.Query) map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": map[string]interface{}{
					"query_string": map[string]interface{}{
						"query": query.Value,
					},
				},
				"filter": map[string]interface{}{
					"term": map[string]string{
						"instance_id": instanceID,
					},
				},
			},
		},
	}
}

// extractDocumentsFromSearchResponse processes the search response and extracts documents.
func (os *OpenSearch) extractDocumentsFromSearchResponse(resp *opensearchapi.Response) ([]search.Document, error) {
	var r struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := decodeResponse(resp, &r); err != nil {
		return nil, err
	}

	documents := make([]search.Document, 0)
	for _, hit := range r.Hits.Hits {
		documents = append(documents, hit.Source)
	}

	return documents, nil
}

// decodeResponse takes an OpenSearch API response and decodes its body into a target.
// This function is a utility for unmarshaling JSON responses from OpenSearch into defined type.
// It checks HTTP error statuses in the response and specifically detecting a document not found condition.
func decodeResponse(resp *opensearchapi.Response, target interface{}) error {
	if resp.IsError() {
		if resp.StatusCode == http.StatusNotFound {
			return ErrDocumentNotFound
		}
		return fmt.Errorf("error in response: %s", resp.String())
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(target)
}

// compareDocuments compares two search.Document maps for equality
func compareDocuments(doc1, doc2 search.Document) bool {
	if len(doc1) != len(doc2) {
		return false
	}

	for key, value1 := range doc1 {
		if value2, ok := doc2[key]; ok {
			if value1 != value2 {
				return false
			}
		} else {
			return false
		}
	}

	return true
}
