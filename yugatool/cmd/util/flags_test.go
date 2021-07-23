package util_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/yugatool/cmd/util"
)

var _ = Describe("Flags", func() {
	Context("ValidateMastersFlag()", func() {
		var (
			argList = map[string][]string{
				"master-1.hostname.com,master-2.hostname.com,master-3.hostname.com": {"master-1.hostname.com", "master-2.hostname.com"},
				"192.168.1.100,192.168.1.101,192.168.1.102":                         {"192.168.1.100", "192.168.1.101", "192.168.1.102"},
				"240b:c0e0:204:5400:b434:2:0a:482c,240b:c0e0:204:5400:b434:2:0a:482d,240b:c0e0:204:5400:b434:2:0a:482e": {
					"240b:c0e0:204:5400:b434:2:0a:482c", "240b:c0e0:204:5400:b434:2:0a:482d", "240b:c0e0:204:5400:b434:2:0a:482e",
				},
				"master-1":      {"master-1"},
				"192.168.1.100": {"192.168.1.100"},
			}
			returnFlags []string
			err         error
		)

		for flagInput, expectedHosts := range argList {
			When(fmt.Sprintf("given hosts: %s", flagInput), func() {
				BeforeEach(func() {
					returnFlags, err = util.ValidateMastersFlag(flagInput)
				})
				It("returns a list of hosts", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(returnFlags).Should(ContainElements(expectedHosts))
				})
			})
		}

		When("given an empty host list", func() {
			BeforeEach(func() {
				returnFlags, err = util.ValidateMastersFlag("")
			})
			It("returns an error", func() {
				Expect(err).To(MatchError("master host list empty"))
			})
		})
	})
})
