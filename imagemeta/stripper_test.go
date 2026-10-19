package imagemeta

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func pngChunk(kind string, data []byte) []byte {
	result := make([]byte, 12+len(data))
	binary.BigEndian.PutUint32(result, uint32(len(data)))
	copy(result[4:8], kind)
	copy(result[8:], data)
	binary.BigEndian.PutUint32(result[8+len(data):], crc32.ChecksumIEEE(result[4:8+len(data)]))
	return result
}

func TestJPEGRejectsTruncatedAndEmptyImages(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, image.NewRGBA(image.Rect(0, 0, 3, 2)), nil); err != nil {
		t.Fatal(err)
	}
	data := source.Bytes()
	scan := bytes.LastIndex(data, []byte{0xff, 0xda})
	if scan < 0 {
		t.Fatal("encoded JPEG has no SOS marker")
	}
	headerLength := int(binary.BigEndian.Uint16(data[scan+2 : scan+4]))
	emptyScan := append([]byte{}, data[:scan+2+headerLength]...)
	emptyScan = append(emptyScan, 0xff, 0xd9)
	for _, input := range [][]byte{data[:len(data)-2], data[:len(data)-1], emptyScan, {0xff, 0xd8, 0xff, 0xd9}, {0xff, 0xd8}} {
		if _, err := StripAISignatures(context.Background(), bytes.NewReader(input), io.Discard, "jpeg"); err == nil {
			t.Fatalf("accepted invalid JPEG of %d bytes", len(input))
		}
	}
}

func TestJPEGRemovesMetadataWithoutChangingImageData(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, image.NewRGBA(image.Rect(0, 0, 3, 2)), nil); err != nil {
		t.Fatal(err)
	}
	original := source.Bytes()
	metadata := []byte("Exif\x00\x00Private GPS")
	segment := []byte{0xff, 0xe1, 0, byte(len(metadata) + 2)}
	segment = append(segment, metadata...)
	input := append([]byte{}, original[:2]...)
	input = append(input, segment...)
	input = append(input, original[2:]...)
	input = append(input, []byte("Private appended data")...)
	var output bytes.Buffer
	report, err := StripAISignatures(context.Background(), bytes.NewReader(input), &output, "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), original) || report.BytesSaved != int64(len(input)-len(original)) {
		t.Fatal("metadata retained or compressed image data changed")
	}
	if _, err := jpeg.Decode(bytes.NewReader(output.Bytes())); err != nil {
		t.Fatal(err)
	}
	_, err = StripAISignatures(context.Background(), io.MultiReader(bytes.NewReader(original), failedReader{}), io.Discard, "jpeg")
	if err == nil {
		t.Fatal("ignored trailing read failure")
	}
}

