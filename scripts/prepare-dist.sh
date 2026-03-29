#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

cd "$ROOT_DIR/web"
npm install
npm run build

cd "$ROOT_DIR"
rm -rf pkg/dist
cp -r web/dist pkg/
