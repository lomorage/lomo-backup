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

cd /root; gunzip localstack_3.4.0.tar.gz; docker load -i localstack_3.4.0.tar; rm localstack_3.4.0.tar

pkill -9 dockerd

while true; do
    if docker ps; then
        echo "Docker still is running. Retrying in 5 seconds..."
        sleep 5
    else
        echo "Docker is dead"
        break
    fi
done

rm -rf /var/run/docker*

$@