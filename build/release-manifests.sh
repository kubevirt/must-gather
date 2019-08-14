#!/usr/bin/env bash
#
# Copyright 2019 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

node_image=${1:-node-gather}
bridge_marker_image_repo=${2}
separator="/"

cd node-gather
for template in *.in; do
    name=$(basename ${template%.in})
    if [ -z $bridge_marker_image_repo ]; then
        separator=""
    fi
    sed \
        -e "s#\${NODE_GATHER_IMAGE}#${node_image}#g" \
        -e "s#\${NODE_GATHER_IMAGE_REPO}#${bridge_marker_image_repo}${separator}#g" \
        ${template} > ${name}
done



