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

#cd /root; gunzip localstack_3.4.0.tar.gz; docker load -i localstack_3.4.0.tar; rm localstack_3.4.0.tar

#docker run --rm -d -p 4566:4566 -p 4510-4559:4510-4559 localstack/localstack:3.4.0 &

#while true; do
#    if curl localhost:4566; then
#        echo "Localstack is running!"
#        break
#    else
#        echo "Localstack is not yet ready. Retrying in 5 seconds..."
#        sleep 5
#    fi
#done