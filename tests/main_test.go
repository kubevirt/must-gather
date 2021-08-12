package tests_test

import (
	"github.com/onsi/ginkgo/reporters"
	"testing"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var (
	kubeconfig string
	client     *testClient
)

func TestTests(t *testing.T) {
	RegisterFailHandler(Fail)

	customReporters := make([]Reporter, 0, 1)
	if JunitOutputFile != "" {
		customReporters = append(customReporters, reporters.NewJUnitReporter(JunitOutputFile))
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Tests Suite", customReporters)
}
