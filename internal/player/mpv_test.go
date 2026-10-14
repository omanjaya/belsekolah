package player

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMpvBerhasil(t *testing.T) {
	if err := (Mpv{Perintah: "true"}).Run(context.Background(), "x.wav", 50); err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestArgumenMpvAO(t *testing.T) {
	for _, a := range argumenMpv("a.wav", 50, "") {
		if strings.HasPrefix(a, "--ao=") {
			t.Fatalf("tanpa env tidak boleh ada --ao")
		}
	}
	dengan := argumenMpv("a.wav", 50, "null")
	ada := false
	for _, a := range dengan {
		ada = ada || a == "--ao=null"
	}
	if !ada || dengan[len(dengan)-2] != "--" || dengan[len(dengan)-1] != "a.wav" {
		t.Fatalf("argumen = %v", dengan)
	}
}

func TestMpvGagal(t *testing.T) {
	if err := (Mpv{Perintah: "false"}).Run(context.Background(), "x.wav", 50); err == nil {
		t.Fatal("seharusnya error")
	}
}

func TestMpvDibatalkan(t *testing.T) {
	skrip := filepath.Join(t.TempDir(), "tidur.sh")
	if err := os.WriteFile(skrip, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	selesai := make(chan error, 1)
	go func() { selesai <- (Mpv{Perintah: skrip}).Run(ctx, "x.wav", 50) }()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-selesai:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, mau context.Canceled", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("Run tidak kembali setelah ctx dibatalkan")
	}
}
