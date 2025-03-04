package jsonapi

import (
	"bytes"
	"encoding/json"
	"errors"
)

var objectSuffix = []byte("{")
var arraySuffix = []byte("[")
var stringSuffix = []byte(`"`)

// A Document represents a JSON API document as specified here: http://jsonapi.org.
type Document struct {
	Links    Links                  `json:"links,omitempty"`
	Data     *DataContainer         `json:"data"`
	Included []Data                 `json:"included,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

// A DataContainer is used to marshal and unmarshal single objects and arrays
// of objects.
type DataContainer struct {
	DataObject *Data
	DataArray  []Data
}

// UnmarshalJSON unmarshals the JSON-encoded data to the DataObject field if the
// root element is an object or to the DataArray field for arrays.
func (c *DataContainer) UnmarshalJSON(payload []byte) error {
	if bytes.HasPrefix(payload, objectSuffix) {
		return json.Unmarshal(payload, &c.DataObject)
	}

	if bytes.HasPrefix(payload, arraySuffix) {
		return json.Unmarshal(payload, &c.DataArray)
	}

	return errors.New("expected a JSON encoded object or array")
}

// MarshalJSON returns the JSON encoding of the DataArray field or the DataObject
// field. It will return "null" if neither of them is set.
func (c *DataContainer) MarshalJSON() ([]byte, error) {
	if c.DataArray != nil {
		return json.Marshal(c.DataArray)
	}

	return json.Marshal(c.DataObject)
}

// Link represents a link for return in the document.
type Link struct {
	Href string `json:"href"`
	Meta Meta   `json:"meta,omitempty"`
}

// UnmarshalJSON marshals a string value into the Href field or marshals an
// object value into the whole struct.
func (l *Link) UnmarshalJSON(payload []byte) error {
	// Links may be null in certain cases, mainly noted in the JSONAPI spec
	// with pagination links (http://jsonapi.org/format/#fetching-pagination).
	if len(payload) == 4 && string(payload) == "null" {
		return nil
	}

	if bytes.HasPrefix(payload, stringSuffix) {
		return json.Unmarshal(payload, &l.Href)
	}

	if bytes.HasPrefix(payload, objectSuffix) {
		obj := make(map[string]interface{})
		err := json.Unmarshal(payload, &obj)
		if err != nil {
			return err
		}
		var ok bool
		l.Href, ok = obj["href"].(string)
		if !ok {
			return errors.New(`link object expects a "href" key`)
		}

		l.Meta, _ = obj["meta"].(map[string]interface{})
		return nil
	}

	return errors.New("expected a JSON encoded string or object")
}

// MarshalJSON returns the JSON encoding of only the Href field if the Meta
// field is empty, otherwise it marshals the whole struct.
func (l Link) MarshalJSON() ([]byte, error) {
	if l.Empty() {
		return json.Marshal(nil)
	}
	if len(l.Meta) == 0 {
		return json.Marshal(l.Href)
	}
	return json.Marshal(map[string]interface{}{
		"href": l.Href,
		"meta": l.Meta,
	})
}

// Empty returns true if the link has no href and no metadata.
func (l Link) Empty() bool {
	return len(l.Meta) == 0 && l.Href == ""
}

// Links contains a map of custom Link objects as given by an element.
type Links map[string]Link

// Meta contains unstructured metadata
type Meta map[string]interface{}

// Data is a general struct for document data and included data.
type Data struct {
	Type          string                  `json:"type"`
	ID            string                  `json:"id"`
	Attributes    json.RawMessage         `json:"attributes"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
	Links         Links                   `json:"links,omitempty"`
	Meta          json.RawMessage         `json:"meta,omitempty"`
}

// Relationship contains reference IDs to the related structs
type Relationship struct {
	Links Links                      `json:"links,omitempty"`
	Data  *RelationshipDataContainer `json:"data,omitempty"`
	Meta  map[string]interface{}     `json:"meta,omitempty"`
}

// A RelationshipDataContainer is used to marshal and unmarshal single relationship
// objects and arrays of relationship objects.
type RelationshipDataContainer struct {
	DataObject *RelationshipData
	DataArray  []RelationshipData
}

// UnmarshalJSON unmarshals the JSON-encoded data to the DataObject field if the
// root element is an object or to the DataArray field for arrays.
func (c *RelationshipDataContainer) UnmarshalJSON(payload []byte) error {
	if bytes.HasPrefix(payload, objectSuffix) {
		// payload is an object
		return json.Unmarshal(payload, &c.DataObject)
	}

	if bytes.HasPrefix(payload, arraySuffix) {
		// payload is an array
		return json.Unmarshal(payload, &c.DataArray)
	}

	return errors.New("Invalid json for relationship data array/object")
}

// MarshalJSON returns the JSON encoding of the DataArray field or the DataObject
// field. It will return "null" if neither of them is set.
func (c *RelationshipDataContainer) MarshalJSON() ([]byte, error) {
	if c.DataArray != nil {
		return json.Marshal(c.DataArray)
	}
	return json.Marshal(c.DataObject)
}

// RelationshipData represents one specific reference ID.
type RelationshipData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Error can be used for all kind of application errors
// e.g. you would use it to define form errors or any
// other semantical application problems
// for more information see http://jsonapi.org/format/#errors
type Error struct {
	ID     string       `json:"id,omitempty"`
	Links  *ErrorLinks  `json:"links,omitempty"`
	Status string       `json:"status,omitempty"`
	Code   string       `json:"code,omitempty"`
	Title  string       `json:"title,omitempty"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
	Meta   interface{}  `json:"meta,omitempty"`
}

// ErrorLinks is used to provide an About URL that leads to
// further details about the particular occurrence of the problem.
//
// for more information see http://jsonapi.org/format/#error-objects
type ErrorLinks struct {
	About string `json:"about,omitempty"`
}

// ErrorSource is used to provide references to the source of an error.
//
// The Pointer is a JSON Pointer to the associated entity in the request
// document.
// The Paramter is a string indicating which query parameter caused the error.
//
// for more information see http://jsonapi.org/format/#error-objects
type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
}
