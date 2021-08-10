package tests_test

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "path"
    "reflect"
    "strings"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/ginkgo/extensions/table"
    . "github.com/onsi/gomega"
    _ "k8s.io/client-go/plugin/pkg/client/auth"
)

var logger = log.New(GinkgoWriter, "", 0)

var _ = Describe("validate the must-gather output", func() {
    logger := log.New(GinkgoWriter, "", 0)
    outputDir, err := getDataDir()
    It("should find the directory", func() {
        Expect(err).ToNot(HaveOccurred())
    })

    Context("validate the installation namespace", func() {
        It("should validate the kubevirt-hyperconverged/crs directory", func() {

            var installationNamespace = "kubevirt-hyperconverged"
            if nsFromVar, found := os.LookupEnv("INSTALLATION_NAMESPACE"); found {
                installationNamespace = nsFromVar
            }

            crsDir := path.Join(outputDir, "namespaces", installationNamespace, "crs")
            crs, err := os.ReadDir(crsDir)
            Expect(err).ToNot(HaveOccurred())

            expectedResources := map[string]bool{
                "hyperconvergeds.hco.kubevirt.io": false,
                "kubevirts.kubevirt.io":           false,
                "ssps.ssp.kubevirt.io":            false,
            }

            for expectedResource := range expectedResources {
                Expect(fileInDir(crs, expectedResource))
            }

            for _, cr := range crs {

                if _, found := expectedResources[cr.Name()]; !found {
                    continue
                }

                Expect(cr.IsDir()).To(BeTrue(), cr.Name(), " should be a directory")
                crDir := path.Join(crsDir, cr.Name())
                crFiles, err := os.ReadDir(crDir)
                Expect(err).ToNot(HaveOccurred())
                Expect(crFiles).Should(HaveLen(1))

                file, err := os.Open(path.Join(crDir, crFiles[0].Name()))
                Expect(err).ToNot(HaveOccurred())

                ext := path.Ext(file.Name())
                Expect(ext).Should(Equal(".yaml"))
                resourceName := path.Base(file.Name())
                resourceName = resourceName[:len(resourceName)-len(ext)]

                resourceTypeSplit := strings.Split(cr.Name(), ".")
                resourceType := resourceTypeSplit[0]
                resourceGroup := strings.Join(resourceTypeSplit[1:], ".")

                objFromCluster, err := client.getNamespacedResource(context.Background(), resourceType, resourceGroup, installationNamespace, resourceName)
                Expect(err).ToNot(HaveOccurred())

                objFromFile, err := getObjectFromFile(file)
                Expect(err).ToNot(HaveOccurred())

                clusterSpec, found := objFromCluster.Object["spec"]
                Expect(found).To(BeTrue())
                fileSpec, found := objFromFile.Object["spec"]
                Expect(found).To(BeTrue())

                Expect(reflect.DeepEqual(fileSpec, clusterSpec)).Should(BeTrue())

                expectedResources[cr.Name()] = true
            }

            Expect(expectedResources).To(BeAllTrueInBoolMap())
        })

        logger.Print("outputDir:", outputDir)
    })

    Context("validate the cluster scoped resources", func() {
        It("should validate the cluster-scoped-resources directory", func() {

            crsDir := path.Join(outputDir, "cluster-scoped-resources")
            crs, err := os.ReadDir(crsDir)
            Expect(err).ToNot(HaveOccurred())

            expectedResources := map[string]bool{
                "cdiconfigs.cdi.kubevirt.io": false,
                "cdis.cdi.kubevirt.io":       false,
                "networkaddonsconfigs.networkaddonsoperator.network.kubevirt.io": false,
                "vmimportconfigs.v2v.kubevirt.io":                                false,
            }

            for expectedResource := range expectedResources {
                Expect(fileInDir(crs, expectedResource))
            }

            for _, cr := range crs {
                if _, found := expectedResources[cr.Name()]; !found {
                    continue
                }

                Expect(cr.IsDir()).To(BeTrue(), cr.Name(), " should be a directory")
                crDir := path.Join(crsDir, cr.Name())
                crFiles, err := os.ReadDir(crDir)
                Expect(err).ToNot(HaveOccurred())
                Expect(crFiles).Should(HaveLen(1))

                if crFiles[0].IsDir() {
                    continue
                }

                file, err := os.Open(path.Join(crDir, crFiles[0].Name()))
                Expect(err).ToNot(HaveOccurred())

                ext := path.Ext(file.Name())
                Expect(ext).Should(Equal(".yaml"), fmt.Sprintf("file %s is not a yaml file", file.Name()))
                resourceName := path.Base(file.Name())
                resourceName = resourceName[:len(resourceName)-len(ext)]

                resourceTypeSplit := strings.Split(cr.Name(), ".")
                resourceType := resourceTypeSplit[0]
                resourceGroup := strings.Join(resourceTypeSplit[1:], ".")

                objFromCluster, err := client.getNonNamespacedResource(context.Background(), resourceType, resourceGroup, resourceName)
                Expect(err).ToNot(HaveOccurred())

                objFromFile, err := getObjectFromFile(file)
                Expect(err).ToNot(HaveOccurred())

                clusterSpec, found := objFromCluster.Object["spec"]
                Expect(found).To(BeTrue())
                fileSpec, found := objFromFile.Object["spec"]
                Expect(found).To(BeTrue())

                Expect(reflect.DeepEqual(fileSpec, clusterSpec)).Should(BeTrue())

                expectedResources[cr.Name()] = true
            }

            Expect(expectedResources).To(BeAllTrueInBoolMap())
        })

        logger.Print("outputDir:", outputDir)
    })

    Context("validate workloads", func() {
        DescribeTable("validate workloads", func(ns string) {
            expectedResources := map[string]bool{
                "datavolumes.cdi.kubevirt.io":         false,
                "virtualmachineinstances.kubevirt.io": false,
                "virtualmachines.kubevirt.io":         false,
            }

            namespace := "ns" + ns

            crsDir := path.Join(outputDir, "namespaces", namespace, "crs")
            crs, err := os.ReadDir(crsDir)
            Expect(err).ToNot(HaveOccurred())

            for expectedResource := range expectedResources {
                Expect(fileInDir(crs, expectedResource))
            }

            for _, cr := range crs {
                if _, found := expectedResources[cr.Name()]; !found {
                    continue
                }

                Expect(cr.IsDir()).To(BeTrue(), cr.Name(), " should be a directory")
                crDir := path.Join(crsDir, cr.Name())
                crFiles, err := os.ReadDir(crDir)
                Expect(err).ToNot(HaveOccurred())
                Expect(crFiles).Should(HaveLen(1))

                if crFiles[0].IsDir() {
                    continue
                }

                file, err := os.Open(path.Join(crDir, crFiles[0].Name()))
                Expect(err).ToNot(HaveOccurred())

                ext := path.Ext(file.Name())
                Expect(ext).Should(Equal(".yaml"), fmt.Sprintf("file %s is not a yaml file", file.Name()))
                resourceName := path.Base(file.Name())
                resourceName = resourceName[:len(resourceName)-len(ext)]

                Expect(resourceName).To(ContainSubstring(ns))

                resourceTypeSplit := strings.Split(cr.Name(), ".")
                resourceType := resourceTypeSplit[0]
                resourceGroup := strings.Join(resourceTypeSplit[1:], ".")

                objFromCluster, err := client.getNamespacedResource(context.Background(), resourceType, resourceGroup, namespace, resourceName)
                Expect(err).ToNot(HaveOccurred())

                objFromFile, err := getObjectFromFile(file)
                Expect(err).ToNot(HaveOccurred())

                clusterSpec, found := objFromCluster.Object["spec"]
                Expect(found).To(BeTrue())
                fileSpec, found := objFromFile.Object["spec"]
                Expect(found).To(BeTrue())

                Expect(reflect.DeepEqual(fileSpec, clusterSpec)).Should(BeTrue())

                expectedResources[cr.Name()] = true
            }
            Expect(expectedResources).To(BeAllTrueInBoolMap())

            },
            Entry("should gather resources in ns1", "1"),
            Entry("should gather resources in ns2", "2"),
            Entry("should gather resources in ns3", "3"),
            Entry("should gather resources in ns4", "4"),
            Entry("should gather resources in ns5", "5"),
        )
    })
})

func getDataDir() (string, error) {
    wd, err := os.Getwd()
    if err != nil {
        return "", err
    }

    outputDir := path.Join(wd, "must-gather-output")

    files, err := os.ReadDir(outputDir)
    if err != nil {
        return "", err
    }

    if len(files) == 0 {
        return "", errors.New("can't find the must-gather output directory")
    }

    for _, file := range files {
        if file.IsDir() {
            return path.Join(outputDir, file.Name()), nil
        }
    }

    return "", errors.New("can't find the cluster directory")
}
