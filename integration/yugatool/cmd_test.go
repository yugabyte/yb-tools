package yugatool_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Yugatool Integration Tests", func() {
	Context("TLS connection", func() {
		When("listing cluster info", func() {
			var (
				err error
			)
			BeforeEach(func() {
				universe := CreateTLSTestUniverseIfNotExists()

				_, err = universe.RunYugatoolCommand("cluster_info")
			})
			It("returns successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("cluster_info", func() {
		When("listing cluster info", func() {
			var (
				err error
			)
			BeforeEach(func() {
				universe := CreateTestUniverseIfNotExists()

				_, err = universe.RunYugatoolCommand("cluster_info", "-o", "json")
				Expect(err).NotTo(HaveOccurred())

				// TODO: Figure out how to decode json when multiple documents can be returned in the same stream

			})
			It("returns successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
