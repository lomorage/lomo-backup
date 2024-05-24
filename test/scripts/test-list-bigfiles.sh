#!/bin/bash

echo "Test list big files in scan table"

expectBigFiles="Name                                                                                               Size
/go/src/github.com/lomorage/lomo-backup/test/data/static/video/Lomorage-tutorial.zh.mp4            21.2 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/android-frame/phone-share.gif    20.9 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/lomorage-setup.png                14.9 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/lomorage-mini.png                 14.6 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/video/Lomorage-tutorial.mp4               13.9 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/video/lomorage.mp4                        3.0 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/local.png                         2.5 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/web-gallery-en.png                1.6 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/local-en.png                      1.6 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/backup.png                        1.4 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/web-gallery.png                   1.2 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/backup-en.png                     1.1 MB
/go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy/web-upload.png                    1.0 MB"

value=$(lomob list bigfiles -s 1M)

if [ "$value" = "$expectBigFiles" ]; then
  echo "Big file lists are same as expected"
else
  echo "Big file lists are different"
  echo "Expect:"
  echo "$expectBigFiles"
  echo "Actual:"
  echo "$value"
fi
