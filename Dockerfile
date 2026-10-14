# syntax=docker/dockerfile:1
# Image development/pengujian. Produksi di mini PC memakai binary + systemd (Rencana 5).
FROM golang:1.27-trixie AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSI=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.versi=${VERSI}" -o /out/belsekolah ./cmd/belsekolah

FROM debian:trixie-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends mpv ffmpeg ca-certificates tzdata python3 pipx \
 && rm -rf /var/lib/apt/lists/*
RUN PIPX_HOME=/opt/pipx PIPX_BIN_DIR=/usr/local/bin pipx install edge-tts \
 && edge-tts --list-voices | grep -q id-ID-GadisNeural
COPY deploy/docker/asound.conf /etc/asound.conf
COPY --from=build /out/belsekolah /usr/local/bin/belsekolah
# Tanpa kartu suara: ALSA "null" tidak punya jam (mpv selesai seketika), sedangkan
# --ao=null memutar real-time sehingga durasi bel dan tombol stop berperilaku nyata.
ENV BELSEKOLAH_MPV_AO=null
VOLUME /var/lib/belsekolah
EXPOSE 8080
ENTRYPOINT ["belsekolah"]
CMD ["-data", "/var/lib/belsekolah", "-addr", ":8080", "-tz", "Asia/Jakarta"]