func TestJPEGProgressiveCompatibility(t *testing.T) {
	// Fixtures distributed with Go cover successive scans, chroma subsampling,
	// grayscale and progressive DC scans without adding image assets to the repo.
	fixtures, err := filepath.Glob(filepath.Join(runtime.GOROOT(), "src/image/testdata/*progressive.jpeg"))
	if err != nil || len(fixtures) == 0 {
		t.Skip("Go progressive JPEG fixtures unavailable")
	}
	for _, filename := range fixtures {
		t.Run(filepath.Base(filename), func(t *testing.T) {
			data, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if _, err := StripAISignatures(context.Background(), bytes.NewReader(data), &output, "jpeg"); err != nil {
				t.Fatal(err)
			}
			if _, err := jpeg.Decode(bytes.NewReader(output.Bytes())); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func webpChunk(kind string, data []byte) []byte {
	result := make([]byte, 8+len(data)+len(data)%2)
	copy(result[:4], kind)
	binary.LittleEndian.PutUint32(result[4:8], uint32(len(data)))
	copy(result[8:], data)
	return result
}

func webpContainer(chunks ...[]byte) []byte {
	result := []byte("RIFF\x00\x00\x00\x00WEBP")
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}
	binary.LittleEndian.PutUint32(result[4:8], uint32(len(result)-8))
	return result
}

func TestWebPRemovesMetadataUnknownChunksAndTrailingData(t *testing.T) {
	extended := make([]byte, 10)
	extended[0] = 0x3e // ICC profile, alpha, EXIF, XMP and animation.
	chunks := [][]byte{
		webpChunk("VP8X", extended), webpChunk("ICCP", []byte("color profile")),
		webpChunk("ANIM", []byte{0, 0, 0, 0, 0, 0}),
		webpChunk("ANMF", []byte("unchanged animation frame payload")),
		webpChunk("EXIF", []byte("Private GPS")), webpChunk("XMP ", []byte("Private author")),
		webpChunk("C2PA", []byte("Private provenance")), webpChunk("NOTE", []byte("Private custom metadata")),
	}
	input := webpContainer(chunks...)
	// This trailer looks like a valid chunk but lies outside the RIFF boundary.
	input = append(input, webpChunk("VP8L", []byte("Private appended GPS"))...)
	var output bytes.Buffer
	report, err := StripAISignatures(context.Background(), bytes.NewReader(input), &output, "")
	if err != nil {
		t.Fatal(err)
	}
	extended[0] &^= 0x0c
	expected := webpContainer(webpChunk("VP8X", extended), chunks[1], chunks[2], chunks[3])
	if !bytes.Equal(output.Bytes(), expected) {
		t.Fatal("metadata retained, flags incorrect, or color/animation payload changed")
	}
	if report.BytesSaved != int64(len(input)-len(expected)) || len(report.StrippedChunks) != 4 {
		t.Fatalf("incorrect report: %+v", report)
	}
}

func TestWebPRejectsInvalidBoundaries(t *testing.T) {
	valid := webpContainer(webpChunk("VP8L", []byte{1, 2}))
	overflow := append([]byte{}, valid...)
	binary.LittleEndian.PutUint32(overflow[16:20], ^uint32(0))
	truncatedHeader := append([]byte{}, valid[:len(valid)-2]...)
	oversizedRIFF := append([]byte{}, valid...)
	binary.LittleEndian.PutUint32(oversizedRIFF[4:8], uint32(len(valid)+10))
	for _, input := range [][]byte{
		overflow, truncatedHeader, oversizedRIFF, webpContainer(),
		webpContainer(webpChunk("VP8X", []byte{1})),
	} {
		if _, err := StripAISignatures(context.Background(), bytes.NewReader(input), io.Discard, "webp"); err == nil {
			t.Fatalf("accepted malformed WebP of %d bytes", len(input))
		}
	}
	_, err := StripAISignatures(context.Background(), io.MultiReader(bytes.NewReader(valid), failedReader{}), io.Discard, "webp")
	if err == nil {
		t.Fatal("ignored trailing read failure")
	}
}
func TestPNGRemovesMetadataAndAppendedData(t *testing.T) {
	var clean bytes.Buffer
	if err := png.Encode(&clean, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	original := clean.Bytes()
	metadata := pngChunk("tEXt", []byte("Author\x00Private name"))
	input := append([]byte{}, original[:len(original)-12]...)
	input = append(input, metadata...)
	input = append(input, original[len(original)-12:]...)
	input = append(input, []byte("Private appended GPS data")...)
	var output bytes.Buffer
	report, err := StripAISignatures(context.Background(), bytes.NewReader(input), &output, "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), original) {
		t.Fatal("metadata retained or pixels changed")
	}
	if report.BytesSaved != int64(len(input)-len(original)) {
		t.Fatalf("saved=%d", report.BytesSaved)
	}
	if _, err := png.Decode(bytes.NewReader(output.Bytes())); err != nil {
		t.Fatal(err)
	}
}
func TestPNGRejectsMissingEnd(t *testing.T) {
	var source bytes.Buffer
	png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	data := source.Bytes()
	if _, err := StripAISignatures(context.Background(), bytes.NewReader(data[:len(data)-12]), io.Discard, ""); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err=%v", err)
	}
}

type failedReader struct{}

func (failedReader) Read(p []byte) (int, error) { return 0, errors.New("read failure") }
func TestPNGPropagatesTrailingReadFailure(t *testing.T) {
	var source bytes.Buffer
	png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	_, err := StripAISignatures(context.Background(), io.MultiReader(bytes.NewReader(source.Bytes()), failedReader{}), io.Discard, "")
	if err == nil {
		t.Fatal("ignored read failure")
	}
}
