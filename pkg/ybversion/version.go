package ybversion

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type YBVersion struct {
	Major  uint64
	Minor  uint64
	Patch  uint64
	Hotfix uint64
	Build  uint64
}

// New expects a string in one of the following formats:
// ["v2.4.1.0", "2.4.1.0", "v2.4.1.0-b21", "2.4.1.0-b21"]
//
// When parsed, version string v2.4.1.0-b21 will translate to:
//   YBVersion{Major: 2, Minor: 4, Patch: 1, Hotfix: 0, Build: 21}
func New(vs string) (YBVersion, error) {
	version := YBVersion{}
	var err error
	ybVersionError := func(err error) (YBVersion, error) {
		return version, fmt.Errorf(`invalid version string "%s": %w`, vs, err)
	}
	if len(vs) == 0 {
		return ybVersionError(errors.New("empty version string"))
	}

	ver := strings.Split(vs, ".")
	if len(ver) != 4 {
		return ybVersionError(errors.New("unable to split version string"))
	}

	major := strings.TrimPrefix(ver[0], "v")
	if !containsOnlyNumbers(major) {
		return ybVersionError(fmt.Errorf(`major version string "%s" contains non-numeric value`, ver[0]))
	}

	version.Major, err = strconv.ParseUint(major, 10, 64)
	if err != nil {
		return ybVersionError(fmt.Errorf("could not parse major version: %w", err))
	}

	if !containsOnlyNumbers(ver[1]) {
		return ybVersionError(fmt.Errorf(`minor version string "%s" contains non-numeric value`, ver[1]))
	}

	version.Minor, err = strconv.ParseUint(ver[1], 10, 64)
	if err != nil {
		return ybVersionError(fmt.Errorf("could not parse minor version: %w", err))
	}

	if !containsOnlyNumbers(ver[2]) {
		return ybVersionError(fmt.Errorf(`patch version string "%s" contains non-numeric value`, ver[2]))
	}

	version.Patch, err = strconv.ParseUint(ver[2], 10, 64)
	if err != nil {
		return ybVersionError(fmt.Errorf("could not parse patch version: %w", err))
	}

	var build string
	if before, after, found := strings.Cut(ver[3], "-"); found {
		ver[3] = before

		build = after
	}

	if !containsOnlyNumbers(ver[3]) {
		return ybVersionError(fmt.Errorf(`hotfix version string "%s" contains non-numeric value`, ver[3]))
	}

	version.Hotfix, err = strconv.ParseUint(ver[3], 10, 64)
	if err != nil {
		return ybVersionError(fmt.Errorf("could not parse hotfix version: %w", err))
	}

	if build != "" {
		if !strings.HasPrefix(build, "b") {
			return ybVersionError(fmt.Errorf(`found build prefix "%s" expected "b"`, string(build[0])))
		}
		b := strings.TrimPrefix(build, "b")

		if !containsOnlyNumbers(b) {
			return ybVersionError(fmt.Errorf(`build version string "%s" contains non-numeric value`, build))
		}

		version.Build, err = strconv.ParseUint(b, 10, 64)
		if err != nil {
			return ybVersionError(fmt.Errorf("could not parse build version: %w", err))
		}
	}

	return version, nil
}

// Compare Interface inspired by the bytes.Compare function
// - If v == other function returns 0
// - If v > other function returns 1
// - If v < other function returns -1
func (v YBVersion) Compare(other YBVersion) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}

	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}

	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}

	if v.Hotfix != other.Hotfix {
		if v.Hotfix > other.Hotfix {
			return 1
		}
		return -1
	}

	if v.Build != other.Build {
		if v.Build > other.Build {
			return 1
		}
		return -1
	}

	return 0
}

func (v YBVersion) Lt(other YBVersion) bool {
	return v.Compare(other) == -1
}

func (v YBVersion) Gt(other YBVersion) bool {
	return v.Compare(other) == 1
}

func (v YBVersion) Eq(other YBVersion) bool {
	return v.Compare(other) == 0
}

func containsOnlyNumbers(v string) bool {
	firstNonNumericIndex := strings.IndexFunc(v, func(r rune) bool {
		return !strings.ContainsRune("0123456789", r)
	})

	return firstNonNumericIndex == -1
}

func MustParse(v string) YBVersion {
	version, err := New(v)
	if err != nil {
		panic(err)
	}

	return version
}
