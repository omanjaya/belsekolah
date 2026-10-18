#!/usr/bin/env bash
# Memasang atau memperbarui Bel Sekolah dari rilis GitHub terbaru (termasuk pre-release).
# Jalankan dari folder hasil git clone:
#   sudo ./pasang.sh [--versi v0.4.0] [--tz Asia/Makassar] [--hostname bel] [--perangkat-audio NAMA]
# Opsi selain --versi diteruskan ke install.sh di dalam paket rilis.
set -euo pipefail

REPO=omanjaya/belsekolah
VERSI=""
OPSI=()

pesan() { printf '\n==> %s\n' "$*"; }
gagal() { printf '\nGAGAL: %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --versi) VERSI="${2:?versi kosong}"; shift 2 ;;
    -h|--help) sed -n '2,5p' "$0"; exit 0 ;;
    *) OPSI+=("$1"); shift ;;
  esac
done

[[ $EUID -eq 0 ]] || gagal "jalankan dengan sudo"

if ! command -v curl >/dev/null 2>&1; then
  pesan "Memasang curl"
  apt-get update -q
  apt-get install -y -q --no-install-recommends curl ca-certificates
fi

case "$(dpkg --print-architecture)" in
  amd64) ARSITEKTUR=amd64 ;;
  arm64) ARSITEKTUR=arm64 ;;
  *) gagal "arsitektur $(dpkg --print-architecture) tidak didukung" ;;
esac

if [[ -z "$VERSI" ]]; then
  VERSI="$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=1" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)" \
    || gagal "tidak bisa menghubungi GitHub; periksa koneksi internet"
  [[ -n "$VERSI" ]] || gagal "rilis terbaru tidak ditemukan"
fi
[[ "$VERSI" =~ ^v[0-9A-Za-z.-]+$ ]] || gagal "versi tidak valid: $VERSI"

NAMA="belsekolah-$VERSI-linux-$ARSITEKTUR"
URL="https://github.com/$REPO/releases/download/$VERSI"
KERJA="$(mktemp -d)"
trap 'rm -rf "$KERJA"' EXIT

pesan "Mengunduh $NAMA"
curl -fL --retry 3 -o "$KERJA/$NAMA.tar.gz" "$URL/$NAMA.tar.gz" || gagal "paket $NAMA tidak bisa diunduh"
curl -fsSL --retry 3 -o "$KERJA/SHA256SUMS" "$URL/SHA256SUMS" || gagal "SHA256SUMS tidak bisa diunduh"

pesan "Memeriksa checksum"
(cd "$KERJA" && grep -F "./$NAMA.tar.gz" SHA256SUMS | sha256sum -c -) || gagal "checksum paket tidak cocok"

tar -C "$KERJA" -xzf "$KERJA/$NAMA.tar.gz"
pesan "Menjalankan install.sh dari $VERSI"
"$KERJA/$NAMA/install.sh" "${OPSI[@]}"
