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
- Per-node system and storage diagnostics (dmesg, dmidecode, sysctl, NFS config, PSI counters, kernel red-flag scan)
- CNV guest events (GuestPanicked, LivenessProbeFailed) across VM namespaces
- Prometheus instant metrics (cluster utilization, VM phases, storage latencies, NFS counters)
- ClusterRole definitions (cluster-reader posture, KubeVirt RBAC baseline)
- Windows worker node posture data (when Windows nodes are present)

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
  > - instancetypes
  > - virtualization
  > - cnv_events
  > - prometheus_instant
  > - clusterroles
  > - windows_nodes

  > You can also choose to enable optional collectors combining one
  > or more of the following parameters:
  --images
  --vms_details

  > Incident collection for a specific VM at a known time.
  > Unlike a full must-gather, this collects ONLY data pertinent to the
  > incident: right VM, right node, right time window. No cluster-wide
  > noise. Timeboxed to 10 minutes.
  --vm-incident --incident-time=<RFC3339 timestamp>
    Requires NS and VM environment variables. Skips all other collectors.
    Example:
      oc adm must-gather --image=quay.io/kubevirt/must-gather \
        -- NS=myns VM=myvm \
        /usr/bin/gather --vm-incident --incident-time=2026-06-06T06:06:00Z
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


### VM incident collection

When a VM crashes (BSOD, kernel panic, I/O hang), the information needed for root-cause
analysis is typically scattered across four separate collection tools:

| What you need | Where it lives today |
|---|---|
| VM definitions, virt-launcher logs, virsh state | CNV must-gather (`--vms_details`) |
| Node journal, kubelet logs, ClusterOperators | OCP must-gather |
| dmesg, dmidecode, NFS config, sysctl, PSI counters | sosreport on the node |
| CPU/memory/storage/network metrics over time | Manual Prometheus queries or screenshots |

Collecting all four can take hours, produces gigabytes of cluster-wide data, and still
requires a support engineer to manually correlate the right node, the right time window,
and the right VM across all of them. Logs are often captured from "now" rather than from
when the incident occurred, missing the relevant entries entirely.

The `--vm-incident` flag replaces this with a **single command** that collects only the data
pertinent to one VM incident at a known time. You provide the VM name, namespace, and when
the incident happened. The tool automatically identifies the right node (via Prometheus
`kubevirt_vmi_info`, so it works even if the VM has since migrated), scopes all logs and
metrics to the incident time window (24 h before to 2 h after), and produces one small,
focused archive that a support engineer can review in minutes.

```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- NS=ns1 VM=myvm \
   /usr/bin/gather --vm-incident \
   --incident-time=2026-06-06T06:06:00Z
```

#### What gets collected

Everything below is scoped to the incident node and time window — no cluster-wide noise.

**Node logs and kernel diagnostics** (`nodes/<node>/`)
- Full journal and kubelet logs (time-scoped to 24 h before → 2 h after)
- `dmesg` — kernel ring buffer
- Kernel red-flag scan — grep of `journalctl -k` for OOM kills, NFS errors, QEMU crashes,
  blocked tasks, and I/O stalls over the last 7 days

**Hardware and system configuration** (`nodes/<node>/`)
- `dmidecode` — CPU microcode revision, BIOS version, memory DIMM layout
- `sysctl -a` — all kernel tunables
- `chronyc sources tracking` — NTP synchronization state
- `ip a` — network interface configuration
- Tuned profile and modprobe config

**Storage and I/O diagnostics** (`nodes/<node>/`)
- `/proc/self/mountstats` — NFS latency, retransmits, per-op timings
- `/proc/diskstats` — block device I/O counters
- `/proc/pressure/{cpu,memory,io}` — PSI (Pressure Stall Information) counters
- `df -h` — filesystem usage
- `/proc/mounts` — mount table
- NFS client config (`nfs.conf`, module parameters, modprobe rules)

**Cluster health snapshot** (`cluster-scoped-resources/`)
- ClusterOperators, nodes overview, MachineConfigPools
- `oc adm top node` for the incident node
- KubeVirt / CNV version (`kubevirt_version`)
- All VMIs running on the incident node at incident time (`vmis_on_incident_node`) —
  queried from Prometheus for noisy-neighbor analysis

**VM and storage chain** (`namespaces/<ns>/`)
- VM and VMI definitions (via `oc adm inspect`)
- PVCs used by the VM, backing PVs, StorageClasses
- DataVolumes (CDI import/clone status and conditions)
- VolumeAttachments for the incident node
- Namespace events (FailedAttach, FailedMount, etc.)

**Pod logs** (`namespaces/<ns>/core/pods/`)
- virt-launcher pod: all containers, current and previous logs (captures crash output)
- virt-handler pod on the incident node: current and previous logs

