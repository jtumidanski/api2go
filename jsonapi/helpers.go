package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/gedex/inflector"
)

// https://github.com/golang/lint/blob/3d26dc39376c307203d3a221bada26816b3073cf/lint.go#L482
var commonInitialisms = map[string]bool{
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SSH":   true,
	"TLS":   true,
	"TTL":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"JWT":   true,
}

// Jsonify returns a JSON formatted key name from a go struct field name.
func Jsonify(s string) string {
	if s == "" {
		return ""
	}

	if commonInitialisms[s] {
		return strings.ToLower(s)
	}

	rs := []rune(s)
	rs[0] = unicode.ToLower(rs[0])

	return string(rs)
}

// Pluralize returns the pluralization of a noun.
func Pluralize(word string) string {
	return inflector.Pluralize(word)
}

var (
	queryFieldsRegex = regexp.MustCompile(`^fields\[(\w+)\]$`)
)

var ErrRequestedInvalidFields = errors.New("Some requested fields were invalid.")

const (
	codeInvalidQueryFields = "API2GO_INVALID_FIELD_QUERY_PARAM"
)

// FilterSparseFields returns a document with only the specific fields in the response on a per-type basis.
// https://jsonapi.org/format/#fetching-sparse-fieldsets
func FilterSparseFields(resp interface{}, queryParams map[string][]string) (interface{}, []Error) {
	if len(queryParams) < 1 {
		return resp, nil
	}

	wrongFields := map[string][]string{}

	var document *Document
	var ok bool
	if document, ok = resp.(*Document); !ok {
		return resp, nil
	}

	// single entry in data
	data := document.Data.DataObject
	if data != nil {
		errs := replaceAttributes(&queryParams, data)
		for t, v := range errs {
			wrongFields[t] = v
		}
	}

	// data can be a slice too
	datas := document.Data.DataArray
	for index, data := range datas {
		errs := replaceAttributes(&queryParams, &data)
		for t, v := range errs {
			wrongFields[t] = v
		}
		datas[index] = data
	}

	// included slice
	for index, include := range document.Included {
		errs := replaceAttributes(&queryParams, &include)
		for t, v := range errs {
			wrongFields[t] = v
		}
		document.Included[index] = include
	}

	if len(wrongFields) > 0 {
		var errs []Error
		//httpError := NewHTTPError(nil, "Some requested fields were invalid", http.StatusBadRequest)
		for k, v := range wrongFields {
			for _, field := range v {
				errs = append(errs, Error{
					Status: "Bad Request",
					Code:   codeInvalidQueryFields,
					Title:  fmt.Sprintf(`Field "%s" does not exist for type "%s"`, field, k),
					Detail: "Please make sure you do only request existing fields",
					Source: &ErrorSource{
						Parameter: fmt.Sprintf("fields[%s]", k),
					},
				})
			}
		}
		return nil, errs
	}
	return resp, nil
}

// ParseQueryFields returns a map containing lists of field name(s) to be returned by resource type.
// https://jsonapi.org/format/#fetching-sparse-fieldsets
func ParseQueryFields(query *url.Values) (result map[string][]string) {
	result = map[string][]string{}
	for name, param := range *query {
		matches := queryFieldsRegex.FindStringSubmatch(name)
		if len(matches) > 1 {
			match := matches[1]
			result[match] = strings.Split(param[0], ",")
		}
	}

	return
}

func filterAttributes(attributes map[string]interface{}, fields []string) (filteredAttributes map[string]interface{}, wrongFields []string) {
	wrongFields = []string{}
	filteredAttributes = map[string]interface{}{}

	for _, field := range fields {
		if attribute, ok := attributes[field]; ok {
			filteredAttributes[field] = attribute
		} else {
			wrongFields = append(wrongFields, field)
		}
	}

	return
}

func replaceAttributes(query *map[string][]string, entry *Data) map[string][]string {
	fieldType := entry.Type
	attributes := map[string]interface{}{}
	_ = json.Unmarshal(entry.Attributes, &attributes)
	fields := (*query)[fieldType]
	if len(fields) > 0 {
		var wrongFields []string
		attributes, wrongFields = filterAttributes(attributes, fields)
		if len(wrongFields) > 0 {
			return map[string][]string{
				fieldType: wrongFields,
			}
		}
		bytes, _ := json.Marshal(attributes)
		entry.Attributes = bytes
	}

	return nil
}

// ProcessIncludeData processes data contained about object, leveraging the references map to fill included relations.
// Relations of object are set, and thus can be processed recursively in SetReferencedStructs implementation.
func ProcessIncludeData(object interface{}, data Data, references map[string]map[string]Data) error {
	err := json.Unmarshal(data.Attributes, &object)
	if err != nil {
		return err
	}
	if rels := data.Relationships; rels != nil {
		if mr, ok := object.(MarshalReferences); ok {
			for _, ref := range mr.GetReferences() {
				if rel, ok := rels[ref.Name]; ok {
					if rdata := rel.Data; rdata != nil {
						if rdata.DataObject != nil {
							if sto, ok := object.(UnmarshalToOneRelations); ok {
								err = sto.SetToOneReferenceID(ref.Name, rdata.DataObject.ID)
								if err != nil {
									return err
								}
							}
						}
						if rdata.DataArray != nil {
							if stm, ok := object.(UnmarshalToManyRelations); ok {
								ids := make([]string, 0)
								for _, rd := range rdata.DataArray {
									ids = append(ids, rd.ID)
								}
								err = stm.SetToManyReferenceIDs(ref.Name, ids)
							}
						}
					}
				}
			}
		}
	}

	if srs, ok := object.(UnmarshalIncludedRelations); ok {
		err = srs.SetReferencedStructs(references)
		if err != nil {
			return err
		}
	}
	return nil
}
