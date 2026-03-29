#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUTPUT_DIR=${1:-"$ROOT_DIR/out/totp"}
ISSUER=${E4_TOTP_ISSUER:-"E4 Diary"}
ACCOUNT=${E4_TOTP_ACCOUNT:-"admin"}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "missing required command: $1" >&2
		exit 1
	fi
}

urlencode() {
	node -e 'process.stdout.write(encodeURIComponent(process.argv[1]))' "$1"
}

require_cmd head
require_cmd base32
require_cmd tr
require_cmd node

mkdir -p "$OUTPUT_DIR"

SECRET=$(head -c 20 /dev/urandom | base32 | tr -d '=\n')
LABEL=$(urlencode "$ISSUER:$ACCOUNT")
ENCODED_ISSUER=$(urlencode "$ISSUER")
URI="otpauth://totp/${LABEL}?secret=${SECRET}&issuer=${ENCODED_ISSUER}&algorithm=SHA1&digits=6&period=30"

printf '%s\n' "$SECRET" >"$OUTPUT_DIR/base32.txt"
printf '%s\n' "$URI" >"$OUTPUT_DIR/otpauth-uri.txt"

if command -v qrencode >/dev/null 2>&1; then
	qrencode -t ANSIUTF8 "$URI" >"$OUTPUT_DIR/otpauth-uri.txt.qrcode.txt"
	qrencode -t SVG -o "$OUTPUT_DIR/otpauth-uri.svg" "$URI"
	QR_BACKEND="qrencode"
elif command -v npx >/dev/null 2>&1; then
	npx --yes qrcode "$URI" --small >"$OUTPUT_DIR/otpauth-uri.txt.qrcode.txt"
	npx --yes qrcode "$URI" -o "$OUTPUT_DIR/otpauth-uri.svg" -t svg
	QR_BACKEND="npx qrcode"
else
	echo "missing QR generator: install qrencode or ensure npx is available" >&2
	exit 1
fi

printf 'Base32 Secret: %s\n' "$SECRET"
printf 'Secret Length: %s\n' "${#SECRET}"
printf 'otpauth URI: %s\n' "$URI"
printf 'QR backend: %s\n' "$QR_BACKEND"
printf 'Saved files in: %s\n' "$OUTPUT_DIR"
printf 'Text QR: %s\n' "$OUTPUT_DIR/otpauth-uri.txt.qrcode.txt"
printf 'SVG QR: %s\n' "$OUTPUT_DIR/otpauth-uri.svg"
