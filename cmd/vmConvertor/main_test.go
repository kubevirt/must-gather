package main

import (
    "strconv"
    "sync"
    "testing"
    "time"

    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestWorkerPull(t *testing.T) {

    res := make(map[int]bool)
    vms := make([]unstructured.Unstructured, 10000)
    for i := 0; i < 10000; i++ {
        name := i + 1
        res[name] = false

        vms[i] = unstructured.Unstructured{
            Object: map[string]interface{}{
                "metadata": map[string]interface{}{
                    "name": strconv.Itoa(name),
                },
            },
        }
    }

    ch := make(chan int, 10)
    defer close(ch)

    go func() {
        for vmName := range ch {
            res[vmName] = true
        }
    }()

    wp := newWorkerPull(10, func(vm unstructured.Unstructured, wg *sync.WaitGroup) {
        defer wg.Done()
        name, err := strconv.Atoi(vm.GetName())
        if err != nil {
            return
        }
        ch <- name
    })
    defer wp.close()

    wg := &sync.WaitGroup{}
    wg.Add(10000)

    for _, vm := range vms {
        wp.execute(vm, wg)
    }

    wg.Wait()
    time.Sleep(time.Second)

    for name, touched := range res {
        if !touched {
            t.Errorf("vm %d was not toughted", name)
        }
    }
}

func TestGetVmIdentity_CustomVM(t *testing.T) {
    ns, vm, vmType := getVmIdentity(
        unstructured.Unstructured{
            Object: map[string]interface{}{
                "metadata": map[string]interface{}{
                    "name":      "vmName",
                    "namespace": "nsName",
                    "labels": map[string]interface{}{
                        "test1": "test1",
                        "test2": "test2",
                    },
                },
            },
        })

    if ns != "nsName" {
        t.Errorf(`namespace shold be "nsName" but it' "%s"'`, ns)
    }
    if vm != "vmName" {
        t.Errorf(`vm name shold be "nsName" but it' "%s"'`, vm)
    }
    if vmType != "custom" {
        t.Errorf(`vmType shold be "custom" but it' "%s"'`, vmType)
    }
}

func TestGetVmIdentity_TemplateBasedVM(t *testing.T) {
    ns, vm, vmType := getVmIdentity(
        unstructured.Unstructured{
            Object: map[string]interface{}{
                "metadata": map[string]interface{}{
                    "name":      "vmName",
                    "namespace": "nsName",
                    "labels": map[string]interface{}{
                        "test1": "test1",
                        "test2": "test2",
                        "vm.kubevirt.io/template": "a template",
                    },
                },
            },
        })

    if ns != "nsName" {
        t.Errorf(`namespace shold be "nsName" but it' "%s"'`, ns)
    }
    if vm != "vmName" {
        t.Errorf(`vm name shold be "nsName" but it' "%s"'`, vm)
    }
    if vmType != "template-based" {
        t.Errorf(`vmType shold be "template-based" but it' "%s"'`, vmType)
    }
}
