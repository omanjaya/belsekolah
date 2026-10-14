# Bel Sekolah

Aplikasi bel sekolah otomatis (Go) dengan antarmuka web, berjalan di mini PC Debian.

## Pengembangan

```
go run ./cmd/belsekolah -data ./tmp-data -addr :8080
```

## Pengujian

```
go vet ./...
go test ./...
./scripts/e2e.sh          # uji end-to-end
./deploy/uji/uji-install.sh   # uji pemasang di container Debian 13 (butuh Docker)
```

## Pemasangan di mini PC

Panduan lengkap untuk operator: [PANDUAN-INSTAL.md](PANDUAN-INSTAL.md).

Untuk membuat paket rilis (amd64 dan arm64):

```
./scripts/rilis.sh [versi]
```

Hasilnya ada di `dist/belsekolah-<versi>-linux-<arch>.tar.gz` beserta `SHA256SUMS`. Paket berisi binary, `install.sh`, `belsekolah.service`, dan `PANDUAN-INSTAL.md`. Salin ke mini PC, ekstrak, lalu jalankan `sudo ./install.sh`.
