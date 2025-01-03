#!/bin/sh
set -o errexit

curl -sSL https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml \
    | yq '.spec.template.spec.containers[0].args += "--kubelet-insecure-tls"' - \
    | kubectl apply -f -