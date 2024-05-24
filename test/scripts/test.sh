#!/bin/bash

echo "Test list iso files in db"

expectISOFiles="ID    Name                          Size       Status                   Region    Bucket    Files Count    Create Time            Local Hash
1     2019-04-03--2024-05-25.iso    30.7 MB    Created, not uploaded                        113            2024-05-25 23:21:21    54979da5dcdf0ff08c046dd97662677a7b186bb13ef700668e36899f2e0d1df5
2     2021-04-26--2021-04-26.iso    32.3 MB    Created, not uploaded                        3              2024-05-25 23:21:21    4a93ac387edc587bce1d5b1a51dd78358cd9b9275a02a90e5b7e433256f94444
3     2021-04-26--2021-07-31.iso    41.9 MB    Created, not uploaded                        16             2024-05-25 23:21:21    44434ef03763d476ed82602f7ee6991f40e2c698d37c7b7771fb39cd14fb3cb7"

value=$(lomob iso list)

if [ "$value" = "$expectISOFiles" ]; then
  echo "ISO file lists are same as expected"
else
  echo "ISO file lists are different"
  echo "Expect:"
  echo "$expectISOFiles"
  echo "Actual:"
  echo "$value"
fi

echo "Test listing files not in iso yet"

expectFiles="In Cloud    Path
            /go/src/github.com/lomorage/lomo-backup/test/data/static/video/lomorage.mp4
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/LICENSE.md
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/README.md
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/theme.toml
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/archetypes/default.md
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/js/app.js
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/js/libraries/flexibility.js
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/js/libraries/responsive-nav.js
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/custom.style.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/style.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/base/_containers.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/base/_globals.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/base/_variables.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_app.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_auth.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_cta.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_expanded.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_footer.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_hero.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_landing.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_navbar.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_page.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_steps.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/components/_verticalMenu.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/elements/_buttons.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/elements/_forms.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/elements/_typography.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/utils/_helpers.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/utils/_mixins.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/assets/sass/utils/_normalize.scss
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/i18n/en.yaml
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/i18n/zh.yaml
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/images/screenshot.png
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/images/screenshot2.png
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/images/screenshot3.png
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/images/tn.png
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/_default/baseof.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/_default/list.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/_default/single.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/_default/terms.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/adsense.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/contact.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/footer.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/header.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/landing_cta.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/landing_hero.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/landing_single.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/landing_triple.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/nav.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/partials/scripts.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/layouts/shortcodes/table.html
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/css/css/style.css_b95b077eb505d5c0aff8055eaced30ad.content
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/css/css/style.css_b95b077eb505d5c0aff8055eaced30ad.json
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/js/js/scripts.js_d11fe7b62c27961c87ecd0f2490357b9.content
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/resources/_gen/assets/js/js/scripts.js_d11fe7b62c27961c87ecd0f2490357b9.json
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/evie_default_bg.jpeg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/hero_sm.png
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/tet.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/together.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_browser.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_creation.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_design.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_designer.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_elements.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_everywhere.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_fans.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_frameworks.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_hello_aeia.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_responsive.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_selfie.svg
            /go/src/github.com/lomorage/lomo-backup/test/data/themes/evie-hugo/static/images/undraw_tabs.svg"

value=$(lomob list files)

if [ "$value" = "$expectFiles" ]; then
  echo "Files not in ISO are same as expected"
else
  echo "Files not in ISO are different"
  echo "Expect:"
  echo "$expectFiles"
  echo "Actual:"
  echo "$value"
fi

echo "Test dump iso files"
expectFiles=""

isoFile="2019-04-03--2024-05-25.iso"
value=$(lomob iso dump $isoFile)

if [ "$value" = "$expectFiles" ]; then
  echo "Dump iso $isoFile are same as expected"
else
  echo "Dump iso $isoFile are different"
  echo "Expect:"
  echo "$expectFiles"
  echo "Actual:"
  echo "$value"
fi