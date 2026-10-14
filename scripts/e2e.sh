#!/usr/bin/env bash
# Menjalankan tes end-to-end terhadap container Docker yang baru (data kosong):
# 1) Playwright (butuh password awal, jadi harus pada container baru),
# 2) container direset, 3) tes Go (yang mengganti password admin).
set -euo pipefail
cd "$(dirname "$0")/.."

PORT_BEL="${BEL_PORT:-8080}"
export BEL_PORT="$PORT_BEL"

docker compose down -v --remove-orphans
trap 'docker compose logs --no-color belsekolah | tail -50; docker compose down -v' EXIT

echo "== Playwright terhadap Docker =="
docker compose up -d --build --wait
(cd web && BEL_URL="http://localhost:${PORT_BEL}" npx playwright test)

echo "== Reset container, lalu tes Go =="
docker compose down -v
docker compose up -d --build --wait
BEL_URL="http://localhost:${PORT_BEL}" BEL_E2E_TTS="${BEL_E2E_TTS:-1}" go test -tags e2e -count=1 -v -timeout 10m ./e2e/
