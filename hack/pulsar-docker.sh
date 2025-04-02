#!/usr/bin/env bash

set -e

ROOT=$(git rev-parse --show-toplevel)
pushd "${ROOT}" > /dev/null

IMAGE=terraform-provider-pulsar-standalone
CONTAINER=terraform-provider-pulsar-dev

case $1 in
run)
  docker build --platform=linux/amd64 -t ${IMAGE} -f hack/pulsarimage/Dockerfile hack/pulsarimage
  docker run --platform=linux/amd64 -d -p 6650:6650 -p 8080:8080 --name ${CONTAINER} ${IMAGE}
  until curl http://localhost:8080/admin/v2/tenants >/dev/null 2>&1; do
    sleep 5
    echo "Wait for pulsar service to be ready...$(date +%H:%M:%S)"
  done
  echo "Pulsar service is ready"

  echo "Uploading functions jar"
  docker exec ${CONTAINER} /pulsar/bin/pulsar-admin packages upload \
    --path /pulsar/examples/api-examples.jar \
    --description "api-examples" \
    function://public/default/api-examples@v1
  echo "functions jar uploaded"

  ;;
remove)
  docker rm -f ${CONTAINER}
  ;;
esac

popd > /dev/null