**Live VM state** (`namespaces/<ns>/vms/<vm>/`) — only if the VM has NOT restarted since the
incident; skipped automatically when the pod postdates the incident, since that data would
reflect the new instance, not the crash:
- `virsh dumpxml` — full libvirt domain XML
- `virsh domblklist` — block device mapping
- `virsh domjobinfo` — migration/save job state
- `virsh domstats` — live per-device block I/O, network stats, balloon info, CPU time
- `virsh domblkerror` — block device error state (QEMU uses `werror=stop, rerror=stop`)
- `virsh list --all` — domain state
- Guest serial console log (`virt-serial0-log`) — kernel panic text, BSOD codes
- QEMU log files from `/var/log/libvirt/qemu/` and runtime log directories
- QEMU process `/proc/<pid>/status` — VmRSS, VmPeak, voluntary/involuntary context switches
- QEMU cgroup stats — `memory.current`, `memory.max`, `memory.events` (OOM kill count),
  `cpu.stat` (throttled time), `cpu.max`

**Performance metrics** (`incident-metrics/`)
- 26 Prometheus metrics exported in OpenMetrics format over the incident window (30 s step),
  covering VM, node, storage, and alert categories:

  | Category | Metrics |
  |---|---|
  | VM | CPU usage rate, resident memory, swap-in traffic, network TX/RX bytes and errors, disk read/write bytes and latency |
  | Node | CPU usage, available memory, CPU/memory/I/O pressure (PSI), disk I/O utilization, 1-min load average, network receive errors |
  | Storage | PV usage %, volume mount duration, volume access-control duration |
  | Alerts | All firing KubeVirt alerts at incident time |

- `metrics-metadata.json` — which metrics were collected, which were skipped (with reasons),
  time window, and query step

**Incident summary** (`incident-summary.yaml`)
- Lists the VM, namespace, incident time, collection window, node (and how it was discovered),
  elapsed time, and itemized lists of what was collected and skipped with reasons

#### Node discovery

The node is identified via a Prometheus query
(`last_over_time(kubevirt_vmi_info{...}[24h])`) at incident time. This works even if the VM
has since live-migrated or restarted on a different node. If the monitoring stack is
unavailable, the tool falls back to the current VMI's `status.nodeName` with a warning.

#### Output structure

Output is written to standard must-gather paths (`nodes/`, `namespaces/`,
`cluster-scoped-resources/`) so tools like [omc](https://github.com/gmeghnag/omc) can parse
the archive normally.

#### Timeout

The entire collection is timeboxed to 10 minutes by default (configurable via
`INCIDENT_TIMEOUT`):

```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- NS=ns1 VM=myvm \
   INCIDENT_TIMEOUT=180 \
   /usr/bin/gather --vm-incident \
   --incident-time=2026-06-06T06:06:00Z
```

#### Backfilling metrics into Prometheus or VictoriaMetrics

The `incident-metrics/incident-metrics.txt` file is in standard
[OpenMetrics](https://openmetrics.io/) format. You can import it into a local Prometheus or
VictoriaMetrics instance to query and graph the incident timeline with Grafana.

**Prometheus** (requires `promtool` from the Prometheus distribution):
```sh
promtool tsdb create-blocks-from openmetrics incident-metrics.txt
```
This creates TSDB blocks in a `data/` directory that can be copied into your Prometheus data
directory. After restarting Prometheus, the historical metrics become queryable.

**VictoriaMetrics**:
```sh
curl -X POST 'http://localhost:8428/api/v1/import/prometheus' \
  --data-binary @incident-metrics.txt
```

#### Workflow

The incident collection is designed as a first-response tool. If it surfaces the root cause,
no further collection is needed. If it narrows the problem but more context is required, the
customer can then run a full CNV must-gather, OCP must-gather, or sosreports — but the
support engineer already knows where to look.

### VirtIO driver version check

Check whether Windows VMs are running outdated VirtIO drivers by comparing
installed versions (queried via the QEMU guest agent) against the baseline
shipped with the cluster's virtio-win container disk.

Only Windows VMs are checked. The OS type is detected via `guest-get-osinfo`,
so Linux VMs are skipped even if they have a guest agent installed. VMs without
the QEMU guest agent are also skipped.

Check a specific VM:
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- NS=default VM=myvm \
   /usr/bin/gather --virtio_win_driver_check=default/myvm
```

Check all running Windows VMs:
```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- /usr/bin/gather \
   --virtio_win_driver_check
```

The driver check also runs automatically as part of `--vm-incident`.

#### Output

Per-VM results follow the standard must-gather output format under
`namespaces/<ns>/vms/<vm>/`:

| File | Description |
|---|---|
| `virtio-win-baseline.json` | Virtio-win container disk image reference and expected driver version |
| `virtio-win-summary.json` | Cluster-wide totals: VMs checked, outdated, skipped |
| `namespaces/<ns>/vms/<vm>/virtio-driver-report.json` | Per-driver comparison with status (OK, OUTDATED, NEWER) |
| `namespaces/<ns>/vms/<vm>/virtio-driver-installed.json` | Raw guest agent device data (VirtIO devices only) |

#### Timeout

The driver check is timeboxed to 10 minutes by default (configurable via
`DRIVER_CHECK_TIMEOUT`):

```sh
oc adm must-gather \
   --image=quay.io/kubevirt/must-gather \
   -- DRIVER_CHECK_TIMEOUT=120 \
   /usr/bin/gather \
   --virtio_win_driver_check
```

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
