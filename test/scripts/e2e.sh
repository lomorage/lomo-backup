#!/bin/bash

set -x

# test scan
./test-scan.sh

# test big file list
./test-list-bigfiles.sh

# test list scan dir
./test-list-dirs.sh

# test iso
./test-iso-create.sh