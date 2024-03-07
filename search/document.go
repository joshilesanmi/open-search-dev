package search

import (
	"errors"
	"fmt"
)

// Document represents a generic structure for storing document data within a search engine.
type Document map[string]interface{}

// GenerateDocumentKey creates a unique key for storing the document.
func GenerateDocumentID(instanceID, entityName, entityID string) string {
	return fmt.Sprintf("%s-%s-%s", instanceID, entityName, entityID)
}

// PrepareForMarshal merges the metadata with the document data into a single map.
func (d Document) AddDocumentMetaData(instanceID, entityName, entityID string) (Document, error) {
	if entityID == "" {
		return nil, errors.New("entityID is required")
	}

	if instanceID == "" {
		return nil, errors.New("instanceID is required")
	}

	if entityName == "" {
		return nil, errors.New("entityName is required")
	}

	// Add metadata fields to the merged map
	d["id"] = entityID
	d["instance_id"] = instanceID
	d["entity_name"] = entityName

	return d, nil
}
