package thumbnail

import (
	"encoding/binary"
	"image"
)

const (
	markerSOS  = 0xDA
	markerAPP1 = 0xE1
	tagOrient  = 0x0112
)

type segment struct {
	marker byte
	body   []byte
	next   int
}

func readSegment(data []byte, pos int) (segment, bool) {
	if pos+4 > len(data) || data[pos] != 0xFF {
		return segment{}, false
	}
	size := int(binary.BigEndian.Uint16(data[pos+2:]))
	end := pos + 2 + size
	if size < 2 || end > len(data) {
		return segment{}, false
	}
	return segment{marker: data[pos+1], body: data[pos+4 : end], next: end}, true
}

func jpegOrientation(data []byte) int {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for {
		seg, ok := readSegment(data, pos)
		if !ok || seg.marker == markerSOS {
			return 1
		}
		if value := segmentOrientation(seg); value != 0 {
			return value
		}
		pos = seg.next
	}
}

func segmentOrientation(seg segment) int {
	if seg.marker != markerAPP1 {
		return 0
	}
	return tiffOrientation(seg.body)
}

func tiffOrientation(body []byte) int {
	if len(body) < 14 || string(body[:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := body[6:]
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	return ifdOrientation(tiff, int(order.Uint32(tiff[4:8])), order)
}

func ifdOrientation(tiff []byte, offset int, order binary.ByteOrder) int {
	if offset < 8 || offset+2 > len(tiff) {
		return 0
	}
	count := int(order.Uint16(tiff[offset:]))
	for i := 0; i < count; i++ {
		entry := offset + 2 + i*12
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:]) != tagOrient {
			continue
		}
		value := int(order.Uint16(tiff[entry+8:]))
		if value < 1 || value > 8 {
			return 0
		}
		return value
	}
	return 0
}

func orient(src *image.RGBA, value int) *image.RGBA {
	if value <= 1 || value > 8 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if value >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx, sy := sourcePoint(value, x, y, w, h)
			dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

func sourcePoint(value, x, y, w, h int) (int, int) {
	switch value {
	case 2:
		return w - 1 - x, y
	case 3:
		return w - 1 - x, h - 1 - y
	case 4:
		return x, h - 1 - y
	case 5:
		return y, x
	case 6:
		return y, h - 1 - x
	case 7:
		return w - 1 - y, h - 1 - x
	default:
		return w - 1 - y, x
	}
}
