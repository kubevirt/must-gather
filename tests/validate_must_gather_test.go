package tests_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var logger = log.New(GinkgoWriter, "", 0)

var _ = Describe("validate the must-gather output", func() {
	logger := log.New(GinkgoWriter, "", 0)

	outputDir, err := getDataDir()

	It("[level:product][level:workloads]should find the directory", Label("level:product", "level:workloads"), func() {
		Expect(err).ToNot(HaveOccurred())
	})

	Context("[level:product]validate the installation namespace", Label("level:product"), func() {
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
				Expect(fileInDir(crs, expectedResource)).To(BeTrue())
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

	Context("[level:product]validate the cluster scoped resources", Label("level:product"), func() {
		It("should validate the cluster-scoped-resources directory", func() {

			crsDir := path.Join(outputDir, "cluster-scoped-resources")
			crs, err := os.ReadDir(crsDir)
			Expect(err).ToNot(HaveOccurred())

			expectedResources := map[string]bool{
				"cdiconfigs.cdi.kubevirt.io": false,
				"cdis.cdi.kubevirt.io":       false,
				"networkaddonsconfigs.networkaddonsoperator.network.kubevirt.io": false,
				"virtualmachineclusterinstancetypes.instancetype.kubevirt.io":    false,
				"virtualmachineclusterpreferences.instancetype.kubevirt.io":      false,
			}

			for expectedResource := range expectedResources {
				Expect(fileInDir(crs, expectedResource)).To(BeTrue())
			}

			for _, cr := range crs {
				if _, found := expectedResources[cr.Name()]; !found {
					continue
				}

				Expect(cr.IsDir()).To(BeTrue(), cr.Name(), " should be a directory")
				crDir := path.Join(crsDir, cr.Name())
				crFiles, err := os.ReadDir(crDir)
				Expect(err).ToNot(HaveOccurred())

				if strings.Contains(cr.Name(), "instancetype") {
					Expect(crFiles).ShouldNot(BeEmpty())
				} else {
					Expect(crFiles).Should(HaveLen(1))
				}

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

	Context("[level:product]validate nodes logs", Label("level:product"), func() {
		It("should validate the nodes logs directories", func() {

			expectedResources := []string{
				"audit.log",
				"bridge",
				"dev_vfio",
				"dmesg",
				"ip.txt",
				"lspci",
				"nftables",
				"opt-cni-bin",
				"proc_cmdline",
				"sys_sriov_numvfs",
				"sys_sriov_totalvfs",
				"var-lib-cni-bin",
				"vlan",
			}

			nodesDir := path.Join(outputDir, "nodes")
			nodes, err := os.ReadDir(nodesDir)
			Expect(err).ToNot(HaveOccurred())
			missingExpectedFile := false
			for _, node := range nodes {
				Expect(node.IsDir()).To(BeTrue(), node.Name(), " should be a directory")
				nodeDir := path.Join(nodesDir, node.Name())
				nodeFiles, err := os.ReadDir(nodeDir)
				Expect(err).ToNot(HaveOccurred())
				for _, expectedResource := range expectedResources {
					if !fileInDir(nodeFiles, expectedResource) {
						logger.Printf("node %s info should include the %s file, but it doesn't", node.Name(), expectedResource)
						missingExpectedFile = true
					}
				}
			}
			Expect(missingExpectedFile).To(BeFalse(), "missing expected files")

		})

		logger.Print("outputDir:", outputDir)

	})

	Context("[level:product]validate usage of inspection parameters", Label("level:product"), func() {
		It("should validate inspect and node-logs parameters usage on all the relevant logged commands", func() {
			logfile, err := getMGlogfile()
			Expect(err).ToNot(HaveOccurred())

			space := regexp.MustCompile(`\s+`)

			readFile, err := os.Open(logfile)
			Expect(err).ToNot(HaveOccurred())
			fileScanner := bufio.NewScanner(readFile)
			fileScanner.Split(bufio.ScanLines)
			var inspectcmdLines []string
			var nodelogsLines []string

			for fileScanner.Scan() {
				line := space.ReplaceAllString(fileScanner.Text(), " ")
				if strings.Contains(line, "oc adm inspect") {
					inspectcmdLines = append(inspectcmdLines, line)
				}
				if strings.Contains(line, "oc adm node-logs") {
					nodelogsLines = append(nodelogsLines, line)
				}
			}

			readFile.Close()

			Expect(inspectcmdLines).To(HaveEach(ContainSubstring("${log_collection_args}")), "all the inspect cmd should pass log collection args")
			Expect(nodelogsLines).To(HaveEach(ContainSubstring("${node_log_collection_args}")), "all the node-logs cmd should pass node log collection args")
		})
	})

	Context("[level:workloads]validate workloads", Label("level:workloads"), func() {
		DescribeTable("validate workloads", func(namespace string) {

			vmFile, err := os.Open(path.Join(outputDir, "namespaces", namespace, "kubevirt.io", "virtualmachines.yaml"))
			Expect(err).ToNot(HaveOccurred())
			defer vmFile.Close()

			vms, err := getObjectFromFile(vmFile)
			Expect(vms.Object["items"]).To(HaveLen(20))

			for i, vm := range vms.Object["items"].([]interface{}) {
				expectedName := fmt.Sprintf("testvm-%s-vm%03d", namespace, i+1)
				md := vm.(map[string]interface{})["metadata"].(map[string]interface{})
				Expect(md["name"]).To(Equal(expectedName))

				objFromCluster, err := client.getNamespacedResource(context.Background(), "virtualmachines", "kubevirt.io", namespace, expectedName)
				Expect(err).ToNot(HaveOccurred())
				Expect(reflect.DeepEqual(vm.(map[string]interface{})["spec"], objFromCluster.Object["spec"])).Should(BeTrue())
			}

			vmiFile, err := os.Open(path.Join(outputDir, "namespaces", namespace, "kubevirt.io", "virtualmachineinstances.yaml"))
			Expect(err).ToNot(HaveOccurred())
			defer vmiFile.Close()

			vmis, err := getObjectFromFile(vmiFile)
			Expect(vmis.Object["items"]).To(HaveLen(1))
			vmi := vmis.Object["items"].([]interface{})[0].(map[string]interface{})
			expectedName := fmt.Sprintf("testvm-%s-vm001", namespace)
			Expect(vmi["metadata"].(map[string]interface{})["name"]).To(Equal(expectedName))
			objFromCluster, err := client.getNamespacedResource(context.Background(), "virtualmachineinstances", "kubevirt.io", namespace, expectedName)
			Expect(err).ToNot(HaveOccurred())
			Expect(reflect.DeepEqual(vmi["spec"], objFromCluster.Object["spec"])).Should(BeTrue())

			vmDir := path.Join(outputDir, "namespaces", namespace, "vms", expectedName)
			dir, err := os.ReadDir(vmDir)
			Expect(err).ToNot(HaveOccurred())

			fileExistsNotEmpty := map[string]bool{
				"bridge.txt":          false,
				"dumpxml.xml":         false,
				"ruletables.txt":      false,
				"ip.txt":              false,
				"capabilities.xml":    false,
				"domcapabilities.xml": false,
				"list.txt":            false,
				"domblklist.txt":      false,
				"domjobinfo.txt":      false,
				"blockjob.txt":        false,
			}

			dotLoc := 0
			podName := ""
			for _, f := range dir {
				if strings.HasPrefix(f.Name(), "virt-launcher-testvm") {
					dotLoc = strings.Index(f.Name(), ".")
					podName = f.Name()[:dotLoc]
					break
				}
			}
			Expect(dotLoc).To(BeNumerically(">", 0))
			Expect(podName).ToNot(Equal(""))

			for _, f := range dir {
				if strings.HasPrefix(f.Name(), podName) {
					fi, err := f.Info()
					Expect(err).ToNot(HaveOccurred())
					if fi.Size() > 0 {
						fileExistsNotEmpty[f.Name()[dotLoc+1:]] = true
					}
				}
			}

			Expect(fileExistsNotEmpty).To(BeAllTrueInBoolMap())

			expectedQemuLogName := fmt.Sprintf("%s_testvm-%s-vm001.log", namespace, namespace)
			foundLogFile := false
			for _, f := range dir {
				if f.Name() == expectedQemuLogName {
					foundLogFile = true
					break
				}
			}
			Expect(foundLogFile).To(BeTrue())

			logFile, err := os.Stat(path.Join(vmDir, expectedQemuLogName))
			Expect(err).ToNot(HaveOccurred())
			Expect(logFile.Size()).ToNot(BeZero())

			podFile, err := os.Open(path.Join(outputDir, "namespaces", namespace, "pods", podName, podName+".yaml"))
			Expect(err).ToNot(HaveOccurred())
			podObj, err := getPodFromFile(podFile)
			Expect(err).ToNot(HaveOccurred())
			pod, err := client.getPod(context.Background(), namespace, podName)
			Expect(err).ToNot(HaveOccurred())
			Expect(reflect.DeepEqual(podObj.Spec, pod.Spec)).Should(BeTrue())
		},
			Entry("should gather resources in ns001", "ns001"),
			Entry("should gather resources in ns002", "ns002"),
			Entry("should gather resources in ns003", "ns003"),
			Entry("should gather resources in ns004", "ns004"),
			Entry("should gather resources in ns005", "ns005"),
		)
	})

	Context("[level:product]validate workloads", Label("level:product"), func() {
		// This test assumes, according to automation/create_workloads.sh, that "odd vms", like testvm-ns001-vm003, are
		// custom VMs, and "even vms", like testvm-ns003-vm008, are template based vms.
		DescribeTable("validate virtual machines", func(namespace string) {
			vmsDir := path.Join(outputDir, "namespaces", namespace, "kubevirt.io", "virtualmachines")

			for i := 1; i <= 20; i++ {
				vmName := fmt.Sprintf("testvm-%s-vm%03d", namespace, i)
				vmType := "template-based"
				if i%2 == 1 {
					vmType = "custom"
				}
				vmPath := path.Join(vmsDir, vmType, vmName+".yaml")
				validateVmFile(vmName, namespace, vmPath)

			}
		},
			Entry("should gather resources in ns001", "ns001"),
			Entry("should gather resources in ns002", "ns002"),
			Entry("should gather resources in ns003", "ns003"),
			Entry("should gather resources in ns004", "ns004"),
			Entry("should gather resources in ns005", "ns005"),
		)
	})

	Context("[level:product]validate the virtualization directory", Label("level:product"), func() {
		virtualizationDir := "virtualization"

		// This test assumes, according to automation/create_workloads.sh, that there are 5 running VMs in the cluster.
		It("[test_id:11280]should validate the running VMs count", func() {
			runningVmsCountPath := path.Join(outputDir, virtualizationDir, "running_vms_count.txt")
			countBytes, err := os.ReadFile(runningVmsCountPath)
			Expect(err).ToNot(HaveOccurred())

			count := strings.TrimSpace(string(countBytes))
			Expect(count).To(Equal("5"))
		})
	})
})

func validateVmFile(vm, ns, vmPath string) {
	file, err := os.Open(vmPath)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "can't open the %s file", vmPath)

	objFromCluster, err := client.getNamespacedResource(context.Background(), "virtualmachines", "kubevirt.io", ns, vm)
	Expect(err).ToNot(HaveOccurred())

	objFromFile, err := getObjectFromFile(file)
	Expect(err).ToNot(HaveOccurred())

	clusterSpec, found := objFromCluster.Object["spec"]
	Expect(found).To(BeTrue())
	fileSpec, found := objFromFile.Object["spec"]
	Expect(found).To(BeTrue())

	Expect(reflect.DeepEqual(fileSpec, clusterSpec)).Should(BeTrue())

}

func getDataDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	mgOutputDir, found := os.LookupEnv("MG_OUTPUT_DIR")
	if !found {
		mgOutputDir = "must-gather-output"
	}

	outputDir := path.Join(wd, mgOutputDir)

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

func getMGlogfile() (string, error) {
	const logfilename = "must-gather.log"
	outputDir, err := getDataDir()
	if err != nil {
		return "", err
	}

	mgLogFile := path.Join(outputDir, logfilename)
	if _, err := os.Stat(mgLogFile); err == nil {
		return mgLogFile, nil

	}

	return "", errors.New("can't find must-gather log file")
}
