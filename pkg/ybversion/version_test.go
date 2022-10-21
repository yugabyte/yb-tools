package ybversion_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/pkg/ybversion"
)

var _ = Describe("Version", func() {
	DescribeTable("parse version string happy path",
		func(v string, expected ybversion.YBVersion) {
			version, err := ybversion.New(v)
			Expect(err).NotTo(HaveOccurred())

			Expect(version.Major).To(Equal(expected.Major))
			Expect(version.Minor).To(Equal(expected.Minor))
			Expect(version.Patch).To(Equal(expected.Patch))
			Expect(version.Hotfix).To(Equal(expected.Hotfix))
			Expect(version.Build).To(Equal(expected.Build))
		},
		Entry("parses v2.1.0.1", "v2.1.0.1", ybversion.YBVersion{
			Major: 2, Minor: 1, Patch: 0, Hotfix: 1, Build: 0,
		}),
		Entry("parses 2.1.0.1", "2.1.0.1", ybversion.YBVersion{
			Major: 2, Minor: 1, Patch: 0, Hotfix: 1, Build: 0,
		}),
		Entry("parses 2.1.0.1-b24", "2.1.0.1-b24", ybversion.YBVersion{
			Major: 2, Minor: 1, Patch: 0, Hotfix: 1, Build: 24,
		}),
		Entry("parses v2.1.0.1-b24", "v2.1.0.1-b24", ybversion.YBVersion{
			Major: 2, Minor: 1, Patch: 0, Hotfix: 1, Build: 24,
		}),
		Entry("parses 2.0.0.0-b69", "2.0.0.0-b69", ybversion.YBVersion{
			Major: 2, Minor: 0, Patch: 0, Hotfix: 0, Build: 69,
		}),
		Entry("parses 1.0.0.0-b69", "1.0.0.0-b28", ybversion.YBVersion{
			Major: 1, Minor: 0, Patch: 0, Hotfix: 0, Build: 28,
		}),
		Entry("parses 14.15.16.17-b18", "14.15.16.17-b18", ybversion.YBVersion{
			Major: 14, Minor: 15, Patch: 16, Hotfix: 17, Build: 18,
		}),
		Entry("parses v14.15.16.17-b18", "v14.15.16.17-b18", ybversion.YBVersion{
			Major: 14, Minor: 15, Patch: 16, Hotfix: 17, Build: 18,
		}),
		Entry("parses 3.4.5.6-b7", "3.4.5.6-b7", ybversion.YBVersion{
			Major: 3, Minor: 4, Patch: 5, Hotfix: 6, Build: 7,
		}),
	)

	DescribeTable("parse version string error path",
		func(v string, errstring string) {
			_, err := ybversion.New(v)
			Expect(err).To(MatchError(errstring))
		},
		Entry("fails to parse an empty string", "", `invalid version string "": empty version string`),
		Entry("fails to parse only three numbers", "v2.1.3", `invalid version string "v2.1.3": unable to split version string`),
		Entry("cannot parse invalid major version", "2a.1.3.0", `invalid version string "2a.1.3.0": major version string "2a" contains non-numeric value`),
		Entry("cannot parse invalid major version", "major.1.3.0-b24", `invalid version string "major.1.3.0-b24": major version string "major" contains non-numeric value`),
		Entry("cannot parse invalid major version", "a.1.3.0", `invalid version string "a.1.3.0": major version string "a" contains non-numeric value`),
		Entry("cannot parse invalid minor version", "1.b.3.0", `invalid version string "1.b.3.0": minor version string "b" contains non-numeric value`),
		Entry("cannot parse invalid minor version", "1.minor.3.0", `invalid version string "1.minor.3.0": minor version string "minor" contains non-numeric value`),
		Entry("cannot parse invalid minor version", "1.minor.3.0-b33", `invalid version string "1.minor.3.0-b33": minor version string "minor" contains non-numeric value`),
		Entry("cannot parse invalid patch version", "1.42.patch.0", `invalid version string "1.42.patch.0": patch version string "patch" contains non-numeric value`),
		Entry("cannot parse invalid patch version", "1.32.patch.0-b0", `invalid version string "1.32.patch.0-b0": patch version string "patch" contains non-numeric value`),
		Entry("cannot parse invalid hotfix version", "1.32.16.hotfix-b1", `invalid version string "1.32.16.hotfix-b1": hotfix version string "hotfix" contains non-numeric value`),
		Entry("cannot parse invalid build version", "1.32.16.1-build", `invalid version string "1.32.16.1-build": build version string "build" contains non-numeric value`),
		Entry("cannot parse invalid build version", "2.32.16.1-ba", `invalid version string "2.32.16.1-ba": build version string "ba" contains non-numeric value`),
		Entry("cannot parse invalid build version", "3.32.16.1-15", `invalid version string "3.32.16.1-15": found build prefix "1" expected "b"`),
		Entry("cannot parse too large build number", "2.32.16.17-b18446744073709551616", `invalid version string "2.32.16.17-b18446744073709551616": could not parse build version: strconv.ParseUint: parsing "18446744073709551616": value out of range`),
		Entry("cannot parse too large hotfix number", "3.32.16.18446744073709551616", `invalid version string "3.32.16.18446744073709551616": could not parse hotfix version: strconv.ParseUint: parsing "18446744073709551616": value out of range`),
		Entry("cannot parse too large patch number", "3.32.18446744073709551616.16", `invalid version string "3.32.18446744073709551616.16": could not parse patch version: strconv.ParseUint: parsing "18446744073709551616": value out of range`),
		Entry("cannot parse too large minor number", "v2.18446744073709551616.32.16-b10", `invalid version string "v2.18446744073709551616.32.16-b10": could not parse minor version: strconv.ParseUint: parsing "18446744073709551616": value out of range`),
		Entry("cannot parse too large major number", "18446744073709551616.2.32.16", `invalid version string "18446744073709551616.2.32.16": could not parse major version: strconv.ParseUint: parsing "18446744073709551616": value out of range`),
		Entry("cannot parse a custom build version", "3.32.16.1-b5-pre-release", `invalid version string "3.32.16.1-b5-pre-release": build version string "b5-pre-release" contains non-numeric value`),
	)

	DescribeTable("comparison",
		func(v1, v2 ybversion.YBVersion, gt bool, lt bool, eq bool) {
			result := v1.Gt(v2)
			Expect(result).To(Equal(gt))
			result = v1.Lt(v2)
			Expect(result).To(Equal(lt))
			result = v1.Eq(v2)
			Expect(result).To(Equal(eq))
		},
		Entry("two equal versions with different formats", ybversion.MustParse("2.1.2.0"), ybversion.MustParse("v2.1.2.0"), false, false, true),
		Entry("two equal versions with different formats", ybversion.MustParse("2.1.2.0-b0"), ybversion.MustParse("v2.1.2.0"), false, false, true),
		Entry("a version with a build number is greater than the same version with no build number", ybversion.MustParse("2.1.2.0"), ybversion.MustParse("v2.1.2.0-b10"), false, true, false),
		Entry("a version with a build number is greater than the same version with no build number", ybversion.MustParse("2.1.2.0-b10"), ybversion.MustParse("v2.1.2.0"), true, false, false),
		Entry("hotfix version higher", ybversion.MustParse("2.8.2.1-b10"), ybversion.MustParse("v2.8.2.0-b10"), true, false, false),
		Entry("hotfix version higher", ybversion.MustParse("2.12.2.0-b20"), ybversion.MustParse("v2.12.2.1-b20"), false, true, false),
		Entry("patch version higher", ybversion.MustParse("v2.9.4.0-b10"), ybversion.MustParse("2.9.3.0-b10"), true, false, false),
		Entry("patch version higher", ybversion.MustParse("v2.13.2.0-b20"), ybversion.MustParse("2.13.3.0-b20"), false, true, false),
		Entry("minor version higher", ybversion.MustParse("2.13.3.1-b10"), ybversion.MustParse("2.12.3.1-b10"), true, false, false),
		Entry("minor version higher", ybversion.MustParse("2.13.3.0-b20"), ybversion.MustParse("2.14.3.0-b20"), false, true, false),
		Entry("major version higher", ybversion.MustParse("3.13.3.1-b10"), ybversion.MustParse("2.13.3.1-b10"), true, false, false),
		Entry("major version higher", ybversion.MustParse("2.14.3.0-b20"), ybversion.MustParse("3.14.3.0-b20"), false, true, false),
	)
})
