package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

var baseDir string

const numWorkers = 100

func main() {
	baseDir = getBaseDir()

	client, err := getClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	vmResource := schema.GroupVersionResource{Group: "kubevirt.io", Version: "v1", Resource: "virtualmachines"}

	list, err := client.Resource(vmResource).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("failed to read the VMs from the cluster", err)
		os.Exit(1)
	}

	vmsSlice := list.Items
	if len(vmsSlice) == 0 {
		fmt.Println("No VM found")
		os.Exit(0)
	}

	vmChannel := make(chan unstructured.Unstructured, numWorkers)
	wp := newWorkerPull(numWorkers, handleOneVM)
	defer wp.close()

	wg := &sync.WaitGroup{}
	wg.Add(len(vmsSlice))

	go func() {
		for vm := range vmChannel {
			wp.execute(vm, wg)
		}
	}()

	for _, vm := range vmsSlice {
		vmChannel <- vm
	}

	wg.Wait()
	close(vmChannel)
}

func getClient() (dynamic.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		if err != nil {
			return nil, fmt.Errorf("can't get kubeconfig; %w", err)
		}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can't create kubernetes client; %w", err)
	}

	return client, err
}

func handleOneVM(vm unstructured.Unstructured, wg *sync.WaitGroup) {
	defer wg.Done()

	ns, vmName, vmType := getVmIdentity(vm)

	dir, err := createOutputDir(ns, vmType)
	if err != nil {
		log.Println("can't create directory", dir, ";", err)
		return
	}

	if metadata, ok := vm.Object["metadata"].(map[string]interface{}); ok {
		delete(metadata, "managedFields")
	}
	//
	//delete(vm.Object["metadata"].(map[string]interface{}), "managedFields")

	vmYaml, err := yaml.Marshal(vm.Object)
	if err != nil {
		log.Println("can't convert vm to yaml;", err)
		return
	}

	fileName := path.Join(dir, vmName+".yaml")
	writeYamlVmFile(fileName, vmYaml)
}

func createOutputDir(ns string, vmType string) (string, error) {
	dir := path.Join(baseDir, "namespaces", ns, "kubevirt.io", "virtualmachines", vmType)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", err
	}
	return dir, nil
}

func writeYamlVmFile(fileName string, vmYaml []byte) {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Println("can't create file", fileName, ";", err)
		return
	}

	defer func() { _ = file.Close() }()
	if _, err = file.Write(vmYaml); err != nil {
		log.Println("failed to write", vmYaml, err)
	}
}

func getVmIdentity(vm unstructured.Unstructured) (string, string, string) {
	ns := vm.GetNamespace()
	vmName := vm.GetName()
	vmType := "custom"

	labels := vm.GetLabels()
	for k := range labels {
		if strings.HasPrefix(k, "vm.kubevirt.io/template") {
			vmType = "template-based"
			break
		}
	}

	return ns, vmName, vmType
}

func getBaseDir() string {
	baseDir, found := os.LookupEnv("BASE_COLLECTION_PATH")
	if !found {
		fmt.Println("the BASE_COLLECTION_PATH environment variable is not set")
		os.Exit(1)
	}

	return baseDir
}

// workerPull is a pull of workers for goroutines. The caller of the goroutine loans a worker from the pull before
// calling the goroutine. If there is available worker, the goroutine is called. if not, the loan request is blocked
// until there will be an available worker.
// When the goroutine exists, it returns a worker to the pull, and then the blocked call is released.
type workerPull struct {
	pull   chan struct{}
	worker func(vm unstructured.Unstructured, wg *sync.WaitGroup)
}

func newWorkerPull(size int, worker func(vm unstructured.Unstructured, wg *sync.WaitGroup)) workerPull {
	wp := make(chan struct{}, size)
	return workerPull{pull: wp, worker: worker}
}

func (wp workerPull) execute(vm unstructured.Unstructured, wg *sync.WaitGroup) {
	wp.pull <- struct{}{} // loan request; since the wp.pull channel is with limited size, if the channel is full, the request is blocked
	go func() {
		defer func() { <-wp.pull }() // return worker; read one worker from the wp.pull channel, to free one worker for the next request
		wp.worker(vm, wg)
	}()
}

func (wp workerPull) close() {
	close(wp.pull)
}
