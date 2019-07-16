KubeVirt must-gather
=================

`must-gather` is a tool built on top of [OpenShift must-gather](https://github.com/openshift/must-gather)
that expands its capabilities to gather KubeVirt information.

### Usage
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather
```

The command above will create a local directory with a dump of the KubeVirt state.
Note that this command will only get data related to the KubeVirt part of the OpenShift cluster.

You will get a dump of:
- The Hyperconverged Cluster Operator namespaces (and its children objects)
- All namespaces (and their children objects) that belong to any KubeVirt resources
- All KubeVirt CRD's definitions
- All namspaces that contains VMs
- All VMs definition

In order to get data about other parts of the cluster (not specific to KubeVirt) you should
run `oc adm must-gather` (without passing a custom image). Run `oc adm must-gather -h` to see more options.
