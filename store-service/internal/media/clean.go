// Package media cleans uploaded photos before they are stored. Phone photos
// carry EXIF metadata, often including the GPS position where they were
// taken; a listing photo must never publish where its seller lives. Each
// photo is decoded and encoded afresh, which drops every metadata block,
// after turning it the way its EXIF orientation says (otherwise portrait
// phone photos would show sideways).
package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
)

// MaxPixels refuses images whose header claims more pixels than any phone
// photo has, so a tiny file can't make the server allocate gigabytes.
const MaxPixels = 50_000_000

var (
	ErrNotImage = errors.New("not a JPEG, PNG or GIF image")
	ErrTooLarge = errors.New("image dimensions are too large")
)

// Clean reads an uploaded image and writes a metadata-free copy in the same
// format to w, returning that format's file extension.
func Clean(r io.Reader, w io.Writer) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", ErrNotImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxPixels {
		return "", ErrTooLarge
	}

	switch format {
	case "jpeg":
		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return "", ErrNotImage
		}
		return ".jpg", jpeg.Encode(w, orient(img, jpegOrientation(data)), &jpeg.Options{Quality: 88})
	case "png":
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return "", ErrNotImage
		}
		return ".png", png.Encode(w, img)
	case "gif":
		anim, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			return "", ErrNotImage
		}
		// EncodeAll writes frames, palette and timing only: comments and
		// application extensions are left behind
		return ".gif", gif.EncodeAll(w, &gif.GIF{Image: anim.Image, Delay: anim.Delay, Disposal: anim.Disposal, LoopCount: anim.LoopCount, Config: anim.Config, BackgroundIndex: anim.BackgroundIndex})
	default:
		return "", ErrNotImage
	}
}

// jpegOrientation reads the EXIF orientation (1-8) from a JPEG, or 1 when
// there is none or it can't be read.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xD9 || marker == 0xDA { // end of image / start of scan
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		segment := data[i+4 : i+2+size]
		if marker == 0xE1 && len(segment) > 6 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[ifd:]))
	for n := 0; n < count; n++ {
		entry := ifd + 2 + n*12
		if entry+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[entry:]) == 0x0112 { // Orientation, a SHORT
			if v := int(order.Uint16(tiff[entry+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// orient turns img upright for an EXIF orientation value.
func orient(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), img, b.Min, draw.Src)

	ow, oh := w, h
	if orientation >= 5 { // the four that swap width and height
		ow, oh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2: // mirrored
				dx, dy = w-1-x, y
			case 3: // upside down
				dx, dy = w-1-x, h-1-y
			case 4: // upside down, mirrored
				dx, dy = x, h-1-y
			case 5: // mirrored, turned
				dx, dy = y, x
			case 6: // turned clockwise
				dx, dy = h-1-y, x
			case 7: // mirrored, turned the other way
				dx, dy = h-1-y, w-1-x
			case 8: // turned anticlockwise
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(x, y))
		}
	}
	return dst
}
