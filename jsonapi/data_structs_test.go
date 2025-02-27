package jsonapi

import (
	"encoding/json"
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JSONAPI Struct tests", func() {
	Context("Testing array and object data payload", func() {
		It("detects object payload", func() {
			sampleJSON := `{
				"data": {
					"type": "test",
					"id": "1",
					"attributes": {"foo": "bar"},
					"relationships": {
						"author": {
							"data": {"type": "author", "id": "1"}
						}
					}
				}
			}`

			expectedData := &Data{
				Type:       "test",
				ID:         "1",
				Attributes: json.RawMessage([]byte(`{"foo": "bar"}`)),
				Relationships: map[string]Relationship{
					"author": {
						Data: &RelationshipDataContainer{
							DataObject: &RelationshipData{
								Type: "author",
								ID:   "1",
							},
						},
					},
				},
			}

			target := Document{}

			err := json.Unmarshal([]byte(sampleJSON), &target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target.Data.DataObject).To(Equal(expectedData))
		})

		It("detects array payload", func() {
			sampleJSON := `{
				"data": [
					{
						"type": "test",
						"id": "1",
						"attributes": {"foo": "bar"},
						"relationships": {
							"comments": {
								"data": [
									{"type": "comments", "id": "1"},
									{"type": "comments", "id": "2"}
								]
							}
						}
					}
				]
			}`

			expectedData := Data{
				Type:       "test",
				ID:         "1",
				Attributes: json.RawMessage([]byte(`{"foo": "bar"}`)),
				Relationships: map[string]Relationship{
					"comments": {
						Data: &RelationshipDataContainer{
							DataArray: []RelationshipData{
								{
									Type: "comments",
									ID:   "1",
								},
								{
									Type: "comments",
									ID:   "2",
								},
							},
						},
					},
				},
			}

			target := Document{}

			err := json.Unmarshal([]byte(sampleJSON), &target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target.Data.DataArray).To(Equal([]Data{expectedData}))
		})
	})

	It("return an error for invalid relationship data format", func() {
		sampleJSON := `
		{
			"data": [
				{
					"type": "test",
					"id": "1",
					"attributes": {"foo": "bar"},
					"relationships": {
						"comments": {
							"data": "foo"
						}
					}
				}
			]
		}`

		target := Document{}

		err := json.Unmarshal([]byte(sampleJSON), &target)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Invalid json for relationship data array/object"))
	})

	It("creates an empty slice for empty to-many relationships and nil for empty toOne", func() {
		sampleJSON := `{
			"data": [
				{
					"type": "test",
					"id": "1",
					"attributes": {"foo": "bar"},
					"relationships": {
						"comments": {
							"data": []
						},
						"author": {
							"data": null
						}
					}
				}
			]
		}`

		expectedData := Data{
			Type:       "test",
			ID:         "1",
			Attributes: json.RawMessage([]byte(`{"foo": "bar"}`)),
			Relationships: map[string]Relationship{
				"comments": {
					Data: &RelationshipDataContainer{
						DataArray: []RelationshipData{},
					},
				},
				"author": {
					Data: nil,
				},
			},
		}

		target := Document{}

		err := json.Unmarshal([]byte(sampleJSON), &target)
		Expect(err).ToNot(HaveOccurred())
		Expect(target.Data.DataArray).To(Equal([]Data{expectedData}))
	})

	Context("Marshal and Unmarshal link structs", func() {
		It("marshals to a string with no metadata", func() {
			link := Link{Href: "test link"}
			ret, err := json.Marshal(&link)
			Expect(err).ToNot(HaveOccurred())
			Expect(ret).To(MatchJSON(`"test link"`))
		})

		It("marshals to an object with metadata", func() {
			link := Link{
				Href: "test link",
				Meta: map[string]interface{}{
					"test": "data",
				},
			}
			ret, err := json.Marshal(&link)
			Expect(err).ToNot(HaveOccurred())
			Expect(ret).To(MatchJSON(`{
				"href": "test link",
				"meta": {"test": "data"}
			}`))
		})

		It("unmarshals from a string", func() {
			expected := Link{Href: "test link"}
			target := Link{}
			err := json.Unmarshal([]byte(`"test link"`), &target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target).To(Equal(expected))
		})

		It("unmarshals from an object", func() {
			expected := Link{
				Href: "test link",
				Meta: Meta{
					"test": "data",
				},
			}
			target := Link{}
			err := json.Unmarshal([]byte(`{
				"href": "test link",
				"meta": {"test": "data"}
			}`), &target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target).To(Equal(expected))
		})

		It("unmarshals from null", func() {
			expected := Link{}
			target := Link{}
			err := json.Unmarshal([]byte(`null`), &target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target).To(Equal(expected))
		})

		It("unmarshals with an error when href is missing", func() {
			err := json.Unmarshal([]byte(`{}`), &Link{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`link object expects a "href" key`))
		})

		It("unmarshals with an error for syntax error", func() {
			badPayloads := []string{`{`, `"`}
			for _, payload := range badPayloads {
				err := json.Unmarshal([]byte(payload), &Link{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("unexpected end of JSON input"))
			}
		})

		It("unmarshals with an error for wrong types", func() {
			badPayloads := []string{`13`, `[]`}
			for _, payload := range badPayloads {
				err := json.Unmarshal([]byte(payload), &Link{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("expected a JSON encoded string or object"))
			}
		})
	})
})

type ErrorMarshaler struct{}

func (e ErrorMarshaler) Marshal(i interface{}) ([]byte, error) {
	return []byte{}, errors.New("this will always fail")
}
func (e ErrorMarshaler) Unmarshal(data []byte, i interface{}) error {
	return nil
}

func (e ErrorMarshaler) MarshalError(error) string {
	return ""
}

var _ = Describe("Errors test", func() {
	Context("validate error logic", func() {
		It("can create array tree", func() {
			httpErr := NewHTTPError(errors.New("hi"), "hi", 0)
			for i := 0; i < 20; i++ {
				httpErr.Errors = append(httpErr.Errors, Error{})
			}

			Expect(len(httpErr.Errors)).To(Equal(20))
		})
	})

	Context("Marshalling", func() {
		It("will be marshalled correctly with default error", func() {
			httpErr := NewHTTPError(nil, "Invalid use case done", http.StatusInternalServerError)
			result := MarshalHTTPError(httpErr)
			expected := `{"errors":[{"status":"500","title":"Invalid use case done"}]}`
			Expect(result).To(Equal(expected))
		})

		It("will be marshalled correctly without child errors", func() {
			httpErr := NewHTTPError(errors.New("Bad Request"), "Bad Request", 400)
			result := MarshalHTTPError(httpErr)
			expected := `{"errors":[{"status":"400","title":"Bad Request"}]}`
			Expect(result).To(Equal(expected))
		})

		It("will be marshalled correctly with child errors", func() {
			httpErr := NewHTTPError(errors.New("Bad Request"), "Bad Request", 500)

			errorOne := Error{
				ID: "001",
				Links: &ErrorLinks{
					About: "http://bla/blub",
				},
				Status: "500",
				Code:   "001",
				Title:  "Title must not be empty",
				Detail: "Never occures in real life",
				Source: &ErrorSource{
					Pointer: "#titleField",
				},
				Meta: map[string]interface{}{
					"creator": "api2go",
				},
			}

			httpErr.Errors = append(httpErr.Errors, errorOne)

			result := MarshalHTTPError(httpErr)
			expected := `{"errors":[{"id":"001","links":{"about":"http://bla/blub"},"status":"500","code":"001","title":"Title must not be empty","detail":"Never occures in real life","source":{"pointer":"#titleField"},"meta":{"creator":"api2go"}}]}`
			Expect(result).To(Equal(expected))
		})

		It("will be marshalled correctly with child errors without links or source", func() {
			httpErr := NewHTTPError(errors.New("Bad Request"), "Bad Request", 500)

			errorOne := Error{
				ID:     "001",
				Status: "500",
				Code:   "001",
				Title:  "Title must not be empty",
				Detail: "Never occures in real life",
				Meta: map[string]interface{}{
					"creator": "api2go",
				},
			}

			httpErr.Errors = append(httpErr.Errors, errorOne)

			result := MarshalHTTPError(httpErr)
			expected := `{"errors":[{"id":"001","status":"500","code":"001","title":"Title must not be empty","detail":"Never occures in real life","meta":{"creator":"api2go"}}]}`
			Expect(result).To(Equal(expected))
		})
	})
})
