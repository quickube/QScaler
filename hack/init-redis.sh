#!/bin/sh
set -o errexit

if [ -z "$(helm list | grep redis)" ]; then
  helm upgrade --install redis oci://registry-1.docker.io/bitnamicharts/redis \
  --set global.redis.password=Aa123456 \
  --set architecture=standalone \
  --set persistence=false
else
  echo "Redis release exists, skipping installation"
fi
