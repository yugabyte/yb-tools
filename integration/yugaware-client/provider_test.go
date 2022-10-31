package yugaware_client_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("yugaware-client integration tests", func() {
	Context("provider commands", func() {
		When(fmt.Sprintf("a %s provider is created", Options.ProviderType), func() {
			BeforeEach(func() {
				err := Provider.ConfigureIfNotExists(ywContext, Options.ProviderName)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can list the provider", func() {
				out, err := ywContext.RunYugawareCommand("provider", "list")
				Expect(err).NotTo(HaveOccurred())

				Expect(string(out)).To(ContainSubstring(Options.ProviderName))
			})
		})
	})
})
