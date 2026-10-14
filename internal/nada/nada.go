// Package nada menyintesis suara bel bawaan (WAV) tanpa file eksternal,
// sehingga aplikasi selalu punya suara walau folder audio kosong atau rusak.
package nada

import (
	"encoding/binary"
	"math"
)

const sampleRate = 44100

// Bawaan adalah satu suara bawaan yang dipasang ke folder audio.
type Bawaan struct {
	Nama string
	File string
	Data func() []byte
}

var Daftar = []Bawaan{
	{Nama: "Ding Dong", File: "bawaan-ding-dong.wav", Data: DingDong},
	{Nama: "Ding Dong Panjang", File: "bawaan-ding-dong-panjang.wav", Data: DingDongPanjang},
	{Nama: "Sirene", File: "bawaan-sirene.wav", Data: Sirene},
}

type not struct {
	frek   float64
	durasi float64 // detik
}

// DingDong: dua nada turun E5 → C5.
func DingDong() []byte {
	return wav(render([]not{{659.25, 0.7}, {523.25, 1.1}}, 3))
}

// DingDongPanjang: empat nada naik C5 → E5 → G5 → C6.
func DingDongPanjang() []byte {
	return wav(render([]not{{523.25, 0.6}, {659.25, 0.6}, {783.99, 0.6}, {1046.5, 1.4}}, 2.5))
}

// Sirene: frekuensi naik-turun 600–1200 Hz, tiga siklus.
func Sirene() []byte {
	const durasi = 4.5
	const siklus = 1.5
	s := make([]int16, int(durasi*sampleRate))
	fase := 0.0
	for i := range s {
		t := float64(i) / sampleRate
		posisi := math.Mod(t, siklus) / siklus
		frek := 600 + 600*(1-math.Abs(2*posisi-1))
		fase += 2 * math.Pi * frek / sampleRate
		s[i] = int16(math.Sin(fase) * 0.7 * math.MaxInt16)
	}
	return wav(s)
}

// Cadangan diputar jika file audio sebuah bel hilang atau rusak.
func Cadangan() []byte {
	return DingDong()
}

// DurasiMS menghitung durasi WAV hasil paket ini.
func DurasiMS(b []byte) int64 {
	if len(b) <= 44 {
		return 0
	}
	sampel := int64(len(b)-44) / 2
	return sampel * 1000 / sampleRate
}

// render membuat nada lonceng: nada dasar + harmonik kedua, peluruhan eksponensial.
func render(nots []not, peluruhan float64) []int16 {
	var out []int16
	for _, n := range nots {
		jumlah := int(n.durasi * sampleRate)
		for i := 0; i < jumlah; i++ {
			t := float64(i) / sampleRate
			amp := math.Exp(-peluruhan * t)
			if t < 0.005 {
				amp *= t / 0.005 // serangan halus agar tidak "klik"
			}
			v := (math.Sin(2*math.Pi*n.frek*t) + 0.3*math.Sin(4*math.Pi*n.frek*t)) / 1.3
			out = append(out, int16(v*amp*0.8*math.MaxInt16))
		}
	}
	return out
}

func wav(s []int16) []byte {
	dataLen := len(s) * 2
	b := make([]byte, 44+dataLen)
	copy(b[0:], "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+dataLen))
	copy(b[8:], "WAVE")
	copy(b[12:], "fmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)           // ukuran chunk fmt
	binary.LittleEndian.PutUint16(b[20:], 1)            // PCM
	binary.LittleEndian.PutUint16(b[22:], 1)            // mono
	binary.LittleEndian.PutUint32(b[24:], sampleRate)   // sample rate
	binary.LittleEndian.PutUint32(b[28:], sampleRate*2) // byte rate
	binary.LittleEndian.PutUint16(b[32:], 2)            // block align
	binary.LittleEndian.PutUint16(b[34:], 16)           // bit per sampel
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(dataLen))
	for i, v := range s {
		binary.LittleEndian.PutUint16(b[44+i*2:], uint16(v))
	}
	return b
}
