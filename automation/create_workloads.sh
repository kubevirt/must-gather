#!/usr/bin/env bash

set -ex

CMD=oc

LABEL_EXP="/^  namespace: ns.*$/a\  labels:\n    vm.kubevirt.io/template: some-template"

# create 100 VMs
for ns in {001..005}; do
  sed "s#__NS__#${ns}#g" automation/ns.yaml | ${CMD} apply -f -
  for vm in {001..020}; do
    EXP=
    n=$(echo $vm | sed "s|^0*||")
    if [[ $(($n%2)) -eq 0 ]]; then
      EXP="; ${LABEL_EXP}"
    fi
    sed -e "s#__NS__#${ns}#g; s|##VM##|${vm}|g${EXP}" automation/vm.yaml | ${CMD} apply -f -

  done

  # start 5 VMs
  if [[ ${ns} -le 5 ]]; then
    ${CMD} patch -n "ns${ns}" virtualmachine "testvm-ns${ns}-vm001" --type merge -p '{"spec":{"running":true}}'
  fi
done

# wait for the execution of the first 5 VMs
for ns in {001..005}; do
  found=false
  for tries in {1..5}; do
    if ${CMD} get -n "ns${ns}" vmi "testvm-ns${ns}-vm001"; then
      found=true
      break
    else
      echo "vmi ns${ns}/testvm-ns${ns}-vm001 does not exist yet. waiting 10 second"
      sleep 10
    fi
  done

  if [[ $found != "true" ]]; then
    echo "vmi ns${ns}/testvm-ns${ns}-vm001 was not created"
    exit 1
  fi

  ${CMD} wait -n "ns${ns}" vmi "testvm-ns${ns}-vm001" --for condition=Ready --timeout=300s
done

${CMD} get vmis --all-namespaces
${CMD} get dvs --all-namespaces
