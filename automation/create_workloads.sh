#!/usr/bin/env bash

set -ex

CMD=oc

ANNOTATION_EXP="/^  namespace: ns.*$/a\  annotations:\n    vm.kubevirt.io/template: some-template"

# create 100 VMs
for ns in {001..005}; do
  sed "s#__NS__#${ns}#g" automation/ns.yaml | ${CMD} apply -f -
  for vm in {001..020}; do
    n=$(echo $vm | sed "s|^0*||")
    if [[ $(($n%2)) -eq 0 ]]; then
      sed -e "s#__NS__#${ns}#g" -e "s|##VM##|${vm}|g" -e "${ANNOTATION_EXP}" automation/vm.yaml >> "vms_ns${ns}.yaml"
    else
      sed -e "s#__NS__#${ns}#g" -e "s|##VM##|${vm}|g" automation/vm.yaml >> "vms_ns${ns}.yaml"
    fi

  done
  ${CMD} apply -f "vms_ns${ns}.yaml"

  # start 5 VMs
  if [[ ${ns} -le 5 ]]; then
    ${CMD} patch -n "ns${ns}" virtualmachine "testvm-ns${ns}-vm001" --type merge -p '{"spec":{"running":true}}'
  fi
done

# wait for the execution of the first 5 VMs
for ns in {001..005}; do
  ${CMD} wait -n "ns${ns}" vmi "testvm-ns${ns}-vm001" --for condition=Ready --timeout="300s"
done

${CMD} get vmis --all-namespaces
${CMD} get dvs --all-namespaces
