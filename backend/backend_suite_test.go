package backend_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// Import the test packages to include them in the suite
	_ "github.com/greenwaltc/kellogg-music-match/backend/business"
)

func TestBackend(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backend Suite")
}
