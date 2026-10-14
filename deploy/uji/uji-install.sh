#!/usr/bin/env bash
# Membangun paket uji, menjalankan install.sh dua kali di container Debian 13,
# lalu menjalankan layanan secara manual sebagai user belsekolah dan memeriksa API.
set -euo pipefail
cd "$(dirname "$0")/../.."

ARSITEKTUR="$(docker info --format '{{.Architecture}}')"
case "$ARSITEKTUR" in
  x86_64|amd64) ARSITEKTUR=amd64 ;;
  aarch64|arm64) ARSITEKTUR=arm64 ;;
esac

KERJA="$(mktemp -d)"
trap 'rm -rf "$KERJA"' EXIT
mkdir -p "$KERJA/paket"
CGO_ENABLED=0 GOOS=linux GOARCH="$ARSITEKTUR" go build -trimpath -o "$KERJA/paket/belsekolah-linux-$ARSITEKTUR" ./cmd/belsekolah
cp deploy/install.sh deploy/belsekolah.service "$KERJA/paket/"
cp PANDUAN-INSTAL.md "$KERJA/paket/" 2>/dev/null || echo "# panduan" > "$KERJA/paket/PANDUAN-INSTAL.md"
cp deploy/uji/Dockerfile "$KERJA/"

docker build -q -t belsekolah-uji-instal "$KERJA" >/dev/null
docker run --rm -e BELSEKOLAH_UJI=1 belsekolah-uji-instal bash -euo pipefail -c '
  ./install.sh --tz Asia/Makassar --hostname bel-uji --perangkat-audio "alsa/default"
  ./install.sh   # dijalankan ulang TANPA opsi: tz, hostname, dan perangkat harus dipertahankan
  test -x /usr/local/bin/belsekolah
  test -x /usr/local/bin/edge-tts
  edge-tts --help >/dev/null
  su -s /bin/sh belsekolah -c "edge-tts --help" >/dev/null
  id belsekolah | grep -q audio
  test "$(stat -c %U:%a /var/lib/belsekolah)" = "belsekolah:750"
  grep -q "^BELSEKOLAH_TZ=Asia/Makassar$" /etc/default/belsekolah
  grep -q "^BELSEKOLAH_MPV_DEVICE=alsa/default$" /etc/default/belsekolah   # dipertahankan saat diulang
  grep -q "Automatic-Reboot \"false\"" /etc/apt/apt.conf.d/51belsekolah-tanpa-reboot
  test -f /etc/systemd/system/belsekolah.service
  test "$(grep -c "^127\.0\.1\.1[[:space:]]" /etc/hosts)" = "1"
  ! grep -q "^127\.0\.1\.1[[:space:]]bel$" /etc/hosts

  su -s /bin/sh belsekolah -c "BELSEKOLAH_MPV_AO=null belsekolah -data /var/lib/belsekolah -addr 127.0.0.1:8099 -tz Asia/Makassar" &
  for i in $(seq 1 30); do curl -fsS http://127.0.0.1:8099/api/sesi >/dev/null && break; sleep 1; done
  curl -fsS http://127.0.0.1:8099/api/sesi
  test "$(stat -c %a /var/lib/belsekolah/password-awal.txt)" = "600"
  echo
  echo "UJI INSTAL BERHASIL"
'
