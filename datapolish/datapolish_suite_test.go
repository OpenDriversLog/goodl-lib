package datapolish_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDatapolish(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Datapolish Suite")
}
