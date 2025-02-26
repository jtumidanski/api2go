package jsonapi

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestJsonapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jsonapi Suite")
}
