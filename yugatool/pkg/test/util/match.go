package util

import (
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

func ContainObjectMatching(keys gstruct.Keys) types.GomegaMatcher {
	return gstruct.MatchKeys(gstruct.IgnoreExtras, keys)
}
