#!/usr/bin/env bash

set -ex

CMD=oc

LABEL_EXP="/^  namespace: ns.*$/a\  labels:\n    vm.kubevirt.io/template: some-template"

for ns in {001..005}; do
  for vm in {001..020}; do
    EXP=
    n=$(echo $vm | sed "s|^0*||")
    if [[ $(($n%2)) -eq 0 ]]; then
      EXP="; ${LABEL_EXP}"
    fi
    sed -e "s#__NS__#${ns}#g; s|##VM##|${vm}|g${EXP}" automation/vm.yaml | ${CMD} delete -f -

  done

  sed "s#__NS__#${ns}#g" automation/ns.yaml | ${CMD} delete -f -
done
