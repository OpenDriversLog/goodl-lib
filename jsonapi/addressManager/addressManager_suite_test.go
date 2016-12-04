package addressManager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAddressManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AddressManager Suite")
}
