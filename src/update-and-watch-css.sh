#!/bin/bash

# Update the CSS file
sass --no-source-map --style compressed ./wwwroot/styles.scss:./wwwroot/styles.css

# Watch the CSS file
sass --no-source-map --watch --style compressed ./wwwroot/styles.scss:./wwwroot/styles.css 