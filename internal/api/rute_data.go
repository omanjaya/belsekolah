package api

import "net/http"

// daftarRuteData mendaftarkan rute CRUD data, pustaka suara, riwayat, dan backup.
func (s *server) daftarRuteData(rute func(pola string, h http.HandlerFunc)) {
	rute("GET /api/profil", s.listProfil)
	rute("POST /api/profil", s.buatProfil)
	rute("PUT /api/profil/{id}", s.ubahProfil)
	rute("DELETE /api/profil/{id}", s.hapusProfil)
	rute("POST /api/profil/{id}/duplikat", s.duplikatProfil)
	rute("GET /api/profil/{id}/ekspor", s.eksporProfil)
	rute("POST /api/profil/{id}/gabung/periksa", s.periksaGabung)
	rute("POST /api/profil/{id}/gabung", s.gabungJadwal)
	rute("POST /api/impor/periksa", s.periksaImpor)
	rute("POST /api/impor", s.impor)

	rute("GET /api/template", s.listTemplate)
	rute("POST /api/template", s.buatTemplate)
	rute("POST /api/template/buat", s.buatDariParameter)
	rute("GET /api/template/{id}", s.ambilTemplate)
	rute("PUT /api/template/{id}", s.ubahTemplate)
	rute("DELETE /api/template/{id}", s.hapusTemplate)
	rute("GET /api/template/{id}/ekspor", s.eksporTemplate)

	rute("GET /api/jadwal", s.listJadwal)
	rute("POST /api/jadwal", s.buatJadwal)
	rute("POST /api/jadwal/salin", s.salinJadwal)
	rute("PUT /api/jadwal/{id}", s.ubahJadwal)
	rute("DELETE /api/jadwal/{id}", s.hapusJadwal)

	rute("GET /api/periode", s.listPeriode)
	rute("POST /api/periode", s.buatPeriode)
	rute("PUT /api/periode/{id}", s.ubahPeriode)
	rute("DELETE /api/periode/{id}", s.hapusPeriode)

	rute("GET /api/libur", s.listLibur)
	rute("POST /api/libur", s.buatLibur)
	rute("PUT /api/libur/{id}", s.ubahLibur)
	rute("DELETE /api/libur/{id}", s.hapusLibur)

	rute("GET /api/jenis-bel", s.listJenisBel)
	rute("POST /api/jenis-bel", s.buatJenisBel)
	rute("PUT /api/jenis-bel/{id}", s.ubahJenisBel)
	rute("DELETE /api/jenis-bel/{id}", s.hapusJenisBel)
	rute("POST /api/jenis-bel/{id}/tes", s.tesJenisBel)

	rute("GET /api/audio", s.listAudio)
	rute("POST /api/audio", s.unggahAudio)
	rute("PUT /api/audio/{id}", s.ubahAudio)
	rute("DELETE /api/audio/{id}", s.hapusAudio)
	rute("GET /api/audio/{id}/file", s.fileAudio)
	rute("POST /api/audio/{id}/tes", s.tesAudio)

	rute("GET /api/riwayat", s.listRiwayat)
	rute("GET /api/backup", s.unduhBackup)
	rute("POST /api/restore", s.pulihkan)
	rute("GET /api/tts/suara", s.listSuaraTTS)
	rute("POST /api/tts/pratinjau", s.pratinjauTTS)
	rute("POST /api/audio/tts", s.buatAudioTTS)
	rute("PUT /api/audio/{id}/tts", s.ubahAudioTTS)
}
