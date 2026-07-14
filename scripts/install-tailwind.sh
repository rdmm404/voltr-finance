#!/bin/sh
set -eu

version="4.3.2"
os="$(uname -s)"
arch="$(uname -m)"
case "$os/$arch" in
  Linux/x86_64) asset=tailwindcss-linux-x64; sha=5036c4fb4328e0bcdbb6065c70d8ac9452e0d4c947113a788a8f94fd390425c1 ;;
  Linux/aarch64|Linux/arm64) asset=tailwindcss-linux-arm64; sha=394ddccc2402cfa3abd97dfba56f3587781a3d6e6ce66e65ceada14beb7664b8 ;;
  Darwin/arm64) asset=tailwindcss-macos-arm64; sha=b800b0659dc64b9f03ede5660244d9415d777d5739ae2889280877ca37be742a ;;
  Darwin/x86_64) asset=tailwindcss-macos-x64; sha=cef8f110471e889c3c4409055cf8aff33076f58a081867b0dfc6534b290bfbb0 ;;
  *) echo "unsupported Tailwind platform: $os/$arch" >&2; exit 1 ;;
esac

destination="${TAILWIND_BIN:-.tools/tailwindcss}"
mkdir -p "$(dirname "$destination")"
temporary="$destination.tmp"
trap 'rm -f "$temporary"' EXIT
curl --fail --location --silent --show-error \
  "https://github.com/tailwindlabs/tailwindcss/releases/download/v${version}/${asset}" \
  --output "$temporary"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$temporary" | cut -d ' ' -f 1)"
else
  actual="$(shasum -a 256 "$temporary" | cut -d ' ' -f 1)"
fi
if [ "$actual" != "$sha" ]; then
  echo "Tailwind checksum mismatch: got $actual, expected $sha" >&2
  exit 1
fi
chmod +x "$temporary"
mv "$temporary" "$destination"
trap - EXIT
echo "Installed Tailwind CSS v${version} at $destination"
