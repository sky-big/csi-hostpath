#!/usr/bin/env bash

set -o errexit
set -o nounset

IMAGE=skybig/csi-hostpath:latest

# work dir
export WORK_DIR=$(cd `dirname $0`; cd ..; pwd)
mkdir -p ${WORK_DIR} || true

# build image
cd ${WORK_DIR}/build
# docker rmi -f ${IMAGE}
docker build -t=${IMAGE} -f Dockerfile .