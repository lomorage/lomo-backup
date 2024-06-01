#!/bin/bash

sqlite3 ./lomob.db "delete from isos"; sqlite3 lomob.db "update files set iso_id=0"; rm *.iso *.iso.meta.txt

set -e

lomob iso create -s 30M --debug

#declare -A filesHash
#filesHash["2019-04-03--2024-04-17.iso"]="616a786cbffd5f81069554e8517b1376e5a7f7f293d1babda4e3cc8b2e390f22"
#filesHash["2021-04-26--2021-04-26.iso"]="8a0a2e091e5a5ea1ebfc260dcc1e058ee4ec0eb869837f323bf17ac6db6392c6"
#filesHash["2021-04-26--2021-07-31.iso"]="988cdf7c20a1606990e66015e62bf82ff4b1d44f8946c06253f4b089181100ce"
#
## Iterate over the keys and values of the map
#for key in "${!filesHash[@]}"; do
#  expecValue="${filesHash[$key]}"
#  value=$(sha256sum $key | awk -F' ' '{print $1}')
#
#  if [ "$value" = "$expecValue" ]; then
#    echo "$key has same file sha as expected"
#  else
#    echo "FAIL: $key has different file sha as expected"
#  fi
#done
