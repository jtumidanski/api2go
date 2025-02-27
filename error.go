package api2go

import (
	"encoding/json"
	"fmt"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

// HTTPError is used for errors
type HTTPError struct {
	err    error
	msg    string
	status int
	Errors []jsonapi.Error `json:"errors,omitempty"`
}

func (h HTTPError) Status() int {
	return h.status
}

// NewHTTPError creates a new error with message and status code.
// `err` will be logged (but never sent to a client), `msg` will be sent and `status` is the http status code.
// `err` can be nil.
func NewHTTPError(err error, msg string, status int) HTTPError {
	return HTTPError{err: err, msg: msg, status: status}
}

// Error returns a nice string represenation including the status
func (e HTTPError) Error() string {
	msg := fmt.Sprintf("http error (%d) %s and %d more errors", e.status, e.msg, len(e.Errors))
	if e.err != nil {
		msg += ", " + e.err.Error()
	}

	return msg
}

// marshalHTTPError marshals an internal httpError
func marshalHTTPError(input HTTPError) string {
	if len(input.Errors) == 0 {
		input.Errors = []jsonapi.Error{{Title: input.msg, Status: strconv.Itoa(input.status)}}
	}

	data, err := json.Marshal(input)

	if err != nil {
		return "{}"
	}

	return string(data)
}
