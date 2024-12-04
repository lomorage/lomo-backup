#!/bin/bash

set -e
docker run -p 4566:4566 -p 4510-4559:4510-4559 localstack/localstack:3.4.0 &

max_retry=10
retry_count=0
while [ $retry_count -lt $max_retry ]; do
    if curl 127.0.0.1:4566; then
        echo "Localstack is running!"
        break
    else
        echo "Localstack is not yet ready. Retrying in 5 seconds..."
        sleep 5
    fi
    retry_count=$((retry_count + 1))
done

if [ $retry_count -eq $max_retry ]; then
    echo "Max retry limit reached. Exiting..."
    exit 1
fi