#!/bin/bash

echo "Test list big files in scan table"

lomob list bigfiles -s 1M > /tmp/tmp

diff /tmp/tmp files_big.txt

if [ $? -eq 0 ]; then
  echo "Big file lists are same as expected"
else
  echo "Big file lists are different"
  exit 1
fi
