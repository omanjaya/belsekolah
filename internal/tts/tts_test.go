package tts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// skrip membuat program edge-tts palsu berisi isi shell yang diberikan.
func skrip(t *testing.T, isi string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "edge-tts")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+isi), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

const bacaArgumen = `
while [ $# -gt 0 ]; do
  case "$1" in
    --voice) suara="$2"; shift 2;;
    --file) teks=$(cat "$2"); shift 2;;
    --write-media) keluar="$2"; shift 2;;
    *) shift;;
  esac
done
`

func TestBuatBerhasil(t *testing.T) {
	log := filepath.Join(t.TempDir(), "log")
	p := skrip(t, bacaArgumen+fmt.Sprintf(`printf '%%s|%%s' "$suara" "$teks" > %q
printf 'ID3-suara-palsu' > "$keluar"
`, log))
	teks := `Selamat pagi! "Kutip" & <tanda> $HOME`
	b, err := EdgeTTS{Perintah: p}.Buat(context.Background(), teks, "id-ID-GadisNeural")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ID3-suara-palsu" {
		t.Errorf("hasil = %q", b)
	}
	got, _ := os.ReadFile(log)
	if string(got) != "id-ID-GadisNeural|"+teks {
		t.Errorf("argumen = %q (teks harus diteruskan apa adanya lewat file)", got)
	}
}

func TestSuaraTidakDikenal(t *testing.T) {
	_, err := EdgeTTS{Perintah: "/bin/true"}.Buat(context.Background(), "halo", "en-US-AriaNeural")
	if !errors.Is(err, ErrSuaraTidakDikenal) {
		t.Fatalf("err = %v", err)
	}
	if !SuaraDikenal("id-ID-ArdiNeural") || SuaraDikenal("") {
		t.Error("SuaraDikenal salah")
	}
	if len(DaftarSuara) != 2 || DaftarSuara[0].ID != "id-ID-GadisNeural" || DaftarSuara[1].Nama != "Ardi (laki-laki)" {
		t.Errorf("DaftarSuara = %+v", DaftarSuara)
	}
}

func TestTidakTerpasang(t *testing.T) {
	for _, p := range []string{"/tidak/ada/edge-tts", "edge-tts-yang-tidak-ada"} {
		_, err := EdgeTTS{Perintah: p}.Buat(context.Background(), "halo", "id-ID-GadisNeural")
		if !errors.Is(err, ErrTidakTerpasang) {
			t.Errorf("%s: err = %v, mau ErrTidakTerpasang", p, err)
		}
	}
}

func TestGagalJaringan(t *testing.T) {
	p := skrip(t, `echo "aiohttp.client_exceptions.ClientConnectorError: Cannot connect to host" >&2; exit 1`)
	_, err := EdgeTTS{Perintah: p}.Buat(context.Background(), "halo", "id-ID-GadisNeural")
	if !errors.Is(err, ErrGagal) {
		t.Fatalf("err = %v, mau ErrGagal", err)
	}
}

func TestHasilKosong(t *testing.T) {
	p := skrip(t, "exit 0")
	_, err := EdgeTTS{Perintah: p}.Buat(context.Background(), "halo", "id-ID-GadisNeural")
	if !errors.Is(err, ErrGagal) {
		t.Fatalf("err = %v, mau ErrGagal", err)
	}
}

func TestBatasWaktu(t *testing.T) {
	p := skrip(t, "sleep 5")
	mulai := time.Now()
	_, err := EdgeTTS{Perintah: p, Batas: 200 * time.Millisecond}.Buat(context.Background(), "halo", "id-ID-GadisNeural")
	if !errors.Is(err, ErrGagal) {
		t.Fatalf("err = %v, mau ErrGagal", err)
	}
	if time.Since(mulai) > 3*time.Second {
		t.Error("batas waktu tidak dihormati")
	}
}
