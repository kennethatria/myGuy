package media

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "GPS 0.3476N 32.5842E"

// exifSegment builds an APP1 Exif block: IFD0 with an Orientation tag, plus
// a trailing GPS-looking payload that must not survive cleaning.
func exifSegment(orientation uint16, order binary.ByteOrder) []byte {
	tiff := &bytes.Buffer{}
	if order == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	_ = binary.Write(tiff, order, uint16(42))
	_ = binary.Write(tiff, order, uint32(8)) // IFD0 at 8
	_ = binary.Write(tiff, order, uint16(1)) // one entry
	_ = binary.Write(tiff, order, uint16(0x0112))
	_ = binary.Write(tiff, order, uint16(3)) // SHORT
	_ = binary.Write(tiff, order, uint32(1))
	_ = binary.Write(tiff, order, orientation)
	_ = binary.Write(tiff, order, uint16(0))
	_ = binary.Write(tiff, order, uint32(0)) // no next IFD
	tiff.WriteString(secret)

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

// photo is a 4x2 JPEG, red on the left half and blue on the right, with an
// EXIF block inserted after the start-of-image marker.
func photo(t *testing.T, orientation uint16, order binary.ByteOrder) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if x >= 20 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	buf := &bytes.Buffer{}
	require.NoError(t, jpeg.Encode(buf, img, &jpeg.Options{Quality: 95}))
	raw := buf.Bytes()
	return append(append(append([]byte{}, raw[:2]...), exifSegment(orientation, order)...), raw[2:]...)
}

// pngChunk builds a PNG chunk with a valid checksum
func pngChunk(kind string, data []byte) []byte {
	c := make([]byte, 4, 12+len(data))
	binary.BigEndian.PutUint32(c, uint32(len(data)))
	c = append(append(c, kind...), data...)
	sum := make([]byte, 4)
	binary.BigEndian.PutUint32(sum, crc32.ChecksumIEEE(c[4:]))
	return append(c, sum...)
}

func isRed(c color.Color) bool {
	r, _, b, _ := c.RGBA()
	return r > 0xC000 && b < 0x4000
}

func TestCleanJPEGDropsMetadata(t *testing.T) {
	in := photo(t, 1, binary.BigEndian)
	require.Contains(t, string(in), secret)

	out := &bytes.Buffer{}
	ext, err := Clean(bytes.NewReader(in), out)

	require.NoError(t, err)
	assert.Equal(t, ".jpg", ext)
	assert.NotContains(t, out.String(), secret)
	assert.NotContains(t, out.String(), "Exif")
	img, err := jpeg.Decode(bytes.NewReader(out.Bytes()))
	require.NoError(t, err)
	assert.Equal(t, image.Pt(40, 20), img.Bounds().Size())
}

func TestCleanJPEGKeepsPhotosUpright(t *testing.T) {
	// Where the red half ends up for each orientation of a 40x20 photo
	cases := []struct {
		orientation uint16
		size        image.Point
		redAt       image.Point
	}{
		{2, image.Pt(40, 20), image.Pt(35, 10)}, // mirrored: red on the right
		{3, image.Pt(40, 20), image.Pt(35, 10)}, // upside down
		{4, image.Pt(40, 20), image.Pt(5, 10)},  // upside down, mirrored
		{5, image.Pt(20, 40), image.Pt(10, 5)},  // mirrored, turned
		{6, image.Pt(20, 40), image.Pt(10, 5)},  // turned clockwise: red on top
		{7, image.Pt(20, 40), image.Pt(10, 35)}, // mirrored, turned the other way
		{8, image.Pt(20, 40), image.Pt(10, 35)}, // turned anticlockwise: red below
	}
	for _, tc := range cases {
		for _, order := range []binary.ByteOrder{binary.BigEndian, binary.LittleEndian} {
			out := &bytes.Buffer{}
			_, err := Clean(bytes.NewReader(photo(t, tc.orientation, order)), out)
			require.NoError(t, err)
			img, err := jpeg.Decode(bytes.NewReader(out.Bytes()))
			require.NoError(t, err)
			assert.Equal(t, tc.size, img.Bounds().Size(), "orientation %d", tc.orientation)
			assert.True(t, isRed(img.At(tc.redAt.X, tc.redAt.Y)), "orientation %d (%v): red expected at %v", tc.orientation, order, tc.redAt)
		}
	}
}

