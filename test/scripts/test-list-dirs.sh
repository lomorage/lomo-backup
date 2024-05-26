#!/bin/bash

echo "Test list scan dirs"

expectDirs="/go/src/github.com/lomorage/lomo-backup/test/data
├── [     2021-06-15]  archetypes
├── [     2024-03-17]  content
│   └── [     2024-03-17]  blog
├── [     2021-06-15]  layouts
├── [     2024-03-31]  public
├── [     2021-07-06]  resources
│   └── [     2021-07-06]  _gen
│       └── [     2021-07-06]  assets
│           ├── [     2021-07-06]  css
│           │   └── [     2021-07-06]  css
│           └── [     2021-06-15]  js
│               └── [     2021-06-15]  js
├── [     2024-04-17]  static
│   ├── [     2023-03-03]  img
│   │   ├── [     2023-03-03]  blog
│   │   │   ├── [     2021-08-26]  android-frame
│   │   │   ├── [     2020-03-30]  covid19
│   │   │   ├── [     2022-10-14]  flyio
│   │   │   ├── [     2021-09-02]  image-quality
│   │   │   ├── [     2020-05-18]  import_my_cloud
│   │   │   ├── [     2023-03-03]  migrate_from_pi_win
│   │   │   ├── [     2020-07-20]  mypains
│   │   │   ├── [     2022-02-12]  photoprism
│   │   │   ├── [     2021-06-15]  raspberrypi-hd
│   │   │   ├── [     2023-03-02]  set_wd_as_backup
│   │   │   ├── [     2020-03-31]  transfer_pc
│   │   │   └── [     2020-11-17]  win-docker
│   │   ├── [     2021-05-03]  buy
│   │   └── [     2021-07-31]  links
│   └── [     2021-06-15]  video
└── [     2023-11-13]  themes
    └── [     2023-11-13]  evie-hugo
        ├── [     2021-06-15]  archetypes
        ├── [     2021-07-06]  assets
        │   ├── [     2021-06-15]  js
        │   │   └── [     2021-06-15]  libraries
        │   └── [     2021-07-06]  sass
        │       ├── [     2021-06-15]  base
        │       ├── [     2021-07-06]  components
        │       ├── [     2021-06-16]  elements
        │       └── [     2021-06-15]  utils
        ├── [     2021-06-15]  i18n
        ├── [     2021-06-15]  images
        ├── [     2023-11-13]  layouts
        │   ├── [     2023-11-13]  _default
        │   ├── [     2023-11-13]  partials
        │   └── [     2021-06-15]  shortcodes
        ├── [     2021-06-15]  resources
        │   └── [     2021-06-15]  _gen
        │       └── [     2021-06-15]  assets
        │           ├── [     2021-06-15]  css
        │           │   └── [     2021-06-15]  css
        │           └── [     2021-06-15]  js
        │               └── [     2021-06-15]  js
        └── [     2022-05-04]  static
            └── [     2021-06-15]  images
"

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
6              15.6 KB            2024-04-17 06:24:25    /go/src/github.com/lomorage/lomo-backup/test/data
1              84 B               2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/archetypes
10             37.8 KB            2024-03-17 18:06:22    /go/src/github.com/lomorage/lomo-backup/test/data/content
20             92.5 KB            2024-03-17 18:06:22    /go/src/github.com/lomorage/lomo-backup/test/data/content/blog
1              182 B              2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/layouts
0              0 B                2024-03-31 04:13:14    /go/src/github.com/lomorage/lomo-backup/test/data/public
0              0 B                2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/resources
0              0 B                2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen
0              0 B                2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets
0              0 B                2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/css
4              42.5 KB            2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/css/css
0              0 B                2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/js
2              28.4 KB            2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/resources/_gen/assets/js/js
10             1.6 MB             2024-04-17 06:24:25    /go/src/github.com/lomorage/lomo-backup/test/data/static
12             77.9 KB            2023-03-03 18:26:15    /go/src/github.com/lomorage/lomo-backup/test/data/static/img
1              77.9 KB            2023-03-03 18:26:15    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog
3              21.6 MB            2021-08-26 18:31:00    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/android-frame
4              610.0 KB           2020-03-30 16:37:12    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/covid19
1              9.8 KB             2022-10-14 17:50:33    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/flyio
4              393.1 KB           2021-09-02 05:38:39    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/image-quality
5              158.6 KB           2020-05-18 15:18:23    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/import_my_cloud
3              87.8 KB            2023-03-03 18:26:15    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/migrate_from_pi_win
2              128.9 KB           2020-07-20 04:22:51    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/mypains
1              25.6 KB            2022-02-12 06:59:17    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/photoprism
1              238.2 KB           2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/raspberrypi-hd
5              223.6 KB           2023-03-02 17:18:16    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/set_wd_as_backup
10             535.2 KB           2020-03-31 02:36:01    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/transfer_pc
3              140.4 KB           2020-11-17 01:04:43    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/blog/win-docker
14             42.1 MB            2021-05-03 20:55:52    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/buy
6              374.1 KB           2021-07-31 07:06:41    /go/src/github.com/lomorage/lomo-backup/test/data/static/img/links
3              38.1 MB            2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/static/video
0              0 B                2023-11-13 01:55:39    /go/src/github.com/lomorage/lomo-backup/test/data/themes
3              2.2 KB             2023-11-13 01:55:39    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo
1              27 B               2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/archetypes
0              0 B                2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets
1              6.3 KB             2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/js
2              39.1 KB            2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/js/libraries
2              1.9 KB             2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass
3              1.8 KB             2021-06-15 23:26:27    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/base
11             13.1 KB            2021-07-06 19:31:23    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components
3              8.2 KB             2021-06-16 06:48:55    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/elements
3              7.0 KB             2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/utils
2              2.2 KB             2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/i18n
4              822.2 KB           2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/images
0              0 B                2023-11-13 01:55:39    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts
4              4.8 KB             2023-11-13 01:55:39    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/_default
10             12.5 KB            2023-11-13 01:55:39    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials
1              215 B              2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/shortcodes
0              0 B                2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources
0              0 B                2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen
0              0 B                2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets
0              0 B                2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/css
2              21.0 KB            2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/css/css
0              0 B                2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/js
2              28.4 KB            2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/js/js
0              0 B                2022-05-04 19:54:39    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static
16             354.4 KB           2021-06-15 22:19:06    /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images
"

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
