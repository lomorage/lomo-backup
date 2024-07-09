#!/bin/bash

set -e

#echo "Test list iso files in db"
#
#expectISOFiles="ID    Name                          Size       Status                   Region    Bucket    Files Count    Create Time            Local Hash
#1     2019-04-03--2024-05-25.iso    30.7 MB    Created, not uploaded                        113            2024-05-25 23:21:21    54979da5dcdf0ff08c046dd97662677a7b186bb13ef700668e36899f2e0d1df5
#2     2021-04-26--2021-04-26.iso    32.3 MB    Created, not uploaded                        3              2024-05-25 23:21:21    4a93ac387edc587bce1d5b1a51dd78358cd9b9275a02a90e5b7e433256f94444
#3     2021-04-26--2021-07-31.iso    41.9 MB    Created, not uploaded                        16             2024-05-25 23:21:21    44434ef03763d476ed82602f7ee6991f40e2c698d37c7b7771fb39cd14fb3cb7"
#
#value=$(lomob iso list)
#
#if [ "$value" = "$expectISOFiles" ]; then
#  echo "ISO file lists are same as expected"
#else
#  echo "ISO file lists are different"
#  echo "Expect:"
#  echo "$expectISOFiles"
#  echo "Actual:"
#  echo "$value"
#fi

echo "Test listing files not in iso yet"

lomob list files > /tmp/tmp

diff /tmp/tmp files_not_in_iso.txt

if [ $? -eq 0 ]; then
  echo "Files not in ISO are same as expected"
else
  echo "Files not in ISO are different"
  exit 1
fi

echo "Test dump iso files"

for name in "2019-04-03--2024-04-17" "2021-04-26--2021-04-26" "2021-04-26--2021-07-31";
do
  isoFile=${name}".iso"
  
  lomob iso dump $isoFile > /tmp/tmp
  
  diff /tmp/tmp files_${name}.txt
  
  if [ $? -eq 0 ]; then
    echo "Dump iso $isoFile are same as expected"
  else
    echo "Dump iso $isoFile are different"
    exit 1
  fi
done