func TestCleanPNGAndGIF(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	pngBuf := &bytes.Buffer{}
	require.NoError(t, png.Encode(pngBuf, img))
	// a tEXt chunk with a secret, spliced in before IEND
	raw := pngBuf.Bytes()
	chunk := pngChunk("tEXt", []byte(secret))
	withText := append(append(append([]byte{}, raw[:len(raw)-12]...), chunk...), raw[len(raw)-12:]...)

	out := &bytes.Buffer{}
	ext, err := Clean(bytes.NewReader(withText), out)
	require.NoError(t, err)
	assert.Equal(t, ".png", ext)
	assert.NotContains(t, out.String(), secret)

	pal := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.Black, color.White})
	gifBuf := &bytes.Buffer{}
	require.NoError(t, gif.EncodeAll(gifBuf, &gif.GIF{Image: []*image.Paletted{pal, pal}, Delay: []int{5, 5}}))
	out.Reset()
	ext, err = Clean(bytes.NewReader(gifBuf.Bytes()), out)
	require.NoError(t, err)
	assert.Equal(t, ".gif", ext)
	anim, err := gif.DecodeAll(bytes.NewReader(out.Bytes()))
	require.NoError(t, err)
	assert.Len(t, anim.Image, 2, "animation frames kept")
}

func TestCleanRefusesNonImagesAndHugeOnes(t *testing.T) {
	_, err := Clean(bytes.NewReader([]byte("<?php echo 'hi'; ?>")), &bytes.Buffer{})
	assert.ErrorIs(t, err, ErrNotImage)

	// a PNG header claiming 100000 x 100000 pixels
	huge := &bytes.Buffer{}
	require.NoError(t, png.Encode(huge, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	b := huge.Bytes()
	binary.BigEndian.PutUint32(b[16:], 100000)
	binary.BigEndian.PutUint32(b[20:], 100000)
	binary.BigEndian.PutUint32(b[29:], crc32.ChecksumIEEE(b[12:29])) // IHDR checksum
	_, err = Clean(bytes.NewReader(b), &bytes.Buffer{})
	assert.ErrorIs(t, err, ErrTooLarge)
}

func TestOrientationReaderToleratesJunk(t *testing.T) {
	assert.Equal(t, 1, jpegOrientation([]byte{1, 2}))
	assert.Equal(t, 1, jpegOrientation([]byte{0xFF, 0xD8, 0x00, 0x00}))
	assert.Equal(t, 1, jpegOrientation([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x01}))
	assert.Equal(t, 1, tiffOrientation([]byte("XX\x00\x2a\x00\x00\x00\x08")))
	assert.Equal(t, 1, tiffOrientation([]byte("II")))
}

func TestCleanExisting(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "1"), 0755))
	old := filepath.Join(dir, "1", "1791203509_0.jpeg")
	require.NoError(t, os.WriteFile(old, photo(t, 6, binary.BigEndian), 0644))
	notes := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(notes, []byte("not a photo"), 0644))

	n, err := CleanExisting(dir)

	require.NoError(t, err)
	assert.Equal(t, 1, n)
	after, _ := os.ReadFile(old)
	assert.NotContains(t, string(after), secret, "metadata gone, same file name")
	img, err := jpeg.Decode(bytes.NewReader(after))
	require.NoError(t, err)
	assert.Equal(t, image.Pt(20, 40), img.Bounds().Size(), "turned upright")
	kept, _ := os.ReadFile(notes)
	assert.Equal(t, "not a photo", string(kept))

	again, err := CleanExisting(dir)
	require.NoError(t, err)
	assert.Zero(t, again, "runs once")

	none, err := CleanExisting(filepath.Join(dir, "missing"))
	require.NoError(t, err)
	assert.Zero(t, none)
}
