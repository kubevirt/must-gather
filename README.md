CNV must-gather
=================

`cnv-must-gather` is a tool built on top of [OpenShift must-gather](https://github.com/openshift/must-gather)
that expands its capabilities to gather Container-Native Virtualization information.

### Usage
```sh
oc adm must-gather --image=quay.io/jsaucier/cnv-must-gather
```

The command above will create a local directory with a dump of the CNV state.
Note that this command will only get data related to the CNV part of the OpenShift cluster.

You will get a dump of:
- The Hyperconverged Cluster Operator namespaces (and its children objects)
- All namespaces (and their children objects) that belong to any CNV resources
- All CNV CRD's definitions
- All namspaces that contains VMs
- All VMs definition

In order to get data about other parts of the cluster (not specific to CNV) you should
run `oc adm must-gather` (without passing a custom image). Run `oc adm must-gather -h` to see more options.
