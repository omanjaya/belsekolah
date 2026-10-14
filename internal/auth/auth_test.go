package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"belsekolah/internal/store"
)

var ctx = context.Background()

type petaPenyimpan struct {
	mu   sync.Mutex
	data map[string]string
}

func (p *petaPenyimpan) GetSetting(_ context.Context, k string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.data[k]
	if !ok {
		return "", store.ErrTidakAda
	}
	return v, nil
}

func (p *petaPenyimpan) SetSetting(_ context.Context, k, v string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data[k] = v
	return nil
}

type jam struct{ t time.Time }

func (j *jam) now() time.Time       { return j.t }
func (j *jam) maju(d time.Duration) { j.t = j.t.Add(d) }

func siapkan(t *testing.T) (*Auth, *jam, string) {
	t.Helper()
	BiayaHash = bcrypt.MinCost
	j := &jam{t: time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)}
	a := New(&petaPenyimpan{data: map[string]string{}}, j.now)
	pw, dibuat, err := a.SiapkanPasswordAwal(ctx)
	if err != nil || !dibuat || len(pw) != 10 {
		t.Fatalf("SiapkanPasswordAwal = %q, %v, %v", pw, dibuat, err)
	}
	return a, j, pw
}

func TestPasswordAwalHanyaSekali(t *testing.T) {
	a, _, _ := siapkan(t)
	pw, dibuat, err := a.SiapkanPasswordAwal(ctx)
	if err != nil || dibuat || pw != "" {
		t.Fatalf("panggilan kedua = %q, %v, %v; mau tidak membuat ulang", pw, dibuat, err)
	}
	if w, _ := a.WajibGanti(ctx); !w {
		t.Error("password awal harus wajib diganti")
	}
}

func TestPasswordAcak(t *testing.T) {
	a, b := PasswordAcak(), PasswordAcak()
	if len(a) != 10 || a == b {
		t.Fatalf("PasswordAcak = %q, %q", a, b)
	}
	if strings.ContainsAny(a, "il1o0") {
		t.Errorf("password %q memuat karakter yang mudah tertukar", a)
	}
}

func TestLoginDanSesi(t *testing.T) {
	a, j, pw := siapkan(t)
	if _, err := a.Login(ctx, "10.0.0.2", "salah"); !errors.Is(err, ErrPasswordSalah) {
		t.Fatalf("login salah: %v", err)
	}
	token, err := a.Login(ctx, "10.0.0.2", pw)
	if err != nil || len(token) < 32 {
		t.Fatalf("login benar: %q, %v", token, err)
	}
	if !a.Valid(token) || a.Valid("palsu") || a.Valid("") {
		t.Fatal("validasi token salah")
	}
	j.maju(6 * 24 * time.Hour)
	if !a.Valid(token) {
		t.Fatal("sesi harus masih berlaku setelah 6 hari")
	}
	j.maju(6 * 24 * time.Hour) // 6 hari sejak terakhir dipakai: masih berlaku (sliding)
	if !a.Valid(token) {
		t.Fatal("sesi harus diperpanjang setiap dipakai")
	}
	j.maju(7*24*time.Hour + time.Second)
	if a.Valid(token) {
		t.Fatal("sesi harus kedaluwarsa setelah 7 hari tidak dipakai")
	}
	token2, _ := a.Login(ctx, "10.0.0.2", pw)
	a.Logout(token2)
	if a.Valid(token2) {
		t.Fatal("token harus tidak berlaku setelah logout")
	}
}

