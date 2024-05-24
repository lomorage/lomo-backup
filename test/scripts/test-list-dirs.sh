#!/bin/bash

echo "Test list scan dirs"

expectDirs="/go/src/github.com/lomorage/lomo-backup/test/data
├── [     2024-05-25]  archetypes
├── [     2024-05-25]  content
│   └── [     2024-05-25]  blog
├── [     2024-05-25]  layouts
├── [     2024-05-25]  public
├── [     2024-05-25]  resources
│   └── [     2024-05-25]  _gen
│       └── [     2024-05-25]  assets
│           ├── [     2024-05-25]  css
│           │   └── [     2024-05-25]  css
│           └── [     2024-05-25]  js
│               └── [     2024-05-25]  js
├── [     2024-05-25]  static
│   ├── [     2024-05-25]  img
│   │   ├── [     2024-05-25]  blog
│   │   │   ├── [     2024-05-25]  android-frame
│   │   │   ├── [     2024-05-25]  covid19
│   │   │   ├── [     2024-05-25]  flyio
│   │   │   ├── [     2024-05-25]  image-quality
│   │   │   ├── [     2024-05-25]  import_my_cloud
│   │   │   ├── [     2024-05-25]  migrate_from_pi_win
│   │   │   ├── [     2024-05-25]  mypains
│   │   │   ├── [     2024-05-25]  photoprism
│   │   │   ├── [     2024-05-25]  raspberrypi-hd
│   │   │   ├── [     2024-05-25]  set_wd_as_backup
│   │   │   ├── [     2024-05-25]  transfer_pc
│   │   │   └── [     2024-05-25]  win-docker
│   │   ├── [     2024-05-25]  buy
│   │   └── [     2024-05-25]  links
│   └── [     2024-05-25]  video
└── [     2024-05-25]  themes
    └── [     2024-05-25]  evie-hugo
        ├── [     2024-05-25]  archetypes
        ├── [     2024-05-25]  assets
        │   ├── [     2024-05-25]  js
        │   │   └── [     2024-05-25]  libraries
        │   └── [     2024-05-25]  sass
        │       ├── [     2024-05-25]  base
        │       ├── [     2024-05-25]  components
        │       ├── [     2024-05-25]  elements
        │       └── [     2024-05-25]  utils
        ├── [     2024-05-25]  i18n
        ├── [     2024-05-25]  images
        ├── [     2024-05-25]  layouts
        │   ├── [     2024-05-25]  _default
        │   ├── [     2024-05-25]  partials
        │   └── [     2024-05-25]  shortcodes
        ├── [     2024-05-25]  resources
        │   └── [     2024-05-25]  _gen
        │       └── [     2024-05-25]  assets
        │           ├── [     2024-05-25]  css
        │           │   └── [     2024-05-25]  css
        │           └── [     2024-05-25]  js
        │               └── [     2024-05-25]  js
        └── [     2024-05-25]  static
            └── [     2024-05-25]  images"

value=$(lomob list dirs)

if [ "$value" = "$expectDirs" ]; then
  echo "Scan dir lists are same as expected"
else
  echo "Scan dir lists are different"
  echo "Expect:"
  echo "$expectDirs"
  echo "Actual:"
  echo "$value"
fi

# table view
expectDirs="File Counts    Total File Size    Mod Time               Path
7              15.6 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data
1              84 B               2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/archetypes
10             37.8 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/content
20             92.5 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/content/blog
1              182 B              2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/layouts
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/public
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/resources
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/css
4              42.5 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/css/css
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/js
2              28.4 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/js/js
10             1.6 MB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static
12             77.9 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img
1              77.9 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog
3              21.6 MB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/android-frame
4              610.0 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/covid19
1              9.8 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/flyio
4              393.1 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/image-quality
5              158.6 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/import_my_cloud
3              87.8 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/migrate_from_pi_win
2              128.9 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/mypains
1              25.6 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/photoprism
1              238.2 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/raspberrypi-hd
5              223.6 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/set_wd_as_backup
10             535.2 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/transfer_pc
3              140.4 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/win-docker
14             42.1 MB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy
6              374.1 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/links
3              38.1 MB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/static/video
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes
3              2.2 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo
1              27 B               2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/archetypes
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets
1              6.3 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/js
2              39.1 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/js/libraries
2              1.9 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass
3              1.8 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/base
11             13.1 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components
3              8.2 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/elements
3              7.0 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/utils
2              2.2 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/i18n
4              822.2 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/images
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts
4              4.8 KB             2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/_default
10             12.5 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials
1              215 B              2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/shortcodes
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/css
2              21.0 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/css/css
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/js
2              28.4 KB            2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/js/js
0              0 B                2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static
16             354.4 KB           2024-05-25 23:18:22    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images"

value=$(lomob list dirs -t)

if [ "$value" = "$expectDirs" ]; then
  echo "Scan dir lists in tablew view are same as expected"
else
  echo "Scan dir lists in table view are different"
  echo "Expect:"
  echo "$expectDirs"
  echo "Actual:"
  echo "$value"
fi
