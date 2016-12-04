package dbMan_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDbMan(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DbMan Suite")
}
