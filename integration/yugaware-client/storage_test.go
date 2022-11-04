package yugaware_client_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("yugaware-client integration tests", func() {
	Context("storage commands", func() {
		When(fmt.Sprintf("a %s provider is created", Options.ProviderType), func() {
			BeforeEach(func() {
				err := Storage.ConfigureIfNotExists(ywContext, Options.StorageProviderName)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can list the storage provider", func() {
				out, err := ywContext.RunYugawareCommand("storage", "list")
				Expect(err).NotTo(HaveOccurred())

				Expect(string(out)).To(ContainSubstring(Options.StorageProviderName))
			})
		})
	})
})
