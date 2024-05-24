#!/bin/bash

set -x

dockerd -s vfs &

# Check if `docker ps` command works, otherwise retry
while true; do
    if docker ps; then
        echo "Docker is running!"
        break
    else
        echo "Docker is not yet ready. Retrying in 5 seconds..."
        sleep 5
    fi
done

cd /root; gunzip localstack_3.4.0.tar.gz; docker load -i localstack_3.4.0.tar

pkill -9 dockerd

rm /var/run/docker.pid

$@