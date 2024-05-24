#!/bin/bash

sqlite3 ./lomob.db "delete from isos"; sqlite3 lomob.db "update files set iso_id=0"; rm *.iso

lomob iso create -s 30M

declare -A filesHash
filesHash["2019-04-03--2024-05-25.iso"]="54979da5dcdf0ff08c046dd97662677a7b186bb13ef700668e36899f2e0d1df5"
filesHash["2021-04-26--2021-04-26.iso"]="4a93ac387edc587bce1d5b1a51dd78358cd9b9275a02a90e5b7e433256f94444"
filesHash["2021-04-26--2021-07-31.iso"]="44434ef03763d476ed82602f7ee6991f40e2c698d37c7b7771fb39cd14fb3cb7"

# Iterate over the keys and values of the map
for key in "${!filesHash[@]}"; do
  expecValue="${filesHash[$key]}"
  value=$(sha256sum $key | awk -F' ' '{print $1}')

  if [ "$value" = "$expecValue" ]; then
    echo "$key has same file sha as expected"
  else
    echo "FAIL: $key has different file sha as expected"
  fi
done