func TestKunciSetelahLimaGagal(t *testing.T) {
	a, j, pw := siapkan(t)
	for i := 1; i <= 4; i++ {
		if _, err := a.Login(ctx, "10.0.0.9", "salah"); !errors.Is(err, ErrPasswordSalah) {
			t.Fatalf("percobaan %d: %v", i, err)
		}
	}
	if _, err := a.Login(ctx, "10.0.0.9", "salah"); !errors.Is(err, ErrTerkunci) {
		t.Fatalf("percobaan ke-5: %v, mau ErrTerkunci", err)
	}
	if _, err := a.Login(ctx, "10.0.0.9", pw); !errors.Is(err, ErrTerkunci) {
		t.Fatalf("password benar saat terkunci: %v, mau ErrTerkunci", err)
	}
	if _, err := a.Login(ctx, "10.0.0.10", pw); err != nil {
		t.Fatalf("IP lain tidak boleh ikut terkunci: %v", err)
	}
	j.maju(5*time.Minute + time.Second)
	if _, err := a.Login(ctx, "10.0.0.9", pw); err != nil {
		t.Fatalf("setelah 5 menit harus bisa login: %v", err)
	}
}

func TestGantiPassword(t *testing.T) {
	a, _, pw := siapkan(t)
	if err := a.GantiPassword(ctx, "salah", "rahasia-baru"); !errors.Is(err, ErrPasswordSalah) {
		t.Errorf("lama salah: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, "pendek"); !errors.Is(err, ErrPasswordPendek) {
		t.Errorf("baru pendek: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, strings.Repeat("x", 73)); !errors.Is(err, ErrPasswordPanjang) {
		t.Errorf("baru terlalu panjang: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, pw); !errors.Is(err, ErrPasswordSama) {
		t.Errorf("baru sama dengan lama: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, "rahasia-baru"); err != nil {
		t.Fatal(err)
	}
	if w, _ := a.WajibGanti(ctx); w {
		t.Error("setelah diganti tidak lagi wajib ganti")
	}
	if _, err := a.Login(ctx, "1.1.1.1", pw); !errors.Is(err, ErrPasswordSalah) {
		t.Error("password lama tidak boleh berlaku lagi")
	}
	if _, err := a.Login(ctx, "1.1.1.1", "rahasia-baru"); err != nil {
		t.Errorf("password baru: %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	a, _, pw := siapkan(t)
	if err := a.GantiPassword(ctx, pw, "rahasia-baru"); err != nil {
		t.Fatal(err)
	}
	token, _ := a.Login(ctx, "1.1.1.1", "rahasia-baru")
	baru, err := a.ResetPassword(ctx)
	if err != nil || len(baru) != 10 {
		t.Fatalf("ResetPassword = %q, %v", baru, err)
	}
	if a.Valid(token) {
		t.Error("reset harus menghapus semua sesi")
	}
	if w, _ := a.WajibGanti(ctx); !w {
		t.Error("password hasil reset wajib diganti")
	}
	if _, err := a.Login(ctx, "1.1.1.1", baru); err != nil {
		t.Errorf("login dengan password reset: %v", err)
	}
}

func TestBelumAdaPassword(t *testing.T) {
	BiayaHash = bcrypt.MinCost
	a := New(&petaPenyimpan{data: map[string]string{}}, time.Now)
	if _, err := a.Login(ctx, "1.1.1.1", "apa"); !errors.Is(err, ErrBelumAdaPassword) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoginParalelTidakMenembusKunci(t *testing.T) {
	a, _, pw := siapkan(t)
	const n = 20
	hasil := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := a.Login(ctx, "10.0.0.5", "salah")
			hasil <- err
		}()
	}
	wg.Wait()
	close(hasil)
	salah := 0
	for err := range hasil {
		switch {
		case errors.Is(err, ErrPasswordSalah):
			salah++
		case errors.Is(err, ErrTerkunci):
		default:
			t.Fatalf("error tak terduga: %v", err)
		}
	}
	if salah > BatasGagal {
		t.Errorf("ErrPasswordSalah = %d kali, maksimal %d", salah, BatasGagal)
	}
	if _, err := a.Login(ctx, "10.0.0.5", pw); !errors.Is(err, ErrTerkunci) {
		t.Fatalf("password benar setelah terkunci: %v, mau ErrTerkunci", err)
	}
}
