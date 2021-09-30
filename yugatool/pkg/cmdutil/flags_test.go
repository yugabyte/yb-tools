package cmdutil_test

import (
	. "github.com/icza/gox/gox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
)

var _ = Describe("Flags", func() {
	Context("ValidateHostnameList()", func() {
		DescribeTable("happy path",
			func(flagInput string, expectedHosts []*common.HostPortPB) {
				returnFlags, err := cmdutil.ValidateHostnameList(flagInput, client.DefaultMasterPort)

				Expect(err).NotTo(HaveOccurred())
				Expect(returnFlags).Should(ContainElements(expectedHosts))
			},
			Entry("a list of hostnames with no port",
				"master-1.hostname.com,master-2.hostname.com,master-3.hostname.com",
				[]*common.HostPortPB{
					{Host: NewString("master-1.hostname.com"), Port: NewUint32(7100)},
					{Host: NewString("master-2.hostname.com"), Port: NewUint32(7100)},
					{Host: NewString("master-3.hostname.com"), Port: NewUint32(7100)},
				}),
			Entry("a list of hostnames with ports",
				"master-1.hostname.com:3100,master-2.hostname.com:3101,master-3.hostname.com:3102",
				[]*common.HostPortPB{
					{Host: NewString("master-1.hostname.com"), Port: NewUint32(3100)},
					{Host: NewString("master-2.hostname.com"), Port: NewUint32(3101)},
					{Host: NewString("master-3.hostname.com"), Port: NewUint32(3102)},
				}),
			Entry("a list of IPv4 addresses with no ports",
				"192.168.1.100,192.168.1.101,192.168.1.102",
				[]*common.HostPortPB{
					{Host: NewString("192.168.1.100"), Port: NewUint32(7100)},
					{Host: NewString("192.168.1.101"), Port: NewUint32(7100)},
					{Host: NewString("192.168.1.102"), Port: NewUint32(7100)},
				}),
			Entry("a list of IPv4 addresses with port numbers",
				"192.168.1.100:3100,192.168.1.101:3101,192.168.1.102:3102",
				[]*common.HostPortPB{
					{Host: NewString("192.168.1.100"), Port: NewUint32(3100)},
					{Host: NewString("192.168.1.101"), Port: NewUint32(3101)},
					{Host: NewString("192.168.1.102"), Port: NewUint32(3102)},
				}),
			Entry("a list of IPv6 addresses with no port numbers",
				"[240b:c0e0:204:5400:b434:2:0a:482c],[240b:c0e0:204:5400:b434:2:0a:482d],[240b:c0e0:204:5400:b434:2:0a:482e]",
				[]*common.HostPortPB{
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482c"), Port: NewUint32(7100)},
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482d"), Port: NewUint32(7100)},
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482e"), Port: NewUint32(7100)},
				}),
			Entry("a list of IPv6 addresses with port numbers",
				"[240b:c0e0:204:5400:b434:2:0a:482c]:8100,[240b:c0e0:204:5400:b434:2:0a:482d]:8101,[240b:c0e0:204:5400:b434:2:0a:482e]:8102",
				[]*common.HostPortPB{
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482c"), Port: NewUint32(8100)},
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482d"), Port: NewUint32(8101)},
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482e"), Port: NewUint32(8102)},
				}),
			Entry("a single hostname with no port numbers",
				"master-1",
				[]*common.HostPortPB{
					{Host: NewString("master-1"), Port: NewUint32(7100)},
				}),
			Entry("a single hostname with port numbers",
				"master-1:2000",
				[]*common.HostPortPB{
					{Host: NewString("master-1"), Port: NewUint32(2000)},
				}),
			Entry("a single IPv4 address with no port number",
				"192.168.1.100",
				[]*common.HostPortPB{
					{Host: NewString("192.168.1.100"), Port: NewUint32(7100)},
				}),
			Entry("a single IPv4 address with port numbers",
				"192.168.1.100:3000",
				[]*common.HostPortPB{
					{Host: NewString("192.168.1.100"), Port: NewUint32(3000)},
				}),
			Entry("a single IPv6 addresses with no port numbers",
				"[240b:c0e0:204:5400:b434:2:0a:482c]",
				[]*common.HostPortPB{
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482c"), Port: NewUint32(7100)},
				}),
			Entry("a single IPv6 addresses with no port numbers",
				"240b:c0e0:204:5400:b434:2:0a:482c",
				[]*common.HostPortPB{
					{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482c"), Port: NewUint32(7100)},
				}),
		)

		When("given an empty host list", func() {
			var err error
			BeforeEach(func() {
				_, err = cmdutil.ValidateHostnameList("", client.DefaultMasterPort)
			})
			It("returns an error", func() {
				Expect(err).To(MatchError("unable to validate master address: master host list empty"))
			})
		})
	})
})
