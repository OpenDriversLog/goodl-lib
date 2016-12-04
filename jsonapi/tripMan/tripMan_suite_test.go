package tripMan_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTripMan(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "tripMan Suite")
}
