CREATE TABLE IF NOT EXISTS audio (
  id         INTEGER PRIMARY KEY,
  nama       TEXT NOT NULL,
  sumber     TEXT NOT NULL CHECK (sumber IN ('bawaan', 'upload', 'tts')),
  file       TEXT NOT NULL,
  durasi_ms  INTEGER NOT NULL DEFAULT 0,
  teks_tts   TEXT NOT NULL DEFAULT '',
  suara_tts  TEXT NOT NULL DEFAULT '',
  dibuat     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS jenis_bel (
  id     INTEGER PRIMARY KEY,
  nama   TEXT NOT NULL UNIQUE,
  warna  TEXT NOT NULL DEFAULT '#2563eb'
);

CREATE TABLE IF NOT EXISTS urutan_bel (
  jenis_bel_id  INTEGER NOT NULL REFERENCES jenis_bel(id) ON DELETE CASCADE,
  urutan        INTEGER NOT NULL,
  audio_id      INTEGER NOT NULL REFERENCES audio(id) ON DELETE RESTRICT,
  PRIMARY KEY (jenis_bel_id, urutan)
);

CREATE TABLE IF NOT EXISTS profil (
  id    INTEGER PRIMARY KEY,
  nama  TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS jadwal (
  id            INTEGER PRIMARY KEY,
  profil_id     INTEGER NOT NULL REFERENCES profil(id) ON DELETE CASCADE,
  hari          INTEGER NOT NULL CHECK (hari BETWEEN 1 AND 7),
  jam           TEXT NOT NULL,
  jenis_bel_id  INTEGER NOT NULL REFERENCES jenis_bel(id) ON DELETE RESTRICT,
  label         TEXT NOT NULL DEFAULT '',
  UNIQUE (profil_id, hari, jam)
);

CREATE TABLE IF NOT EXISTS periode_khusus (
  id           INTEGER PRIMARY KEY,
  tgl_mulai    TEXT NOT NULL,
  tgl_selesai  TEXT NOT NULL,
  profil_id    INTEGER NOT NULL REFERENCES profil(id) ON DELETE RESTRICT,
  keterangan   TEXT NOT NULL DEFAULT '',
  CHECK (tgl_mulai <= tgl_selesai)
);

CREATE TABLE IF NOT EXISTS libur (
  id           INTEGER PRIMARY KEY,
  tgl_mulai    TEXT NOT NULL,
  tgl_selesai  TEXT NOT NULL,
  keterangan   TEXT NOT NULL DEFAULT '',
  CHECK (tgl_mulai <= tgl_selesai)
);

CREATE TABLE IF NOT EXISTS pengaturan (
  kunci  TEXT PRIMARY KEY,
  nilai  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS riwayat_bel (
  id            INTEGER PRIMARY KEY,
  waktu         TEXT NOT NULL,
  tanggal       TEXT NOT NULL,
  jadwal_id     INTEGER,
  jenis_bel_id  INTEGER,
  label         TEXT NOT NULL DEFAULT '',
  sumber        TEXT NOT NULL CHECK (sumber IN ('jadwal', 'manual', 'darurat')),
  status        TEXT NOT NULL CHECK (status IN ('berhasil', 'gagal', 'terlewat', 'cadangan', 'dihentikan')),
  pesan         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS riwayat_bel_tanggal ON riwayat_bel (tanggal);
