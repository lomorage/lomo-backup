#!/bin/bash

set -e

echo "Test upload ISO files into aws with raw file"

sqlite3 lomob.db "update files set iso_id=0; update isos set upload_key='', upload_id='', hash_remote='', status=1; delete from parts"

export LOCALSTACK_ENDPOINT="http://localhost:4566"
export LOMOB_MASTER_KEY=1234
export AWS_ACCESS_KEY_ID=dummy
export AWS_SECRET_ACCESS_KEY=dummy
export AWS_DEFAULT_REGION=us-east-1

#LOCALSTACK_ENDPOINT="http://localhost:4566" LOMOB_MASTER_KEY=1234 AWS_ACCESS_KEY_ID=dummy AWS_SECRET_ACCESS_KEY=dummy AWS_DEFAULT_REGION=us-east-1 lomob iso upload 2021-04-26--2021-07-31.iso
#lomob iso upload --no-encrypt 2019-04-03--2024-04-17.iso 2021-04-26--2021-04-26.iso 2021-04-26--2021-07-31.iso

for isoFile in 2019-04-03--2024-04-17.iso 2021-04-26--2021-04-26.iso 2021-04-26--2021-07-31.iso; do
  # clean the left contents
  curl -X DELETE http://localhost:4566/lomorage/$isoFile
  curl -X DELETE http://localhost:4566/lomorage/$isoFile.meta.txt

  lomob iso upload --no-encrypt $isoFile

  # download and compare
  curl -s -o /tmp/tmp.data http://localhost:4566/lomorage/$isoFile
  expectSHA=$(sha256sum $isoFile | awk -F' ' '{print $1}')
  actualSHA=$(sha256sum /tmp/tmp.data | awk -F' ' '{print $1}')

  if [ "$expectSHA" = "$actualSHA" ]; then
    echo "Upload $isoFile success"
  else
    echo "Upload $isoFile fail"
    exit 1
  fi

  # compare meta with expect files
  diff $isoFile.meta.txt files_`basename $isoFile .iso`.txt
  if [ $? -eq 0 ]; then
    echo "$isoFile metadata file is same as expected"
  else
    echo "$isoFile metadata file is different"
    exit 1
  fi

  # compare meta with the one uploaded
  curl -s -o /tmp/tmp.meta http://localhost:4566/lomorage/$isoFile.meta.txt
  expectSHA=$(sha256sum $isoFile.meta.txt | awk -F' ' '{print $1}')
  actualSHA=$(sha256sum /tmp/tmp.meta | awk -F' ' '{print $1}')

  if [ "$expectSHA" = "$actualSHA" ]; then
    echo "$isoFile metadata file is uploaded success as well"
  else
    echo "$isoFile metadata file upload fail"
    exit 1
  fi

  # same file upload again will skip
  lomob iso upload --no-encrypt $isoFile > /tmp/tmp

  diff /tmp/tmp duplicate_upload_`basename $isoFile .iso`.txt

  if [ $? -eq 0 ]; then
    echo "Upload $isoFile again works as expected"
  else
    echo "Upload $isoFile again fail"
    exit 1
  fi
done