// Package auth mengelola password admin tunggal dan sesi login.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"belsekolah/internal/store"
)

const (
	KunciHash            = "password_hash"
	KunciWajibGanti      = "wajib_ganti_password"
	NamaFilePasswordAwal = "password-awal.txt"
	BatasGagal           = 5
	LamaKunci            = 5 * time.Minute
	UmurSesi             = 7 * 24 * time.Hour
	PanjangMinimal       = 8
	panjangMaksimalByte  = 72 // batas bcrypt
)

// BiayaHash adalah cost bcrypt; paket test menurunkannya ke bcrypt.MinCost agar cepat.
var BiayaHash = bcrypt.DefaultCost

var (
	ErrPasswordSalah    = errors.New("password salah")
	ErrTerkunci         = errors.New("terlalu banyak percobaan login, coba lagi dalam 5 menit")
	ErrPasswordPendek   = fmt.Errorf("password baru minimal %d karakter", PanjangMinimal)
	ErrPasswordPanjang  = fmt.Errorf("password baru maksimal %d byte", panjangMaksimalByte)
	ErrPasswordSama     = errors.New("password baru harus berbeda dari password lama")
	ErrBelumAdaPassword = errors.New("password admin belum dibuat")
)

type Penyimpan interface {
	GetSetting(ctx context.Context, kunci string) (string, error)
	SetSetting(ctx context.Context, kunci, nilai string) error
}

type catatanGagal struct {
	gagal       int
	kunciSampai time.Time
}

type Auth struct {
	st  Penyimpan
	now func() time.Time

	mu    sync.Mutex
	sesi  map[string]time.Time     // token → terakhir dipakai
	gagal map[string]*catatanGagal // alamat IP → percobaan gagal
}

func New(st Penyimpan, now func() time.Time) *Auth {
	return &Auth{st: st, now: now, sesi: map[string]time.Time{}, gagal: map[string]*catatanGagal{}}
}

// SiapkanPasswordAwal membuat password acak jika belum ada password sama sekali.
func (a *Auth) SiapkanPasswordAwal(ctx context.Context) (string, bool, error) {
	_, err := a.st.GetSetting(ctx, KunciHash)
	if err == nil {
		return "", false, nil
	}
	if !errors.Is(err, store.ErrTidakAda) {
		return "", false, err
	}
	pw := PasswordAcak()
	if err := a.simpan(ctx, pw, true); err != nil {
		return "", false, err
	}
	return pw, true, nil
}

// ResetPassword membuat password acak baru yang wajib diganti, dan mengakhiri semua sesi.
func (a *Auth) ResetPassword(ctx context.Context) (string, error) {
	pw := PasswordAcak()
	if err := a.simpan(ctx, pw, true); err != nil {
		return "", err
	}
	a.mu.Lock()
	a.sesi = map[string]time.Time{}
	a.mu.Unlock()
	return pw, nil
}

func (a *Auth) simpan(ctx context.Context, pw string, wajibGanti bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), BiayaHash)
	if err != nil {
		return err
	}
	if err := a.st.SetSetting(ctx, KunciHash, string(hash)); err != nil {
		return err
	}
	nilai := "0"
	if wajibGanti {
		nilai = "1"
	}
	return a.st.SetSetting(ctx, KunciWajibGanti, nilai)
}

func (a *Auth) cocok(ctx context.Context, pw string) (bool, error) {
	hash, err := a.st.GetSetting(ctx, KunciHash)
	if errors.Is(err, store.ErrTidakAda) {
		return false, ErrBelumAdaPassword
	}
	if err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil, nil
}

// Login memeriksa password dan membuat sesi baru. Lima kegagalan dari IP yang sama mengunci IP itu 5 menit.
func (a *Auth) Login(ctx context.Context, ip, password string) (string, error) {
	// Bagian 1: cek kunci dan cadangkan satu percobaan sebelum bcrypt,
	// agar permintaan paralel tidak lolos semua sebelum kegagalan tercatat.
	a.mu.Lock()
	c := a.gagal[ip]
	if c == nil {
		c = &catatanGagal{}
		a.gagal[ip] = c
	}
	if a.now().Before(c.kunciSampai) {
		a.mu.Unlock()
		return "", ErrTerkunci
	}
	c.gagal++
	if c.gagal > BatasGagal {
		c.gagal = 0
		c.kunciSampai = a.now().Add(LamaKunci)
		a.mu.Unlock()
		return "", ErrTerkunci
	}
	a.mu.Unlock()

	ok, err := a.cocok(ctx, password)

	// Bagian 2: selesaikan percobaan.
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		if c.gagal > 0 {
			c.gagal-- // kesalahan internal bukan percobaan salah: kembalikan cadangan
		}
		return "", err
	}
	if ok {
		if a.now().Before(c.kunciSampai) {
			return "", ErrTerkunci
		}
		delete(a.gagal, ip) // mengembalikan cadangan percobaan
		token := tokenAcak()
		a.sesi[token] = a.now()
		return token, nil
	}
	if c.gagal >= BatasGagal {
		c.gagal = 0
		c.kunciSampai = a.now().Add(LamaKunci)
		return "", ErrTerkunci
	}
	return "", ErrPasswordSalah
}

func (a *Auth) Logout(token string) {
	a.mu.Lock()
	delete(a.sesi, token)
	a.mu.Unlock()
}

// Valid memeriksa token dan memperpanjang masa berlakunya.
func (a *Auth) Valid(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	terakhir, ok := a.sesi[token]
	if !ok {
		return false
	}
	if a.now().Sub(terakhir) > UmurSesi {
		delete(a.sesi, token)
		return false
	}
	a.sesi[token] = a.now()
	return true
}

func (a *Auth) WajibGanti(ctx context.Context) (bool, error) {
	v, err := a.st.GetSetting(ctx, KunciWajibGanti)
	if errors.Is(err, store.ErrTidakAda) {
		return false, nil
	}
	return v == "1", err
}

func (a *Auth) GantiPassword(ctx context.Context, lama, baru string) error {
	ok, err := a.cocok(ctx, lama)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPasswordSalah
	}
	switch {
	case utf8.RuneCountInString(baru) < PanjangMinimal:
		return ErrPasswordPendek
	case len(baru) > panjangMaksimalByte:
		return ErrPasswordPanjang
	case baru == lama:
		return ErrPasswordSama
	}
	return a.simpan(ctx, baru, false)
}

const alfabet = "abcdefghjkmnpqrstuvwxyz23456789"

// PasswordAcak membuat password 10 karakter tanpa karakter yang mudah tertukar.
func PasswordAcak() string {
	b := make([]byte, 10)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alfabet))))
		if err != nil {
			panic(err) // crypto/rand tidak pernah gagal pada OS yang didukung
		}
		b[i] = alfabet[n.Int64()]
	}
	return string(b)
}

func tokenAcak() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
