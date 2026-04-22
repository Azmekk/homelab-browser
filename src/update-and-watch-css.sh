#!/bin/bash
# Compile SCSS once then watch for changes. Requires `sass` on PATH
# (`npm i -g sass`, or invoke via `npx sass ...` if you prefer).
set -e

sass --no-source-map --style compressed ./wwwroot/styles.scss:./wwwroot/styles.css
sass --no-source-map --watch --style compressed ./wwwroot/styles.scss:./wwwroot/styles.css
