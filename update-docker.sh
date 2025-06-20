#! /bin/bash

docker build -t go-pod-server:$1 .
docker save go-pod-server:$1 | sudo k3s ctr images import -