package business_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBusiness(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Business Suite")
}
