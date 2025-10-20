FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.21 AS go-builder

WORKDIR /go/src/github.com/kubevirt/must-gather/cmd
COPY cmd .
RUN (cd vmConvertor && go build -ldflags="-s -w" .)

FROM registry.ci.openshift.org/ocp/4.21:cli

RUN oc version

ENV INSTALLATION_NAMESPACE=kubevirt-hyperconverged

# For gathering data from nodes
RUN dnf update -y && \
    dnf install iproute tcpdump pciutils util-linux nftables rsync -y && \
    dnf clean all

COPY --from=go-builder /go/src/github.com/kubevirt/must-gather/cmd/vmConvertor/vmConvertor /usr/bin/

# Copy all collection scripts to /usr/bin
COPY collection-scripts/* /usr/bin/

# Copy node-gather resources to /etc
COPY node-gather /etc/

ENTRYPOINT /usr/bin/gather
