#!/bin/bash

set -e

cd "$(dirname "$0")"

HASH=$(openssl dgst -sha384 -binary landing.js | openssl base64 -A)
echo "sha384-${HASH}"

sed -i -E "s|(<script src=\"landing\.js\" integrity=\")sha384-[^\"]+(\")|\1sha384-${HASH}\2|" index.html
