#!/bin/bash -x

CONTENT_PATH=$1
# find all tar files recursively
for tarfile in $(find $CONTENT_PATH -name "*.tar" -type f)
do
  # try to import the tar file into containerd up to ten times
  for i in $(seq 10)
  do
    ctr -n k8s.io image import $tarfile --all-platforms
    cp $tarfile /var/lib/rancher/k3s/agent/images/
    if [ "$?" -eq 0 ]; then
      echo "Added: $tarfile (attempt $i) to /var/lib/rancher/k3s/agent/images/"
      break
    else
      if [ "$i" -eq 10 ]; then
        echo "failed to add : $tarfile (attempt $i) to /var/lib/rancher/k3s/agent/images/"
      fi
    fi
  done
done