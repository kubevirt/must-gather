# KubeVirt must-gather

`must-gather` is a tool built on top of [OpenShift must-gather](https://github.com/openshift/must-gather)
that expands its capabilities to gather KubeVirt information.

## Usage
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather
```

The command above will create a local directory with a dump of the KubeVirt state.
Note that this command will only get data related to the KubeVirt part of the OpenShift cluster.

You will get a dump of:
- The Hyperconverged Cluster Operator namespaces (and its children objects)
- All namespaces (and their children objects) that belong to any KubeVirt resources
- All KubeVirt CRD's definitions

By default, the VMs definitions won't be included, but only the VM Instances' custom resources.

In order to get data about other parts of the cluster (not specific to KubeVirt) you should
run `oc adm must-gather` (without passing a custom image). Run `oc adm must-gather -h` to see more options.

### Flags

`must-gather` provides a series of options to select which information to
collect from the cluster. The tool will always collect all control-plane logs and information.
Optional collectors can be enabled with CLI options.


To run only the default collectors:
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- /usr/bin/gather
```

To collect all default information and VMs details:
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- /usr/bin/gather --vms_details
```

### Help Menu

At any time you can check the help menu for usage details of the KubeVirt must-gather

```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- /usr/bin/gather --help
```

```
Usage: oc adm must-gather --image=quay.io/kubevirt/must-gather -- /usr/bin/gather [params...]

  A client tool for gathering KubeVirt information in an OpenShift cluster

  Available options:

  > To see this help menu and exit use
  --help

  > The tool will always collect all control-plane logs and information.
  > This will include:
  > - apiservices
  > - cdi
  > - crds
  > - crs
  > - hco
  > - nodes
  > - ns
  > - resources
  > - ssp
  > - virtualmachines
  > - webhooks

  > You can also choose to enable optional collectors combining one
  > or more of the following parameters:
  --images
  --vms_details
```

### Parallelism
Some gathering activity can be done in parallel. Collecting resources one by one may be slow, and collecting too many 
resources in parallel may fail. By default, 5 processes are running in parallel, and the rest of the processes are 
waiting for running processes to complete. It is possible to change this default number of processes by setting the
`PROS` environment variable, but then, the default command must be specified as well, like this:

```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- PROS=7 \
   /usr/bin/gather
```

### Targeted gathering - VM information

To collect the default control plane information and VM detailed information you can append `--vms_details` command line flag:
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- /usr/bin/gather --vms_details
```

#### VMs in a Namespace
The `--vms_details` flag supports targeted gathering. By specifying a namespace, the command will only collect detailed VM logs for the VMs in this namespace (control plane logs are always collected). For example, collecting all the VM information in namespace "vm1":
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- NS=ns1 \
   /usr/bin/gather --vms_details
```

#### Specific VM
By specifying the VM name in addition to the namespace, the `--vms_details` flag will only collect the specific
VM information (control plane logs are always collected). For example, collecting the information of a specific VM called "testvm" in namespace "vm1":
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- NS=ns1 \
   VM=testvm \
   /usr/bin/gather --vms_details
```
***Note***: When collecting information for a specific VM, you must specify the namespace as well. Without the namespace,
the `gather --vms_details` command exits and prints an error message.

#### List of Specific VMs
The `VM` environment variable can also be a comma-seperated list of VM names (without a space). For example:
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- NS=ns1 \
   VM="testvm1,testvm34,testvm52,testvm74" \
   /usr/bin/gather --vms_details
```
#### Gather VM by Regex Expression
The `--vms_details` flag also support gathering VM with regex expression.

For example, suppose we have the following VMs in the cluster:
```
testvm1-1 testvm1-2 testvm1-3 testvm1-4 testvm1-5  
testvm1-6 testvm1-7 testvm1-8 testvm1-9 testvm1-10
testvm2-1 testvm2-2 testvm2-3 testvm2-4 testvm2-5 
testvm2-6 testvm2-7 testvm2-8 testvm2-9 testvm2-10
testvm3-1 testvm3-2 testvm3-3 testvm3-4 testvm3-5
testvm3-6 testvm3-7 testvm3-8 testvm3-9 testvm3-10
testvm4-1 testvm4-2 testvm4-3 testvm4-4 testvm4-5
testvm4-6 testvm4-7 testvm4-8 testvm4-9 testvm4-10
testvm5-1 testvm5-2 testvm5-3 testvm5-4 testvm5-5 
testvm5-6 testvm5-7 testvm5-8 testvm5-9 testvm5-10
```

If we want to read only VMs that starts with testvm2, testvm3 or testvm4, and that their postfix number is odd, we can use this regex expression to for that: `^testvm[2-4]-[0-9]*[1,3,5,7,9]$`.

Here is how to use it in the `--vms_details` flag, to search VMs by regex:
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   VM_EXP="^testvm[2-4]-[0-9]*[1,3,5,7,9]$" \
   /usr/bin/gather --vms_details
```

Here is how to use it in the `--vms_details` flag, to search VMs by regex in the `ns1` namespace:
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- NS=ns1 \
   VM_EXP="^testvm[2-4]-[0-9]*[1,3,5,7,9]$" \
   /usr/bin/gather --vms_details
```

***Note***: When collecting information using the `VM` variable, the command will ignore the `VM_EPR` variable. Do not use both of them together.


### Targeted gathering - Images information

It is possible to collect image, image-stream and image-stream-tags information using the `--images` flag:
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- /usr/bin/gather --images
```

The `--vms_details` and the `--images` flags support parallelism as well. To change the default number of processes of 5, add the
`PROS` environment variable. This is only works when not using the `NS` environment variable:
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- PROS=7 /usr/bin/gather --vms_details
```
Or
```sh
oc adm must-gather --image=quay.io/kubevirt/must-gather -- PROS=3 /usr/bin/gather --images
```

## Development
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
