// Package thumbnail renders small JPEG previews of stored images so a folder
// of full-size photos does not have to be downloaded to draw its grid.
package thumbnail

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"

	_ "image/gif"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	// MaxPixels is the largest source, in pixels, that Render will decode.
	MaxPixels = 40_000_000
	// MaxBytes is the largest source, in bytes, that Render will read.
	MaxBytes = 30 << 20
	// Quality is the JPEG quality thumbnails are encoded at.
	Quality = 80
)

// ErrUnsupported means the source is not an image this package can decode.
var ErrUnsupported = errors.New("thumbnail: unsupported image")

// ErrTooLarge means the source exceeds MaxBytes or MaxPixels.
var ErrTooLarge = errors.New("thumbnail: image too large")

// Generator renders thumbnails with a bounded number of decodes in flight, so a
// grid of large photos cannot exhaust the API's memory.
type Generator struct {
	slots chan struct{}
}

// New returns a Generator that runs at most concurrency renders at once.
func New(concurrency int) *Generator {
	return &Generator{slots: make(chan struct{}, max(1, concurrency))}
}

// Render decodes src and returns it as a JPEG no wider than width, upright
// according to its EXIF orientation: the width limit applies to the upright
// image, so a portrait photo is as wide as a landscape one. A source narrower
// than width is not enlarged. It returns ErrTooLarge or ErrUnsupported for sources it refuses.
func (g *Generator) Render(ctx context.Context, src io.Reader, width int) ([]byte, error) {
	if err := g.acquire(ctx); err != nil {
		return nil, err
	}
	defer g.release()

	data, err := readBounded(src)
	if err != nil {
		return nil, err
	}
	img, err := decode(data)
	if err != nil {
		return nil, err
	}
	turn := jpegOrientation(data)
	return encode(orient(scale(img, width, turn >= 5), turn))
}

func (g *Generator) acquire(ctx context.Context) error {
	select {
	case g.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *Generator) release() { <-g.slots }

func readBounded(src io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(src, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

func decode(data []byte) (image.Image, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrUnsupported
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return nil, ErrTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupported
	}
	return img, nil
}

func scale(img image.Image, width int, quarterTurn bool) *image.RGBA {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	shown := w
	if quarterTurn {
		shown = h
	}
	if width > 0 && width < shown {
		w = max(1, int(int64(w)*int64(width)/int64(shown)))
		h = max(1, int(int64(h)*int64(width)/int64(shown)))
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)
	return dst
}

func encode(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: Quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
