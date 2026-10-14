#!/usr/bin/env bash
# Menjalankan tes end-to-end terhadap container Docker yang baru (data kosong).
set -euo pipefail
cd "$(dirname "$0")/.."

docker compose down -v --remove-orphans
docker compose up -d --build
trap 'docker compose logs --no-color belsekolah | tail -50; docker compose down -v' EXIT

BEL_URL=http://localhost:${BEL_PORT:-8080} BEL_E2E_TTS="${BEL_E2E_TTS:-1}" go test -tags e2e -count=1 -v -timeout 10m ./e2e/
