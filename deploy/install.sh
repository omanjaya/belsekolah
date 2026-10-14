#!/usr/bin/env bash
# Pemasang Bel Sekolah untuk Debian 13. Jalankan dari folder hasil ekstrak paket rilis:
#   sudo ./install.sh [--tz Asia/Jakarta] [--hostname bel] [--perangkat-audio NAMA]
# Aman dijalankan ulang: pemasangan berikutnya memperbarui binary dan konfigurasi.
set -euo pipefail

TZ_SEKOLAH=""
NAMA_HOST=""
PERANGKAT_AUDIO=""
UJI="${BELSEKOLAH_UJI:-0}"
DIR="$(cd "$(dirname "$0")" && pwd)"

pesan() { printf '\n==> %s\n' "$*"; }
gagal() { printf '\nGAGAL: %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tz) TZ_SEKOLAH="${2:?zona waktu kosong}"; shift 2 ;;
    --hostname) NAMA_HOST="${2:?hostname kosong}"; shift 2 ;;
    --perangkat-audio) PERANGKAT_AUDIO="${2:?perangkat audio kosong}"; shift 2 ;;
    -h|--help) sed -n '2,4p' "$0"; exit 0 ;;
    *) gagal "opsi tidak dikenal: $1" ;;
  esac
done

[[ $EUID -eq 0 ]] || gagal "jalankan dengan sudo"
# shellcheck source=/dev/null
[[ -r /etc/os-release ]] && . /etc/os-release
[[ "${ID:-}" == "debian" ]] || gagal "skrip ini untuk Debian (terdeteksi: ${ID:-tidak diketahui})"

# Pemasangan ulang tanpa --tz / --hostname mempertahankan nilai yang sudah ada,
# agar jadwal bel tidak bergeser.
KONFIG=/etc/default/belsekolah
if [[ -z "$TZ_SEKOLAH" ]]; then
  [[ -f "$KONFIG" ]] && TZ_SEKOLAH="$(sed -n 's/^BELSEKOLAH_TZ=//p' "$KONFIG" | tail -n 1)"
  TZ_SEKOLAH="${TZ_SEKOLAH:-Asia/Jakarta}"
fi
if [[ -z "$NAMA_HOST" ]]; then
  if [[ -f "$KONFIG" ]]; then NAMA_HOST="$(hostname)"; else NAMA_HOST="bel"; fi
fi
[[ "$NAMA_HOST" =~ ^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$ ]] || gagal "hostname tidak valid: $NAMA_HOST (pakai huruf kecil, angka, dan tanda minus)"
[[ -e "/usr/share/zoneinfo/$TZ_SEKOLAH" ]] || gagal "zona waktu tidak dikenal: $TZ_SEKOLAH"

case "$(dpkg --print-architecture)" in
  amd64) ARSITEKTUR=amd64 ;;
  arm64) ARSITEKTUR=arm64 ;;
  *) gagal "arsitektur $(dpkg --print-architecture) tidak didukung" ;;
esac
BINARY="$DIR/belsekolah-linux-$ARSITEKTUR"
[[ -x "$BINARY" ]] || gagal "binary $BINARY tidak ditemukan di paket"
for f in PANDUAN-INSTAL.md belsekolah.service; do
  [[ -f "$DIR/$f" ]] || gagal "berkas $f tidak ditemukan di paket"
done

jalankan() {
  # Di mode uji (container tanpa systemd) perintah sistem dilewati.
  if [[ "$UJI" == "1" ]]; then printf '[uji] lewati: %s\n' "$*"; else "$@"; fi
}

pesan "Memasang paket sistem"
export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -y -q --no-install-recommends \
  mpv ffmpeg alsa-utils avahi-daemon chrony unattended-upgrades \
  python3 pipx ca-certificates curl

pesan "Memasang edge-tts (suara TTS)"
export PIPX_HOME=/opt/pipx PIPX_BIN_DIR=/usr/local/bin
DAFTAR_PIPX="$(pipx list --short 2>/dev/null || true)"
if grep -q '^edge-tts ' <<<"$DAFTAR_PIPX"; then
  pipx upgrade edge-tts || true
else
  pipx install edge-tts
