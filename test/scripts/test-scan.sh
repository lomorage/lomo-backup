#!/bin/bash

set -e

expectTotalDirs=58
expectTotalFiles=202

# make sure timestamp is correct
cd ../data
find . | while read -r file; do git log -1 --format="%ad" --date=iso "$file" | xargs -I{} touch -d {} "$file"; done
cd ../scripts

# try different parallel scan
for i in 10 5 1; do
  echo "scan directory with $i threads"
  rm ./lomob.db
  sqlite3 ./lomob.db < ../../common/dbx/schema/1.sql

  lomob scan -t $i ../data

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
done