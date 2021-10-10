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

By default, the VMs definitions won't be included, but only the VM Instances' custom resources.

In order to get data about other parts of the cluster (not specific to KubeVirt) you should
run `oc adm must-gather` (without passing a custom image). Run `oc adm must-gather -h` to see more options.

#### Targeted gathering
To collect the VM information, call directly the `gather_vms_details` command:
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- /usr/bin/gather_vms_details
```

The `gather_vms_details` command supports targeted gathering. By specifying a namespace, the command will only 
collect the VMs in this namespace. For example, collecting all the VM information in namespace "vm1":
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- NS=ns1 /usr/bin/gather_vms_details
```

By specifying the VM name in addition to the namespace, the `gather_vms_details` command will only collect the specific
VM information. For example, collecting the information of a specific VM called "testvm" in namespace "vm1":
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- NS=ns1 VM=testvm /usr/bin/gather_vms_details
```
***Note***: When collecting information for a specific VM, you must specify the namespace as well. Without the namespace,
the `gather_vms_details` command exits and prints an error message.

### Development
You can build the image locally using the Dockerfile included.

A `makefile` is also provided. To use it, you must pass a repository via the command-line using the variable `MUST_GATHER_IMAGE`.
You can also specify the registry using the variable `IMAGE_REGISTRY` (default is [quay.io](https://quay.io)) and the tag via `IMAGE_TAG` (default is `latest`).

The targets for `make` are as follows:
- `build`: builds the image with the supplied name and pushes it
- `docker-build`: builds the image but does not push it
- `docker-push`: pushes an already-built image

For example:
```sh
make build MUST_GATHER_IMAGE=kubevirt/must-gather
```
would build the local repository as `quay.io/kubevirt/must-gather:latest` and then push it.