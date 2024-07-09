#!/bin/bash

set -e

expectTotalDirs=22
expectTotalFiles=118

rm ./lomob.db *.iso
sqlite3 ./lomob.db < ../../common/dbx/schema/1.sql

echo "scan data/content"
lomob scan -t 1 ../data/content

echo "scan data/static"
lomob scan -t 1 ../data/static


data=`sqlite3 ./lomob.db "select count(*) from dirs"`
if [ "$data" = "$expectTotalDirs" ]; then
  echo "Number of scanned dirs pass"
else
  echo "Number of scanned dirs is $data while expect $expectTotalDirs, failed!"
  exit 1
fi

data=`sqlite3 ./lomob.db "select count(*) from files"`
if [ "$data" = "$expectTotalFiles" ]; then
  echo "Number of scanned files pass"
else
  echo "Number of scanned file is $data while expect $expectTotalFiles, failed!"
  exit 1
fi

echo "start generate iso includes 2 root directory"

lomob iso create -s 30M --debug

echo "start compare iso output"
for name in "2019-04-03--2024-04-17" "2021-04-26--2021-04-26" "2021-04-26--2021-07-31";
do
  isoFile=${name}".iso"
  
  lomob iso dump $isoFile > /tmp/tmp
  
  diff /tmp/tmp files_2_dirs_${name}.txt
  
  if [ $? -eq 0 ]; then
    echo "Dump iso $isoFile are same as expected"
  else
    echo "Dump iso $isoFile are different"
    exit 1
  fi
done
