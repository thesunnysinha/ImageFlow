package imageproc

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func noisy(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 7), uint8(y * 13), uint8((x ^ y) * 5), 255})
		}
	}
	return img
}

func jpegBytes(t *testing.T, img image.Image, q int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: q}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestCompressShrinksAHighQualityJPEG(t *testing.T) {
	in := jpegBytes(t, noisy(400, 300), 100)
	out, ext, err := Compress(in, DefaultOptions())
	if err != nil || ext != "jpg" {
		t.Fatalf("ext=%q err=%v", ext, err)
	}
	if len(out) >= len(in) {
		t.Fatalf("expected smaller output: %d >= %d", len(out), len(in))
	}
}

func TestCompressResizesLargeImages(t *testing.T) {
	o := DefaultOptions()
	o.MaxDim = 100
	out, _, err := Compress(jpegBytes(t, noisy(400, 200), 90), o)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(out))
	if cfg.Width != 100 || cfg.Height != 50 {
		t.Fatalf("got %dx%d, want 100x50", cfg.Width, cfg.Height)
	}
}

func TestCompressKeepsPNGAndAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	img.Set(3, 3, color.NRGBA{255, 0, 0, 128})
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	out, ext, err := Compress(b.Bytes(), DefaultOptions())
	if err != nil || ext != "png" {
		t.Fatalf("ext=%q err=%v", ext, err)
	}
	if _, f, err := image.Decode(bytes.NewReader(out)); err != nil || f != "png" {
		t.Fatalf("output is not a PNG: %v %s", err, f)
	}
}

func TestCompressRejectsBadInput(t *testing.T) {
	small := DefaultOptions()
	small.MaxPixels = 100
	cases := map[string][]byte{
		"not an image":    []byte("hello world"),
		"too many pixels": jpegBytes(t, noisy(50, 50), 80),
		"truncated":       jpegBytes(t, noisy(200, 200), 80)[:200],
	}
	for name, data := range cases {
		o := DefaultOptions()
		if name == "too many pixels" {
			o = small
		}
		if _, _, err := Compress(data, o); !IsPermanent(err) {
			t.Errorf("%s: want a permanent error, got %v", name, err)
		}
	}
}

func TestFetchClassifiesErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("data"))
		case "/gone":
			w.WriteHeader(404)
		case "/busy":
			w.WriteHeader(503)
		case "/big":
			_, _ = w.Write([]byte(strings.Repeat("x", 100)))
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	if b, err := Fetch(ctx, srv.Client(), srv.URL+"/ok", 10); err != nil || string(b) != "data" {
		t.Fatalf("ok: %q %v", b, err)
	}
	if _, err := Fetch(ctx, srv.Client(), srv.URL+"/gone", 10); !IsPermanent(err) {
		t.Errorf("404 must be permanent, got %v", err)
	}
	if _, err := Fetch(ctx, srv.Client(), srv.URL+"/busy", 10); err == nil || IsPermanent(err) {
		t.Errorf("503 must be retryable, got %v", err)
	}
	if _, err := Fetch(ctx, srv.Client(), srv.URL+"/big", 10); !IsPermanent(err) {
		t.Errorf("oversize must be permanent, got %v", err)
	}
}