fi

pesan "Menyiapkan user dan folder data"
if ! id belsekolah >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/belsekolah --shell /usr/sbin/nologin --user-group belsekolah
fi
usermod -a -G audio belsekolah
install -d -m 0750 -o belsekolah -g belsekolah /var/lib/belsekolah

pesan "Memasang binary dan dokumentasi"
jalankan systemctl stop belsekolah.service 2>/dev/null || true
install -m 0755 "$BINARY" /usr/local/bin/belsekolah
install -d /usr/local/share/doc/belsekolah
install -m 0644 "$DIR/PANDUAN-INSTAL.md" /usr/local/share/doc/belsekolah/PANDUAN-INSTAL.md
install -m 0644 "$DIR/belsekolah.service" /etc/systemd/system/belsekolah.service

{
  echo "# Dibuat oleh install.sh. Ubah lalu: sudo systemctl restart belsekolah"
  echo "BELSEKOLAH_TZ=$TZ_SEKOLAH"
  if [[ -n "$PERANGKAT_AUDIO" ]]; then
    echo "BELSEKOLAH_MPV_DEVICE=$PERANGKAT_AUDIO"
  elif [[ -f /etc/default/belsekolah ]]; then
    grep '^BELSEKOLAH_MPV_DEVICE=' /etc/default/belsekolah || true
  fi
} > /etc/default/belsekolah.baru
mv /etc/default/belsekolah.baru /etc/default/belsekolah

pesan "Mengatur zona waktu, sinkronisasi jam, dan nama perangkat"
jalankan timedatectl set-timezone "$TZ_SEKOLAH"
jalankan systemctl enable --now chrony.service
jalankan hostnamectl set-hostname "$NAMA_HOST"
if ! grep -qE "^127\.0\.1\.1[[:space:]]+$NAMA_HOST\$" /etc/hosts; then
  # Tulis ulang di tempat (bukan sed -i): /etc/hosts bisa berupa bind-mount di container.
  HOSTS_BARU="$(grep -v '^127\.0\.1\.1[[:space:]]' /etc/hosts || true)"
  printf '%s\n127.0.1.1\t%s\n' "$HOSTS_BARU" "$NAMA_HOST" > /etc/hosts
fi
jalankan systemctl enable --now avahi-daemon.service

pesan "Mengaktifkan pembaruan keamanan otomatis (tanpa reboot otomatis)"
cat > /etc/apt/apt.conf.d/20auto-upgrades <<'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
EOF
cat > /etc/apt/apt.conf.d/51belsekolah-tanpa-reboot <<'EOF'
Unattended-Upgrade::Automatic-Reboot "false";
EOF

pesan "Mengatur volume ALSA ke maksimum (volume diatur dari aplikasi)"
jalankan amixer -q sset Master 100% unmute 2>/dev/null || true
jalankan amixer -q sset PCM 100% unmute 2>/dev/null || true
jalankan alsactl store 2>/dev/null || true

pesan "Menjalankan layanan"
jalankan systemctl daemon-reload
jalankan systemctl enable belsekolah.service
jalankan systemctl restart belsekolah.service

if [[ "$UJI" != "1" ]]; then
  for _ in $(seq 1 30); do
    curl -fsS -o /dev/null http://127.0.0.1/api/sesi && break
    sleep 1
  done
  curl -fsS -o /dev/null http://127.0.0.1/api/sesi || gagal "layanan tidak merespons; lihat: journalctl -u belsekolah -n 50"
fi

IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
cat <<EOF

Bel Sekolah terpasang.

  Buka dari HP/laptop di WiFi sekolah:
    http://${IP:-<alamat-ip>}   (atau http://${NAMA_HOST}.local)

  Password admin awal (hanya untuk login pertama):
    sudo cat /var/lib/belsekolah/password-awal.txt

  Perintah berguna:
    sudo systemctl status belsekolah       status layanan
    sudo journalctl -u belsekolah -f       log
    sudo -u belsekolah belsekolah -data /var/lib/belsekolah -reset-password   reset password

  Panduan lengkap: /usr/local/share/doc/belsekolah/PANDUAN-INSTAL.md
EOF
