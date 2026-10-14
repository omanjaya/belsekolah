package nada

import (
	"encoding/binary"
	"testing"
)

func periksaWAV(t *testing.T, nama string, b []byte) {
	t.Helper()
	if len(b) < 44 {
		t.Fatalf("%s: terlalu pendek (%d byte)", nama, len(b))
	}
	if string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" || string(b[12:16]) != "fmt " || string(b[36:40]) != "data" {
		t.Fatalf("%s: header WAV tidak valid", nama)
	}
	if got := binary.LittleEndian.Uint32(b[4:]); int(got) != len(b)-8 {
		t.Errorf("%s: ukuran RIFF = %d, mau %d", nama, got, len(b)-8)
	}
	if got := binary.LittleEndian.Uint32(b[40:]); int(got) != len(b)-44 {
		t.Errorf("%s: ukuran data = %d, mau %d", nama, got, len(b)-44)
	}
	if binary.LittleEndian.Uint16(b[20:]) != 1 || binary.LittleEndian.Uint16(b[22:]) != 1 ||
		binary.LittleEndian.Uint32(b[24:]) != 44100 || binary.LittleEndian.Uint16(b[34:]) != 16 {
		t.Errorf("%s: format harus PCM mono 44100 Hz 16-bit", nama)
	}
	var puncak int
	for i := 44; i+1 < len(b); i += 2 {
		v := int(int16(binary.LittleEndian.Uint16(b[i:])))
		if v < 0 {
			v = -v
		}
		if v > puncak {
			puncak = v
		}
	}
	if puncak < 10000 {
		t.Errorf("%s: suara terlalu pelan (puncak %d)", nama, puncak)
	}
}

func TestSemuaNadaValid(t *testing.T) {
	periksaWAV(t, "cadangan", Cadangan())
	for _, b := range Daftar {
		periksaWAV(t, b.Nama, b.Data())
	}
}

func TestDurasi(t *testing.T) {
	kasus := []struct {
		nama      string
		data      []byte
		min, maks int64
	}{
		{"DingDong", DingDong(), 1500, 2500},
		{"DingDongPanjang", DingDongPanjang(), 2500, 4500},
		{"Sirene", Sirene(), 4000, 5000},
	}
	for _, k := range kasus {
		d := DurasiMS(k.data)
		if d < k.min || d > k.maks {
			t.Errorf("%s durasi = %d ms, mau %d..%d", k.nama, d, k.min, k.maks)
		}
	}
}

func TestDaftarUnik(t *testing.T) {
	nama, file := map[string]bool{}, map[string]bool{}
	for _, b := range Daftar {
		if nama[b.Nama] || file[b.File] {
			t.Errorf("duplikat: %+v", b)
		}
		nama[b.Nama], file[b.File] = true, true
	}
	if len(Daftar) != 3 {
		t.Errorf("jumlah nada bawaan = %d, mau 3", len(Daftar))
	}
}
