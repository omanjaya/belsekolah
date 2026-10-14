#!/usr/bin/env bash
# Membangun paket rilis untuk mini PC: dist/belsekolah-<versi>-linux-<arch>.tar.gz
# Pemakaian: ./scripts/rilis.sh [versi]   (default: git describe)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSI="${1:-$(git describe --tags --always --dirty)}"
if [[ -f web/package.json ]]; then
  (cd web && npm ci && npm run build)
else
  echo "peringatan: web/ belum ada, memakai internal/webui/dist apa adanya" >&2
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
  tar -C dist -czf "dist/$NAMA.tar.gz" "$NAMA"
  rm -rf "dist/$NAMA"
done
(cd dist && { sha256sum ./*.tar.gz 2>/dev/null || shasum -a 256 ./*.tar.gz; } > SHA256SUMS)
ls -lh dist
