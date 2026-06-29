package tests_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
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

	Context("[level:incident]validate vm-incident output", Label("level:incident"), func() {
		var incidentDir string
		var incidentNs string

		BeforeEach(func() {
			incidentOutputDir, found := os.LookupEnv("MG_INCIDENT_OUTPUT_DIR")
			if !found {
				incidentOutputDir = "must-gather-incident-output"
			}

			wd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			baseDir := path.Join(wd, incidentOutputDir)
			files, err := os.ReadDir(baseDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).ToNot(BeEmpty())

			for _, file := range files {
				if file.IsDir() {
					dirPath := path.Join(baseDir, file.Name())
					if checkVersionFile(dirPath) {
						incidentDir = dirPath
						break
					}
				}
			}
			Expect(incidentDir).ToNot(BeEmpty(), "should find the incident must-gather output directory")

			// Derive incident namespace from incident-summary.yaml
			// so tests don't depend on env vars for data already in the archive
			if ns := os.Getenv("INCIDENT_NS"); ns != "" {
				incidentNs = ns
			} else {
				summaryContent, readErr := os.ReadFile(path.Join(incidentDir, "incident-summary.yaml"))
				Expect(readErr).ToNot(HaveOccurred(), "incident-summary.yaml must be readable")
				for _, line := range strings.Split(string(summaryContent), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "namespace:") {
						incidentNs = strings.TrimSpace(strings.TrimPrefix(line, "namespace:"))
						break
					}
				}
				Expect(incidentNs).ToNot(BeEmpty(), "should find namespace in incident-summary.yaml")
			}
		})

		It("should produce an incident-summary.yaml", func() {
			summaryPath := path.Join(incidentDir, "incident-summary.yaml")
			info, err := os.Stat(summaryPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Size()).To(BeNumerically(">", 0))

			content, err := os.ReadFile(summaryPath)
			Expect(err).ToNot(HaveOccurred())
			summaryStr := string(content)
			Expect(summaryStr).To(ContainSubstring("incident:"))
			Expect(summaryStr).To(ContainSubstring("namespace:"))
			Expect(summaryStr).To(ContainSubstring("vm:"))
			Expect(summaryStr).To(ContainSubstring("incident_time:"))
			Expect(summaryStr).To(ContainSubstring("node:"))
			Expect(summaryStr).To(ContainSubstring("discovery_method:"))
			Expect(summaryStr).To(ContainSubstring("collected:"))
			Expect(summaryStr).To(ContainSubstring("elapsed_seconds:"))
		})

		It("should complete incident collection within 3 minutes", func() {
			summaryPath := path.Join(incidentDir, "incident-summary.yaml")
			content, err := os.ReadFile(summaryPath)
			Expect(err).ToNot(HaveOccurred())

			var elapsed int
			for _, line := range strings.Split(string(content), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "elapsed_seconds:") {
					val := strings.TrimSpace(strings.TrimPrefix(line, "elapsed_seconds:"))
					elapsed, err = strconv.Atoi(val)
					Expect(err).ToNot(HaveOccurred(), "elapsed_seconds should be an integer")
					break
				}
			}
			Expect(elapsed).To(BeNumerically(">", 0), "elapsed_seconds should be recorded")
			Expect(elapsed).To(BeNumerically("<=", 180),
				fmt.Sprintf("incident collection took %ds, must complete within 180s (3 minutes)", elapsed))
			logger.Printf("  incident collection elapsed: %ds (limit: 180s)", elapsed)
		})

		It("should produce an archive smaller than 100MB", func() {
			var totalBytes int64
			err := filepath.Walk(incidentDir, func(_ string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				totalBytes += info.Size()
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			totalMB := float64(totalBytes) / (1024 * 1024)
			Expect(totalMB).To(BeNumerically("<", 100),
				fmt.Sprintf("incident archive is %.1f MB, must be under 100 MB", totalMB))
			logger.Printf("  incident archive size: %.1f MB (limit: 100 MB)", totalMB)
		})

		It("should collect node diagnostics", func() {
			nodesDir := path.Join(incidentDir, "nodes")
			nodes, err := os.ReadDir(nodesDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).ToNot(BeEmpty(), "should have at least one node directory")

			nodeDir := path.Join(nodesDir, nodes[0].Name())
			nodeFiles, err := os.ReadDir(nodeDir)
			Expect(err).ToNot(HaveOccurred())

			expectedFiles := []string{
				"dmesg",
			}
			for _, expected := range expectedFiles {
				Expect(fileInDir(nodeFiles, expected)).To(BeTrue(),
					fmt.Sprintf("node directory should contain %s", expected))
			}

			optionalNodeFiles := []string{
				"dmidecode", "lspci", "cmdline", "sysctl", "chrony", "ip_addr",
				"nfs_module_params", "tuned_active_profile", "tuned_modprobe",
				"nfs.conf", "nfs_modprobe", "kernel_red_flags.log",
			}
			for _, opt := range optionalNodeFiles {
				if fileInDir(nodeFiles, opt) {
					logger.Printf("  node diagnostic present: %s", opt)
				} else {
					logger.Printf("  node diagnostic absent (may not be available on this host): %s", opt)
				}
			}

			journalPattern := "_logs_journal"
			kubeletPattern := "_logs_kubelet"
			foundJournal := false
			foundKubelet := false
			for _, f := range nodeFiles {
				if strings.Contains(f.Name(), journalPattern) {
					foundJournal = true
				}
				if strings.Contains(f.Name(), kubeletPattern) {
					foundKubelet = true
				}
			}
			Expect(foundJournal).To(BeTrue(), "should have journal logs")
			Expect(foundKubelet).To(BeTrue(), "should have kubelet logs")
		})

		It("should collect storage diagnostics from the node", func() {
			nodesDir := path.Join(incidentDir, "nodes")
			nodes, err := os.ReadDir(nodesDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).ToNot(BeEmpty())

			nodeDir := path.Join(nodesDir, nodes[0].Name())
			nodeFiles, err := os.ReadDir(nodeDir)
			Expect(err).ToNot(HaveOccurred())

			requiredFiles := []string{
				"mountstats",
				"diskstats",
				"df",
				"mounts",
			}
			for _, expected := range requiredFiles {
				Expect(fileInDir(nodeFiles, expected)).To(BeTrue(),
					fmt.Sprintf("node directory should contain %s", expected))
			}

			// PSI pressure files require kernel CONFIG_PSI — not available on all hosts
			optionalPSI := []string{"pressure_io", "pressure_memory", "pressure_cpu"}
			for _, opt := range optionalPSI {
				if fileInDir(nodeFiles, opt) {
					logger.Printf("  optional storage diagnostic present: %s", opt)
				} else {
					logger.Printf("  optional storage diagnostic absent (kernel may lack PSI support): %s", opt)
				}
			}
		})

		It("should collect kernel tunables and tuned profile from the node", func() {
			nodesDir := path.Join(incidentDir, "nodes")
			nodes, err := os.ReadDir(nodesDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).ToNot(BeEmpty())

			nodeDir := path.Join(nodesDir, nodes[0].Name())
			nodeFiles, err := os.ReadDir(nodeDir)
			Expect(err).ToNot(HaveOccurred())

			Expect(fileInDir(nodeFiles, "sysctl")).To(BeTrue(),
				"node directory should contain sysctl")
			sysctlPath := path.Join(nodeDir, "sysctl")
			info, err := os.Stat(sysctlPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Size()).To(BeNumerically(">", 0), "sysctl should not be empty")

			optionalTunables := []string{"tuned_active_profile", "tuned_modprobe", "nfs_module_params"}
			for _, opt := range optionalTunables {
				if fileInDir(nodeFiles, opt) {
					logger.Printf("  tunable present: %s", opt)
				} else {
					logger.Printf("  tunable absent (not available on this host): %s", opt)
				}
			}
		})

		It("should collect network, time sync, and kernel red-flags from the node", func() {
			nodesDir := path.Join(incidentDir, "nodes")
			nodes, err := os.ReadDir(nodesDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).ToNot(BeEmpty())

			nodeDir := path.Join(nodesDir, nodes[0].Name())
			nodeFiles, err := os.ReadDir(nodeDir)
			Expect(err).ToNot(HaveOccurred())

			diagnosticFiles := []string{
				"ip_addr",
				"chrony",
			}
			for _, expected := range diagnosticFiles {
				Expect(fileInDir(nodeFiles, expected)).To(BeTrue(),
					fmt.Sprintf("node directory should contain %s", expected))
			}
		})

		It("should have kernel_red_flags.log with content", func() {
			nodesDir := path.Join(incidentDir, "nodes")
			nodes, err := os.ReadDir(nodesDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).ToNot(BeEmpty())

			nodeDir := path.Join(nodesDir, nodes[0].Name())
			redFlagsPath := path.Join(nodeDir, "kernel_red_flags.log")
			info, err := os.Stat(redFlagsPath)
			Expect(err).ToNot(HaveOccurred(), "kernel_red_flags.log should exist")
			Expect(info.Size()).To(BeNumerically(">", 0), "kernel_red_flags.log should not be empty")

			content, err := os.ReadFile(redFlagsPath)
			Expect(err).ToNot(HaveOccurred())
			contentStr := string(content)
			hasRedFlags := strings.Contains(contentStr, "Call Trace") ||
				strings.Contains(contentStr, "BUG:") ||
				strings.Contains(contentStr, "WARNING:") ||
				strings.Contains(contentStr, "OOM") ||
				strings.Contains(contentStr, "error") ||
				strings.Contains(contentStr, "hung_task")
			hasSentinel := strings.Contains(contentStr, "# No kernel red-flag patterns found")
			Expect(hasRedFlags || hasSentinel).To(BeTrue(),
				"kernel_red_flags.log should contain either red-flag patterns or the sentinel message")
		})

		It("should collect cluster health snapshot", func() {
			clusterDir := path.Join(incidentDir, "cluster-scoped-resources")
			_, err := os.Stat(clusterDir)
			Expect(err).ToNot(HaveOccurred(), "cluster-scoped-resources directory should exist")

			clusterFiles, err := os.ReadDir(clusterDir)
			Expect(err).ToNot(HaveOccurred())

			expectedFiles := []string{
				"cluster_operators",
				"nodes_overview",
				"machineconfigpools",
				"top_node",
				"kubevirt_version",
				"vmis_on_incident_node",
			}
			for _, expected := range expectedFiles {
				Expect(fileInDir(clusterFiles, expected)).To(BeTrue(),
					fmt.Sprintf("cluster-scoped-resources should contain %s", expected))
			}
		})

		It("should collect VM and VMI definitions", func() {
			nsDir := path.Join(incidentDir, "namespaces", incidentNs)
			_, err := os.Stat(nsDir)
			Expect(err).ToNot(HaveOccurred(), "namespace directory should exist")
		})

		It("should collect virt-launcher pod logs with actual log content", func() {
			podsDir := path.Join(incidentDir, "namespaces", incidentNs, "core", "pods")
			_, err := os.Stat(podsDir)
			Expect(err).ToNot(HaveOccurred(), "pods directory should exist")

			pods, err := os.ReadDir(podsDir)
			Expect(err).ToNot(HaveOccurred())

			var launcherPodDir string
			for _, pod := range pods {
				if strings.Contains(pod.Name(), "virt-launcher") && pod.IsDir() {
					launcherPodDir = path.Join(podsDir, pod.Name())
					break
				}
			}
			Expect(launcherPodDir).ToNot(BeEmpty(), "should have a virt-launcher pod directory")

			logFiles, err := os.ReadDir(launcherPodDir)
			Expect(err).ToNot(HaveOccurred())

			foundCurrentLog := false
			for _, f := range logFiles {
				if strings.HasSuffix(f.Name(), "-current.log") {
					foundCurrentLog = true
					info, statErr := f.Info()
					Expect(statErr).ToNot(HaveOccurred())
					Expect(info.Size()).To(BeNumerically(">", 0),
						fmt.Sprintf("log file %s should not be empty", f.Name()))
				}
			}
			Expect(foundCurrentLog).To(BeTrue(),
				"virt-launcher pod directory should contain at least one *-current.log file")
		})

		It("should collect virt-handler pod logs with actual log content", func() {
			installNs, found := os.LookupEnv("INSTALLATION_NAMESPACE")
			if !found {
				installNs = "kubevirt-hyperconverged"
			}

			handlerPodsDir := path.Join(incidentDir, "namespaces", installNs, "core", "pods")
			if _, err := os.Stat(handlerPodsDir); os.IsNotExist(err) {
				installNs = "openshift-cnv"
				handlerPodsDir = path.Join(incidentDir, "namespaces", installNs, "core", "pods")
			}
			_, err := os.Stat(handlerPodsDir)
			Expect(err).ToNot(HaveOccurred(), "virt-handler pods directory should exist under the installation namespace")

			pods, err := os.ReadDir(handlerPodsDir)
			Expect(err).ToNot(HaveOccurred())

			var handlerPodDir string
			for _, pod := range pods {
				if strings.Contains(pod.Name(), "virt-handler") && pod.IsDir() {
					handlerPodDir = path.Join(handlerPodsDir, pod.Name())
					break
				}
			}
			Expect(handlerPodDir).ToNot(BeEmpty(), "should have a virt-handler pod directory")

			logFiles, err := os.ReadDir(handlerPodDir)
			Expect(err).ToNot(HaveOccurred())

			foundCurrentLog := false
			for _, f := range logFiles {
				if strings.HasSuffix(f.Name(), "-current.log") {
					foundCurrentLog = true
					info, statErr := f.Info()
					Expect(statErr).ToNot(HaveOccurred())
					Expect(info.Size()).To(BeNumerically(">", 0),
						fmt.Sprintf("log file %s should not be empty", f.Name()))
				}
			}
			Expect(foundCurrentLog).To(BeTrue(),
				"virt-handler pod directory should contain at least one *-current.log file")
		})

		It("should collect incident metrics", func() {
			metricsDir := path.Join(incidentDir, "incident-metrics")
			_, err := os.Stat(metricsDir)
			Expect(err).ToNot(HaveOccurred(), "incident-metrics directory should exist")

			metricsFile := path.Join(metricsDir, "incident-metrics.txt")
			info, err := os.Stat(metricsFile)
			Expect(err).ToNot(HaveOccurred(), "incident-metrics.txt should exist")
			Expect(info.Size()).To(BeNumerically(">", 0), "incident-metrics.txt should not be empty")

			content, err := os.ReadFile(metricsFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("# EOF"), "OpenMetrics file should end with # EOF")

			metadataFile := path.Join(metricsDir, "metrics-metadata.json")
			metaInfo, err := os.Stat(metadataFile)
			Expect(err).ToNot(HaveOccurred(), "metrics-metadata.json should exist")
			Expect(metaInfo.Size()).To(BeNumerically(">", 0))
		})

		It("should collect metrics for vm and node categories and match the registry", func() {
			metricsDir := path.Join(incidentDir, "incident-metrics")

			metadataBytes, err := os.ReadFile(path.Join(metricsDir, "metrics-metadata.json"))
			Expect(err).ToNot(HaveOccurred(), "metrics-metadata.json should be readable")

			var metadata struct {
				Total     int `json:"total"`
				Collected int `json:"collected"`
				Items     []struct {
					Slug     string `json:"slug"`
					Category string `json:"category"`
				} `json:"collected_metrics"`
				Skipped []struct {
					Slug     string `json:"slug"`
					Category string `json:"category"`
					Reason   string `json:"reason"`
				} `json:"skipped_metrics"`
			}
			Expect(json.Unmarshal(metadataBytes, &metadata)).To(Succeed())

			Expect(metadata.Collected+len(metadata.Skipped)).To(Equal(metadata.Total),
				"collected + skipped should account for every metric (none silently dropped)")

			// Cross-reference against incident_metrics.conf shipped in the archive
			confPath := path.Join(incidentDir, "incident-metrics", "incident_metrics.conf")
			confContent, confErr := os.ReadFile(confPath)
			Expect(confErr).ToNot(HaveOccurred(),
				"incident_metrics.conf should be shipped in the archive alongside metrics")

			expectedTotal := 0
			for _, line := range strings.Split(string(confContent), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "|", 4)
				if len(parts) >= 2 {
					expectedTotal++
				}
			}
			Expect(metadata.Total).To(Equal(expectedTotal),
				"metrics total in metadata should match number of metrics defined in incident_metrics.conf")

			vmMetrics := 0
			nodeMetrics := 0
			for _, m := range metadata.Items {
				switch m.Category {
				case "vm":
					vmMetrics++
				case "node":
					nodeMetrics++
				}
			}

			Expect(vmMetrics).To(BeNumerically(">=", 1),
				"should collect at least one VM metric from Prometheus")
			Expect(nodeMetrics).To(BeNumerically(">=", 1),
				"should collect at least one node metric from Prometheus")

			metricsContent, err := os.ReadFile(path.Join(metricsDir, "incident-metrics.txt"))
			Expect(err).ToNot(HaveOccurred())
			metricsStr := string(metricsContent)

			for _, m := range metadata.Items {
				Expect(metricsStr).To(ContainSubstring("# HELP "+m.Slug),
					fmt.Sprintf("metrics file should contain HELP line for collected metric %s/%s", m.Category, m.Slug))
				Expect(metricsStr).To(MatchRegexp(`# TYPE `+regexp.QuoteMeta(m.Slug)+` (gauge|counter|histogram|summary)`),
					fmt.Sprintf("metrics file should contain TYPE line for collected metric %s/%s", m.Category, m.Slug))
			}

			for _, s := range metadata.Skipped {
				logger.Printf("  metric %s/%s skipped: %s", s.Category, s.Slug, s.Reason)
			}

			logger.Printf("  metrics accounting: %d total (%d collected, %d skipped)",
				metadata.Total, metadata.Collected, len(metadata.Skipped))
		})

		It("should collect virsh and QEMU data when VM has not restarted", func() {

			vmsBaseDir := path.Join(incidentDir, "namespaces", incidentNs, "vms")
			if _, err := os.Stat(vmsBaseDir); os.IsNotExist(err) {
				logger.Printf("  vms/ directory absent — VM restarted after incident, virsh data intentionally skipped")
				Skip("VM restarted after incident — virsh/QEMU data not collected (expected)")
			}
			vmEntries, err := os.ReadDir(vmsBaseDir)
			Expect(err).ToNot(HaveOccurred())
			if len(vmEntries) == 0 {
				Skip("vms/ directory is empty — VM restarted after incident")
			}
			incidentVM := vmEntries[0].Name()
			logger.Printf("  detected VM from archive: %s", incidentVM)

			vmsDir := path.Join(vmsBaseDir, incidentVM)

			vmFiles, err := os.ReadDir(vmsDir)
			Expect(err).ToNot(HaveOccurred())

			virshSuffixes := []string{".dumpxml.xml", ".domblklist", ".domstats", ".domblkerror", ".domjobinfo", ".list"}
			for _, suffix := range virshSuffixes {
				found := false
				for _, f := range vmFiles {
					if strings.HasSuffix(f.Name(), suffix) {
						found = true
						info, statErr := f.Info()
						Expect(statErr).ToNot(HaveOccurred())
						Expect(info.Size()).To(BeNumerically(">", 0),
							fmt.Sprintf("virsh file %s should not be empty", f.Name()))
						logger.Printf("  virsh data present: %s (%d bytes)", f.Name(), info.Size())
						break
					}
				}
				Expect(found).To(BeTrue(),
					fmt.Sprintf("should have at least one file with suffix %s in vms/%s/", suffix, incidentVM))
			}

			// Serial console log — optional (not all VMs produce serial output)
			for _, f := range vmFiles {
				if strings.HasSuffix(f.Name(), ".serial-console.log") {
					info, statErr := f.Info()
					Expect(statErr).ToNot(HaveOccurred())
					Expect(info.Size()).To(BeNumerically(">", 0),
						"serial console log should not be empty if present")
					logger.Printf("  serial console log present: %s (%d bytes)", f.Name(), info.Size())
				}
			}

			// QEMU process status and cgroup stats — optional (QEMU PID may not be found)
			qemuOptional := []string{".qemu-proc-status", ".qemu-cgroup-stats"}
			for _, suffix := range qemuOptional {
				for _, f := range vmFiles {
					if strings.HasSuffix(f.Name(), suffix) {
						info, statErr := f.Info()
						Expect(statErr).ToNot(HaveOccurred())
						Expect(info.Size()).To(BeNumerically(">", 0),
							fmt.Sprintf("QEMU file %s should not be empty if present", f.Name()))
						logger.Printf("  QEMU data present: %s (%d bytes)", f.Name(), info.Size())
					}
				}
			}
		})

		It("should collect storage chain and namespace events", func() {

			eventsPath := path.Join(incidentDir, "namespaces", incidentNs, "events")
			info, err := os.Stat(eventsPath)
			Expect(err).ToNot(HaveOccurred(), "namespace events file should exist")
			Expect(info.Size()).To(BeNumerically(">", 0), "namespace events should not be empty")

			// PVCs — created by oc adm inspect under core/persistentvolumeclaims/
			pvcDir := path.Join(incidentDir, "namespaces", incidentNs, "core", "persistentvolumeclaims")
			if entries, err := os.ReadDir(pvcDir); err == nil && len(entries) > 0 {
				logger.Printf("  PVCs collected: %d", len(entries))
				for _, e := range entries {
					logger.Printf("    PVC: %s", e.Name())
				}
			} else {
				logger.Printf("  no PVCs found (VM may not use persistent storage)")
			}

			// PVs — created by oc adm inspect under cluster-scoped-resources/core/persistentvolumes/
			pvDir := path.Join(incidentDir, "cluster-scoped-resources", "core", "persistentvolumes")
			if entries, err := os.ReadDir(pvDir); err == nil && len(entries) > 0 {
				logger.Printf("  PVs collected: %d", len(entries))
			}

			// StorageClasses — under cluster-scoped-resources/storage.k8s.io/storageclasses/
			scDir := path.Join(incidentDir, "cluster-scoped-resources", "storage.k8s.io", "storageclasses")
			if entries, err := os.ReadDir(scDir); err == nil && len(entries) > 0 {
				logger.Printf("  StorageClasses collected: %d", len(entries))
			}
		})

		It("should not leak ServiceAccount tokens or bearer credentials into the archive", func() {
			// The Prometheus query functions use a ServiceAccount token. If set -x
			// fires at the wrong time or stderr is misdirected, the Authorization
			// header could leak into collected files or must-gather.log.
			tokenPatterns := []*regexp.Regexp{
				regexp.MustCompile(`Authorization:\s*Bearer\s+\S+`),
				regexp.MustCompile(`eyJhbG[A-Za-z0-9_-]{20,}`), // JWT prefix: base64("{"alg"...")
			}

			filesScanned := 0
			err := filepath.Walk(incidentDir, func(filePath string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if info.Size() == 0 || info.Size() > 50*1024*1024 {
					return nil
				}
				ext := strings.ToLower(filepath.Ext(filePath))
				if ext == ".gz" || ext == ".tar" || ext == ".zip" || ext == ".png" || ext == ".jpg" {
					return nil
				}

				content, readErr := os.ReadFile(filePath)
				if readErr != nil {
					return nil
				}
				filesScanned++

				relPath, _ := filepath.Rel(incidentDir, filePath)
				for _, pat := range tokenPatterns {
					match := pat.Find(content)
					Expect(match).To(BeNil(),
						fmt.Sprintf("SECURITY: file %s contains what looks like a leaked credential: %s",
							relPath, pat.String()))
				}
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			logger.Printf("  scanned %d files for credential leaks — none found", filesScanned)
		})

		It("should not collect cluster-wide control plane data", func() {
			crsDir := path.Join(incidentDir, "namespaces", "kubevirt-hyperconverged", "crs")
			_, err := os.Stat(crsDir)
			Expect(os.IsNotExist(err)).To(BeTrue(),
				"vm-incident should NOT collect control plane CRs (that's what full must-gather is for)")
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
			dirPath := path.Join(outputDir, file.Name())
			if checkVersionFile(dirPath) {
				return dirPath, nil
			}
		}
	}

	return "", errors.New("can't find the cluster directory")
}

// checkVersionFile checks if the current directory is ours.
// if there is a version file, and if it starts with "kubevirt/must-gather", then
// it was created by our must-gather image
func checkVersionFile(dirPath string) bool {
	versionFile, err := os.Open(path.Join(dirPath, "version"))
	if err != nil {
		return false
	}
	defer versionFile.Close()

	scanner := bufio.NewScanner(versionFile)
	if scanner.Scan() {
		return scanner.Text() == "kubevirt/must-gather"
	}

	return false
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
