// Package imageproc downloads an image safely and re-encodes it smaller.
package imageproc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	"golang.org/x/image/draw"
)

// PermanentError marks failures that a retry cannot fix (bad image, 4xx, too large).
type PermanentError struct{ Reason string }

func (e *PermanentError) Error() string { return e.Reason }

func permanent(format string, args ...any) error {
	return &PermanentError{Reason: fmt.Sprintf(format, args...)}
}

func IsPermanent(err error) bool {
	var p *PermanentError
	return errors.As(err, &p)
}

type Options struct {
	MaxBytes  int64 // largest download accepted
	MaxPixels int   // largest decoded image (width*height), guards decompression bombs
	MaxDim    int   // longest side after resizing
	Quality   int   // JPEG quality
}

func DefaultOptions() Options {
	return Options{MaxBytes: 20 << 20, MaxPixels: 40_000_000, MaxDim: 2048, Quality: 80}
}

// Fetch downloads rawURL with the given (SSRF-safe) client, reading at most maxBytes.
func Fetch(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, permanent("invalid URL")
	}
	req.Header.Set("User-Agent", "ImageFlow/1.0")
	req.Header.Set("Accept", "image/jpeg,image/png,*/*;q=0.5")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err // network errors are retried
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return nil, fmt.Errorf("source returned status %d", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, permanent("source returned status %d", resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return nil, permanent("image is larger than %d bytes", maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, permanent("image is larger than %d bytes", maxBytes)
	}
	return data, nil
}

// Compress decodes a JPEG or PNG, shrinks it to at most MaxDim on the longest side and
// re-encodes it (PNG stays PNG so transparency survives). It returns the original bytes
// when re-encoding would not make the file smaller. ext is "jpg" or "png".
func Compress(data []byte, o Options) (out []byte, ext string, err error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", permanent("not a supported image (JPEG or PNG)")
	}
	if format != "jpeg" && format != "png" {
		return nil, "", permanent("unsupported image format %q", format)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > o.MaxPixels {
		return nil, "", permanent("image dimensions are too large")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", permanent("image could not be decoded")
	}
	originalExt := map[string]string{"jpeg": "jpg", "png": "png"}[format]

	resized := false
	if w, h := cfg.Width, cfg.Height; w > o.MaxDim || h > o.MaxDim {
		scale := float64(o.MaxDim) / float64(max(w, h))
		nw, nh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
		img, resized = dst, true
	}

	var buf bytes.Buffer
	if format == "png" {
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err = enc.Encode(&buf, img)
		ext = "png"
	} else {
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: o.Quality})
		ext = "jpg"
	}
	if err != nil {
		return nil, "", err
	}
	if !resized && buf.Len() >= len(data) {
		return data, originalExt, nil
	}
	return buf.Bytes(), ext, nil
}
