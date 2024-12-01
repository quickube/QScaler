#!/bin/sh
set -o errexit

if [ -z "$(helm list | grep qscaler)" ]; then
  # 8. Install qscaler
  helm upgrade --install qscaler ./helm -f ./hack/values.yaml
else
  echo "Qscaler release exists, skipping installation"
fi
