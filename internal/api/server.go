// Package api menyajikan REST API dan antarmuka web bel sekolah.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"belsekolah/internal/auth"
	"belsekolah/internal/player"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/store"
)

type Penjadwal interface {
	Status() scheduler.Status
	Reload()
	BunyikanManual(ctx context.Context, jenisBelID int64, label string, darurat bool)
	FileUrutan(ctx context.Context, jenisBelID int64) []string
}

type Pemutar interface {
	Play(req player.Permintaan)
	Stop()
	SetVolume(v int)
}

type Deps struct {
	Store       *store.Store
	Auth        *auth.Auth
	Penjadwal   Penjadwal
	Pemutar     Pemutar
	DataDir     string
	AudioDir    string
	Versi       string
	Zona        string // nama IANA zona aktif; "" = lokasi dari Now()
	Web         fs.FS
	Log         *slog.Logger
	Now         func() time.Time
	CekNTP      func() *bool                                          // nil = tidak diketahui
	DurasiAudio func(ctx context.Context, path string) (int64, error) // durasi ms; error jika bukan audio
	MulaiUlang  func()                                                // dipanggil setelah restore disiapkan
	TTS         PembuatTTS                                            // nil = TTS tidak tersedia

	// SiapkanRestore menyiapkan restore; nil = backup.Siapkan. Hook untuk pengujian.
	SiapkanRestore func(ctx context.Context, zipPath, dataDir string) error
	// LookPath mencari program di PATH; nil = exec.LookPath.
	LookPath func(nama string) (string, error)
}

// Status restore pada server.
const (
	restoreIdle   int32 = iota
	restoreProses       // restore sedang diproses
	restoreSiap         // restore sudah disiapkan; aplikasi akan dimulai ulang
)

type server struct {
	Deps
	boot    string       // acak per proses; berubah berarti aplikasi sudah dimulai ulang
	restore atomic.Int32 // restoreIdle / restoreProses / restoreSiap

	akanMulaiUlang atomic.Bool // zona diganti; aplikasi akan dimulai ulang
}

// zonaAktif mengembalikan nama IANA zona yang sedang dipakai proses.
func (s *server) zonaAktif() string {
	if s.Zona != "" {
		return s.Zona
	}
	return s.Now().Location().String()
}

func idBoot() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand tidak pernah gagal pada OS yang didukung
	}
	return hex.EncodeToString(b)
}

const namaCookie = "sesi"

func New(d Deps) http.Handler {
	s := &server{Deps: d, boot: idBoot()}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("GET /api/sesi", s.sesi)
	mux.Handle("POST /api/logout", s.wajibLogin(s.logout, false))
	mux.Handle("POST /api/password", s.wajibLogin(s.gantiPassword, false))

	rute := func(pola string, h http.HandlerFunc) { mux.Handle(pola, s.wajibLogin(h, true)) }
	rute("GET /api/status", s.status)
	rute("POST /api/bel/manual", s.belManual)
	rute("POST /api/bel/darurat", s.belDarurat)
	rute("POST /api/bel/stop", s.belStop)
	rute("GET /api/pengaturan", s.ambilPengaturan)
	rute("PUT /api/pengaturan", s.ubahPengaturan)
	rute("GET /api/sistem", s.sistem)
	s.daftarRuteData(rute)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		tulisError(w, http.StatusNotFound, "alamat API tidak dikenal")
	})
	mux.Handle("/", s.ui())
	return s.pulihkanPanic(s.headerAman(s.blokirSaatRestore(s.cekOrigin(mux))))
}

// ubahData dipanggil setelah perubahan data berhasil agar penjadwal memuat ulang.
func (s *server) ubahData() {
	s.Penjadwal.Reload()
}

func (s *server) wajibLogin(h http.HandlerFunc, wajibPasswordBaru bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(namaCookie)
		if err != nil || !s.Auth.Valid(c.Value) {
			tulisError(w, http.StatusUnauthorized, "silakan login terlebih dahulu")
			return
		}
		if wajibPasswordBaru {
			wajib, err := s.Auth.WajibGanti(r.Context())
			if err != nil {
				s.tulisErrorData(w, err)
				return
			}
			if wajib {
				tulisError(w, http.StatusForbidden, "ganti password awal terlebih dahulu")
				return
			}
		}
		h(w, r)
	})
}

// headerAman menambah header pengaman ke setiap respons.
func (s *server) headerAman(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// blokirSaatRestore menolak semua permintaan pengubah data (kecuali login/logout)
// setelah restore disiapkan, supaya tidak ada perubahan yang hilang saat aplikasi dimulai ulang.
func (s *server) blokirSaatRestore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.restore.Load() == restoreSiap || s.akanMulaiUlang.Load() {
			switch r.Method {
			case http.MethodGet, http.MethodHead:
			default:
				if r.URL.Path != "/api/login" && r.URL.Path != "/api/logout" {
					pesan := "aplikasi sedang dimulai ulang untuk menerapkan restore"
					if s.restore.Load() != restoreSiap {
						pesan = "aplikasi sedang dimulai ulang untuk menerapkan zona waktu"
					}
					tulisError(w, http.StatusServiceUnavailable, pesan)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// cekOrigin menolak permintaan pengubah data dari situs lain (pelengkap SameSite=Strict).
func (s *server) cekOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if o := r.Header.Get("Origin"); o != "" {
				u, err := url.Parse(o)
				if err != nil || u.Host != r.Host {
					tulisError(w, http.StatusForbidden, "permintaan dari situs lain ditolak")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) pulihkanPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.Log.Error("panic di handler", "path", r.URL.Path, "nilai", v)
				tulisError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ui menyajikan frontend; path yang bukan file dikembalikan ke index.html (SPA).
func (s *server) ui() http.Handler {
	if s.Web == nil {
		return http.NotFoundHandler()
	}
	berkas := http.FileServerFS(s.Web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p != "" {
			if info, err := fs.Stat(s.Web, p); err != nil || info.IsDir() {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/"
				berkas.ServeHTTP(w, r2)
				return
			}
		}
		berkas.ServeHTTP(w, r)
	})
}

// PembuatTTS mengubah teks menjadi MP3 (diimplementasikan tts.EdgeTTS).
type PembuatTTS interface {
	Buat(ctx context.Context, teks, suara string) ([]byte, error)
}
