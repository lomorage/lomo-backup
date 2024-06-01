#!/bin/bash

set -ex

rm -f lomob.db; sqlite3 ./lomob.db < ../../common/dbx/schema/1.sql

# test scan
./test-scan.sh

# test big file list
./test-list-bigfiles.sh

# test list scan dir
./test-list-dirs.sh

# test iso
./test-iso-create.sh
./test-iso-list.sh

# test upload raw
#./test-upload-aws-raw.sh

# test upload encrypt file
#./test-upload-aws-encrypt.sh
