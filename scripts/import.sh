#!/bin/sh
set -x
CONTENT_PATH=$1
mkdir -p /var/lib/rancher/k3s/agent/images

find -L "$CONTENT_PATH" -name "*.tar" -type f | while read -r tarfile; do
  cp $tarfile /var/lib/rancher/k3s/agent/images
done