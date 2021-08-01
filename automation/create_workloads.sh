#!/usr/bin/env bash

set -ex

CMD=oc

for ns in {1..5}; do
  sed "s#__NS__#${ns}#g" automation/vm.yaml | ${CMD} apply -f -
done


for ns in {1..5}; do
  ${CMD} wait -n "ns${ns}" vmi "testvm${ns}" --for condition=Ready --timeout="300s"
done

${CMD} get vmis --all-namespaces
${CMD} get dvs --all-namespaces
