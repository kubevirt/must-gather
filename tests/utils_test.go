package tests_test

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "k8s.io/apimachinery/pkg/util/yaml"
    "os"
    "path"
    "strings"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func init() {
    var envSet bool
    kubeconfig, envSet = os.LookupEnv("KUBECONFIG")
    if !envSet {
        flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
        flag.Parse()

        if !path.IsAbs(kubeconfig) {
            wd, err := os.Getwd()
            if err != nil {
                panic(err)
            }
            kubeconfig = path.Join(wd, kubeconfig)
        }
    }

    clusterConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        panic(err)
    }

    clt, err := kubernetes.NewForConfig(clusterConfig)
    if err != nil {
        panic(err)
    }

    client = newTestClient(clt)
    if err = client.loadResourceTypes(); err != nil {
        panic(err)
    }
}

type testClient struct {
    *kubernetes.Clientset
    namespacedResources    resourceTypes
}

func newTestClient(client *kubernetes.Clientset) *testClient {
    return &testClient{
        Clientset:              client,
        namespacedResources:    make(map[string]resourceVersion),
    }
}

func (clt testClient) loadResourceTypes() error {
    clusterResourceTypesList, err := clt.Discovery().ServerPreferredResources()
    if err != nil {
        return err
    }

    for _, api := range clusterResourceTypesList {
        clt.namespacedResources.addApi(api)
    }

    return nil
}

func (clt testClient) getNamespacedResource(ctx context.Context, resourceType, group, namespace, name string) (*unstructured.Unstructured, error) {

    api, found := clt.namespacedResources[group]
    if !found {
        return nil, fmt.Errorf("can't find API Resource for %s", group)
    }

    if !stringInSlice(api.resources, resourceType) {
        return nil, fmt.Errorf("can't find Resource for %s", resourceType)
    }

    absUrl := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s", group, api.version, namespace, resourceType, name)

    var obj = new(unstructured.Unstructured)
    res, err := clt.RESTClient().Get().AbsPath(absUrl).DoRaw(ctx)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(res, obj)
    if err != nil {
        return nil, err
    }

    return obj, nil
}

func (clt testClient) getNonNamespacedResource(ctx context.Context, resourceType, group, name string) (*unstructured.Unstructured, error) {

    api, found := clt.namespacedResources[group]
    if !found {
        return nil, fmt.Errorf("can't find API Resource for %s", group)
    }

    if !stringInSlice(api.resources, resourceType) {
        return nil, fmt.Errorf("can't find Resource for %s", resourceType)
    }

    absUrl := fmt.Sprintf("/apis/%s/%s/%s/%s", group, api.version, resourceType, name)

    var obj = new(unstructured.Unstructured)
    res, err := clt.RESTClient().Get().AbsPath(absUrl).DoRaw(ctx)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(res, obj)
    if err != nil {
        return nil, err
    }

    return obj, nil
}

type resourceTypes map[string]resourceVersion

func (clusterResourceTypes resourceTypes) addApi(api *metav1.APIResourceList) {
    splitApiVersion := strings.Split(api.GroupVersion, "/")
    if len(splitApiVersion) == 2 {
        group := splitApiVersion[0]
        version := splitApiVersion[1]

        existing, found := clusterResourceTypes[group]
        if !found || compK8sVersions(existing.version, version) == version {
            resources := make([]string, len(api.APIResources))
            for i, resource := range api.APIResources {
                resources[i] = resource.Name
            }
            clusterResourceTypes[group] = resourceVersion{version: version, resources: resources}
        }
    }
}

type resourceVersion struct {
    version   string
    resources []string
}

func compK8sVersions(v1, v2 string) string {
    i := 0
    for ; i < len(v1) && i < len(v2); i++ {
        if v1[i] > v2[i] {
            return v1
        } else if v2[i] > v1[i] {
            return v2
        }
    }

    if len(v2) < len(v1) {
        return v2
    }

    return v1
}

func getObjectFromFile(reader io.Reader) (*unstructured.Unstructured, error) {
    objFromFile := new(unstructured.Unstructured)
    if err := yaml.NewYAMLOrJSONDecoder(reader, 1024).Decode(objFromFile); err != nil {
        return nil, err
    }

    return objFromFile, nil
}

func stringInSlice(slice []string, requested string) bool {
    for _, fromSlice := range slice {
        if requested == fromSlice {
            return true
        }
    }
    return false
}

func fileInDir(slice []os.DirEntry, file string) bool {
    for _, fromSlice := range slice {
        if file == path.Base(fromSlice.Name()) {
            return true
        }
    }
    return false
}

func BeAllTrueInBoolMap() *boolMapAllTrueMatcher {
    return new(boolMapAllTrueMatcher)
}

type boolMapAllTrueMatcher struct{}

func (boolMapAllTrueMatcher) Match(actual interface{}) (success bool, err error) {
    boolMap, ok := actual.(map[string]bool)
    if !ok {
        return false, errors.New("should be map[bool]string")
    }

    for _, value := range boolMap {
        if !value {
            return false, nil
        }
    }
    return true, nil
}
func (boolMapAllTrueMatcher) FailureMessage(actual interface{}) (message string) {
    return fmt.Sprintf("Expected\n\t%#v\nto be all true", actual)
}
func (boolMapAllTrueMatcher) NegatedFailureMessage(actual interface{}) (message string) {
    return fmt.Sprintf("Expected\n\t%#v\nto not be all true", actual)
}
