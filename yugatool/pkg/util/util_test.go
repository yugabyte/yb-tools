package util_test

import (
	. "github.com/icza/gox/gox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

var _ = Describe("Util", func() {
	Context("HostPortString()", func() {
		DescribeTable("translates into a string", func(hostport *common.HostPortPB, expectedString string) {
			hostString := util.HostPortString(hostport)

			Expect(hostString).To(Equal(expectedString))
		},
			Entry("an IPv4 hostport", &common.HostPortPB{Host: NewString("192.168.1.1"), Port: NewUint32(7100)}, "192.168.1.1:7100"),
			Entry("an IPv6 hostport", &common.HostPortPB{Host: NewString("240b:c0e0:204:5400:b434:2:0a:482c"), Port: NewUint32(7100)}, "[240b:c0e0:204:5400:b434:2:0a:482c]:7100"),
			Entry("an IPv6 hostport", &common.HostPortPB{Host: NewString("[240b:c0e0:204:5400:b434:2:0a:482c]"), Port: NewUint32(7100)}, "[240b:c0e0:204:5400:b434:2:0a:482c]:7100"),
			Entry("an hostname hostport", &common.HostPortPB{Host: NewString("master-1.common.svc.local"), Port: NewUint32(7100)}, "master-1.common.svc.local:7100"),
			// TODO: What to do about notation like this?
			// Entry("an IPv4 literal hostport", &common.HostPortPB{Host: NewString("::FFFF:50.214.15.3"), Port: NewUint32(7100)}, "[::FFFF:50.214.15.3]:7100"),
		)
	})

	Context("IsBasicIPv6()", func() {
		DescribeTable("identifies the host string", func(host string, expected bool) {
			hostString := util.IsBasicIPv6(host)

			Expect(hostString).To(Equal(expected))
		},
			Entry("an IPv4 hostport", "192.168.1.1", false),
			Entry("an IPv6 hostport", "240b:c0e0:204:5400:b434:2:0a:482c", true),
			Entry("an IPv6 hostport", "[240b:c0e0:204:5400:b434:2:0a:482c]", false),
			Entry("an hostname hostport", "master-1.common.svc.local", false),
			// TODO: What to do about notation like this?
			// Entry("an IPv4 literal hostport", &common.HostPortPB{Host: NewString("::FFFF:50.214.15.3"), Port: NewUint32(7100)}, "[::FFFF:50.214.15.3]:7100"),
		)
	})

	Context("IsTLS()", func() {
		DescribeTable("when given TlsOptionsPB", func(tlsOptions *config.TlsOptionsPB, expected bool) {
			hasTLS := util.HasTLS(tlsOptions)

			Expect(hasTLS).To(Equal(expected))
		},
			Entry("a nil TlsOptionsPB", nil, false),
			Entry("an empty TlsOptionsPB", &config.TlsOptionsPB{}, false),
			Entry("TlsOptionsPB with all defaults", &config.TlsOptionsPB{
				SkipHostVerification: NewBool(false),
				CaCertPath:           NewString(""),
				CertPath:             NewString(""),
				KeyPath:              NewString(""),
			}, false),
			Entry("TlsOptionsPB with just SkipHostVerification set to false", &config.TlsOptionsPB{
				SkipHostVerification: NewBool(false),
			}, false),
			Entry("TlsOptionsPB with just SkipHostVerification set to true", &config.TlsOptionsPB{
				SkipHostVerification: NewBool(true),
			}, true),
			Entry("TlsOptionsPB with CaCertPath set", &config.TlsOptionsPB{
				CaCertPath: NewString("/test.crt"),
			}, true),
			Entry("TlsOptionsPB with CaCertPath set", &config.TlsOptionsPB{
				CertPath: NewString("cert.set"),
				KeyPath:  NewString("key.set"),
			}, true),
		)
	})
})
