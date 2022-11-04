package ybversion_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestYbversion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ybversion Suite")
}
