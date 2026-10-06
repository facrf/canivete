package imagemeta

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"io"
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
