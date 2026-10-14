#!/usr/bin/env bash
# Membangun paket rilis untuk mini PC: dist/belsekolah-<versi>-linux-<arch>.tar.gz
# Pemakaian: ./scripts/rilis.sh [versi]   (default: git describe)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSI="${1:-$(git describe --tags --always --dirty)}"
(cd web && npm ci && env -u VITE_E2E npm run build)
# Build produksi tidak boleh membawa kait uji (__bel) dari build E2E.
if grep -rq __bel internal/webui/dist; then
  echo "GAGAL: internal/webui/dist mengandung __bel (build E2E). Bangun ulang tanpa VITE_E2E." >&2
  exit 1
fi
go vet ./...
go test ./...

rm -rf dist && mkdir -p dist
for ARSITEKTUR in amd64 arm64; do
  NAMA="belsekolah-$VERSI-linux-$ARSITEKTUR"
  mkdir -p "dist/$NAMA"
  CGO_ENABLED=0 GOOS=linux GOARCH="$ARSITEKTUR" go build -trimpath \
    -ldflags="-s -w -X main.versi=$VERSI" -o "dist/$NAMA/belsekolah-linux-$ARSITEKTUR" ./cmd/belsekolah
  cp deploy/install.sh deploy/belsekolah.service PANDUAN-INSTAL.md "dist/$NAMA/"
  COPYFILE_DISABLE=1 tar -C dist -czf "dist/$NAMA.tar.gz" "$NAMA"
  rm -rf "dist/$NAMA"
done
(cd dist && { sha256sum ./*.tar.gz 2>/dev/null || shasum -a 256 ./*.tar.gz; } > SHA256SUMS)
ls -lh dist
