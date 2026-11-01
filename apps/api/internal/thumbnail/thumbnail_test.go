package thumbnail

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func quadrantJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{B: 255, A: 255}
			if x < w/2 && y < h/2 {
				c = color.RGBA{R: 255, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func withOrientation(t *testing.T, data []byte, value uint16) []byte {
	t.Helper()
	tiff := []byte("MM\x00\x2A\x00\x00\x00\x08")
	tiff = binary.BigEndian.AppendUint16(tiff, 1)
	tiff = binary.BigEndian.AppendUint16(tiff, tagOrient)
	tiff = binary.BigEndian.AppendUint16(tiff, 3)
	tiff = binary.BigEndian.AppendUint32(tiff, 1)
	tiff = binary.BigEndian.AppendUint16(tiff, value)
	tiff = append(tiff, 0, 0, 0, 0, 0, 0)
	body := append([]byte("Exif\x00\x00"), tiff...)
	segment := []byte{0xFF, markerAPP1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(body)+2))
	segment = append(segment, body...)
	return append(append(append([]byte{}, data[:2]...), segment...), data[2:]...)
}

func render(t *testing.T, data []byte, width int) image.Image {
	t.Helper()
	out, err := New(2).Render(context.Background(), bytes.NewReader(data), width)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func isRed(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 > 200 && g>>8 < 80 && b>>8 < 80
}

func TestRenderAppliesExifOrientation(t *testing.T) {
	img := render(t, withOrientation(t, quadrantJPEG(t, 80, 40), 6), 80)

	if got := img.Bounds(); got.Dx() != 40 || got.Dy() != 80 {
		t.Fatalf("size = %dx%d, want 40x80", got.Dx(), got.Dy())
	}
	if !isRed(img.At(30, 10)) {
		t.Errorf("top-right should be the red quadrant after a 90 degree clockwise turn")
	}
	if isRed(img.At(5, 10)) {
		t.Errorf("top-left should no longer be red")
	}
}

func TestRenderLimitsTheWidthOfTheUprightImage(t *testing.T) {
	img := render(t, withOrientation(t, quadrantJPEG(t, 800, 400), 6), 320)

	if got := img.Bounds(); got.Dx() != 320 || got.Dy() != 640 {
		t.Fatalf("size = %dx%d, want 320x640", got.Dx(), got.Dy())
	}
}

func TestRenderWithoutOrientationKeepsTheImageAsItIs(t *testing.T) {
	img := render(t, quadrantJPEG(t, 80, 40), 80)

	if got := img.Bounds(); got.Dx() != 80 || got.Dy() != 40 {
		t.Fatalf("size = %dx%d, want 80x40", got.Dx(), got.Dy())
	}
	if !isRed(img.At(10, 10)) {
		t.Errorf("top-left should stay red")
	}
}

func TestRenderDownscalesButNeverEnlarges(t *testing.T) {
	big := render(t, quadrantJPEG(t, 800, 400), 320)
	if got := big.Bounds(); got.Dx() != 320 || got.Dy() != 160 {
		t.Errorf("downscaled size = %dx%d, want 320x160", got.Dx(), got.Dy())
	}

	small := render(t, quadrantJPEG(t, 100, 50), 320)
	if got := small.Bounds(); got.Dx() != 100 || got.Dy() != 50 {
		t.Errorf("small image size = %dx%d, want 100x50", got.Dx(), got.Dy())
	}
}

func hugePNGHeader(width, height uint32) []byte {
	header := binary.BigEndian.AppendUint32(nil, width)
	header = binary.BigEndian.AppendUint32(header, height)
	header = append(header, 8, 2, 0, 0, 0)
	chunk := append([]byte("IHDR"), header...)

	out := []byte("\x89PNG\r\n\x1a\n")
	out = binary.BigEndian.AppendUint32(out, uint32(len(header)))
	out = append(out, chunk...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(chunk))
}

func TestRenderRefusesAnOversizedHeaderWithoutDecoding(t *testing.T) {
	_, err := New(1).Render(context.Background(), bytes.NewReader(hugePNGHeader(20000, 20000)), 320)

	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestRenderRefusesAnOversizedSource(t *testing.T) {
	_, err := New(1).Render(context.Background(), bytes.NewReader(make([]byte, MaxBytes+1)), 320)

	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestRenderRefusesNonImages(t *testing.T) {
	_, err := New(1).Render(context.Background(), bytes.NewReader([]byte("just some text")), 320)

	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestRenderGivesUpWhenTheCallerLeavesWhileWaiting(t *testing.T) {
	generator := New(1)
	if err := generator.acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := generator.Render(ctx, bytes.NewReader(quadrantJPEG(t, 10, 10)), 320)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestJPEGOrientationIgnoresMalformedData(t *testing.T) {
	for _, data := range [][]byte{nil, {0xFF}, {0xFF, 0xD8}, {0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xFF}, []byte("not a jpeg")} {
		if got := jpegOrientation(data); got != 1 {
			t.Errorf("jpegOrientation(%q) = %d, want 1", data, got)
		}
	}
}
