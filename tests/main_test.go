package tests_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	kubeconfig string
	client     *testClient
)

func TestTests(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Tests Suite")
}
