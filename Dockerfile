FROM golang:1.19.2 AS go-builder
WORKDIR /go/src/github.com/kubevirt/must-gather/cmd
COPY cmd .
RUN (cd vmConvertor && go build -ldflags="-s -w" .)

FROM quay.io/openshift/origin-must-gather:4.16 as builder

FROM quay.io/centos/centos:stream8

ENV INSTALLATION_NAMESPACE kubevirt-hyperconverged

# For gathering data from nodes
RUN dnf update -y && \
    dnf install iproute tcpdump pciutils util-linux nftables rsync -y && \
    dnf clean all

COPY --from=builder /usr/bin/oc /usr/bin/oc
COPY --from=go-builder /go/src/github.com/kubevirt/must-gather/cmd/vmConvertor/vmConvertor /usr/bin/

# Copy all collection scripts to /usr/bin
COPY collection-scripts/* /usr/bin/

# Copy node-gather resources to /etc
COPY node-gather /etc/

ENTRYPOINT /usr/bin/gather
