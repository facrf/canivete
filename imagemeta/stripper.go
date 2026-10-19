package imagemeta

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Report represents the result of a metadata stripping operation.
type Report struct {
	Format         string
	StrippedChunks []string
	BytesSaved     int64
}

// StripAISignatures processes an image stream, removing metadata without re-encoding.
func StripAISignatures(ctx context.Context, r io.Reader, w io.Writer, format string) (Report, error) {
	format = strings.ToLower(format)
	if format == "" || format == "auto" {
		br := bufio.NewReader(r)
		format = sniffFormat(br)
		r = br
	}

	switch format {
	case "jpeg", "jpg":
		return stripJPEG(ctx, r, w)
	case "png":
		return stripPNG(ctx, r, w)
	case "webp":
		return stripWebP(ctx, r, w)
	default:
		return Report{}, errors.New("unsupported or unknown image format")
	}
}

func sniffFormat(br *bufio.Reader) string {
	sig, err := br.Peek(12)
	if err != nil && len(sig) < 3 {
		return ""
	}
	if len(sig) >= 3 && sig[0] == 0xFF && sig[1] == 0xD8 && sig[2] == 0xFF {
		return "jpeg"
	}
	if len(sig) >= 8 && bytes.Equal(sig[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "png"
	}
	if len(sig) >= 12 && string(sig[0:4]) == "RIFF" && string(sig[8:12]) == "WEBP" {
		return "webp"
	}
	return ""
}

func stripJPEG(ctx context.Context, r io.Reader, w io.Writer) (Report, error) {
	rep := Report{Format: "jpeg"}
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

	hasFrame, hasScan, hasEntropy := false, false, false

	soi := make([]byte, 2)
	if _, err := io.ReadFull(br, soi); err != nil {
		return rep, err
	}
	if soi[0] != 0xFF || soi[1] != 0xD8 {
		return rep, errors.New("not a valid JPEG")
	}
	if _, err := w.Write(soi); err != nil {
		return rep, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return rep, err
		}

		var b byte
		var err error
		for {
			b, err = br.ReadByte()
			if err != nil {
				if err == io.EOF {
					return rep, io.ErrUnexpectedEOF
				}
				return rep, err
			}
			if b == 0xFF {
				var b2 byte
				for {
					b2, err = br.ReadByte()
					if err != nil {
						return rep, err
					}
					if b2 != 0xFF {
						break
					}
				}
				if b2 == 0x00 {
					return rep, errors.New("unexpected stuffed byte outside JPEG scan")
				}
				b = b2
				break
			} else {
				return rep, errors.New("unexpected data outside JPEG scan")
			}
		}

		marker := b

	processMarker:
		if marker == 0xD8 {
			return rep, errors.New("unexpected JPEG start marker")
		}
		if marker == 0xD9 && (!hasFrame || !hasScan || !hasEntropy) {
			return rep, errors.New("JPEG has no image frame or scan data")
		}
		if (marker >= 0xD0 && marker <= 0xD9) || marker == 0x01 {
			if _, err := w.Write([]byte{0xFF, marker}); err != nil {
				return rep, err
			}
			if marker == 0xD9 {
				// EOI reached. We intentionally DROP any trailing data
				// to prevent appended metadata chunks.
				skipped, err := io.Copy(io.Discard, &contextReader{ctx: ctx, reader: br})
				rep.BytesSaved += skipped
				return rep, err
			}
			continue
		}

		lenBuf := make([]byte, 2)
		if _, err := io.ReadFull(br, lenBuf); err != nil {
			return rep, err
		}
		length := int(binary.BigEndian.Uint16(lenBuf))
		if length < 2 {
			return rep, errors.New("invalid JPEG marker length")
		}

		dataLen := length - 2
		isFrame := marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC
		if isFrame || marker == 0xDA {
			if dataLen < 6 || dataLen > 18 {
				return rep, errors.New("invalid JPEG frame or scan header")
			}
			data := make([]byte, dataLen)
			if _, err := io.ReadFull(br, data); err != nil {
				return rep, err
			}
			if isFrame {
				components := int(data[5])
				if components < 1 || components > 4 || dataLen != 6+3*components ||
					binary.BigEndian.Uint16(data[1:3]) == 0 || binary.BigEndian.Uint16(data[3:5]) == 0 {
					return rep, errors.New("invalid JPEG image frame")
				}
				hasFrame = true
			} else {
				components := int(data[0])
				if !hasFrame || components < 1 || components > 4 || dataLen != 4+2*components {
					return rep, errors.New("invalid JPEG scan")
				}
				hasScan = true
			}
			if _, err := w.Write(append([]byte{0xFF, marker, lenBuf[0], lenBuf[1]}, data...)); err != nil {
				return rep, err
			}
			dataLen = 0
		}
		strip := false
		chunkName := ""

		if marker >= 0xE0 && marker <= 0xEF {
			chunkName = fmt.Sprintf("APP%d", marker-0xE0)
			// Strip APP1 (XMP/Exif), APP11 (JUMBF/C2PA), APP13 (IPTC)
			if marker == 0xE1 || marker == 0xEB || marker == 0xED {
				strip = true
			}
		} else if marker == 0xFE {
			chunkName = "COM"
			strip = true
		}

		if strip {
			skipped, err := io.CopyN(io.Discard, br, int64(dataLen))
			if err != nil {
				return rep, err
			}
			rep.StrippedChunks = append(rep.StrippedChunks, chunkName)
			rep.BytesSaved += skipped + 4 // 2 marker + 2 length
		} else if !isFrame && marker != 0xDA {
			if _, err := w.Write([]byte{0xFF, marker, lenBuf[0], lenBuf[1]}); err != nil {
				return rep, err
			}
			if dataLen > 0 {
				written, err := io.CopyN(w, br, int64(dataLen))
				if err != nil {
					return rep, err
				}
				if written != int64(dataLen) {
					return rep, io.ErrUnexpectedEOF
				}
			}
		}

		if marker == 0xDA {
			// Entropy data starts. Fast-scan until we hit a marker that ends it.
			for {
				if err := ctx.Err(); err != nil {
					return rep, err
				}
				chunk, err := br.ReadSlice(0xFF)

				if err == bufio.ErrBufferFull {
					// 0xFF not found in this buffer segment
					hasEntropy = true
					if _, wErr := w.Write(chunk); wErr != nil {
						return rep, wErr
					}
					continue
				} else if err != nil && err != io.EOF {
					return rep, err
				}

				hasFF := len(chunk) > 0 && chunk[len(chunk)-1] == 0xFF
				if (!hasFF && len(chunk) > 0) || (hasFF && len(chunk) > 1) {
					hasEntropy = true
				}

				// Write everything EXCEPT the 0xFF (if present)
				if hasFF {
					if _, wErr := w.Write(chunk[:len(chunk)-1]); wErr != nil {
						return rep, wErr
					}
				} else {
					if _, wErr := w.Write(chunk); wErr != nil {
						return rep, wErr
					}
					if err == io.EOF {
						return rep, io.ErrUnexpectedEOF
					}
				}

				if hasFF {
					var b2 byte
					for {
						b2, err = br.ReadByte()
						if err != nil {
							if err == io.EOF {
								return rep, io.ErrUnexpectedEOF
							}
							return rep, err
						}
						if b2 != 0xFF {
							break
						}
						// Escaped/Padding FF
						if _, wErr := w.Write([]byte{0xFF}); wErr != nil {
							return rep, wErr
						}
					}

					if b2 == 0x00 || (b2 >= 0xD0 && b2 <= 0xD7) {
						if b2 == 0x00 {
							hasEntropy = true
						}
						if _, wErr := w.Write([]byte{0xFF, b2}); wErr != nil {
							return rep, wErr
						}
						continue
					}

					// We found a new marker that breaks entropy
					marker = b2
					goto processMarker
				}
			}
		}
	}
}

func stripPNG(ctx context.Context, r io.Reader, w io.Writer) (Report, error) {
	rep := Report{Format: "png"}
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		return rep, err
	}
	if !bytes.Equal(sig, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return rep, errors.New("not a valid PNG")
	}
	if _, err := w.Write(sig); err != nil {
		return rep, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return rep, err
		}

		header := make([]byte, 8)
		if _, err := io.ReadFull(r, header); err != nil {
			if err == io.EOF {
				return rep, io.ErrUnexpectedEOF
			}
			return rep, err
		}

		length := binary.BigEndian.Uint32(header[0:4])
		chunkType := string(header[4:8])

		keep := false
		// Keep critical chunks, APNG chunks, and standard structural/color chunks
		switch chunkType {
		case "IHDR", "PLTE", "IDAT", "IEND", "tRNS", "cHRM", "gAMA", "sRGB", "bKGD", "pHYs", "sBIT", "acTL", "fcTL", "fdAT":
			keep = true
		}

		if !keep {
			rep.StrippedChunks = append(rep.StrippedChunks, chunkType)
			rep.BytesSaved += int64(length) + 12

			if _, err := io.CopyN(io.Discard, r, int64(length)+4); err != nil {
				return rep, err
			}
		} else {
			if _, err := w.Write(header); err != nil {
				return rep, err
			}
			if _, err := io.CopyN(w, r, int64(length)+4); err != nil {
				return rep, err
			}
		}

		if chunkType == "IEND" {
			if length != 0 {
				return rep, errors.New("invalid IEND length")
			}
			// Discard appended data, which may contain identifying metadata.
			discarded, err := io.Copy(io.Discard, r)
			rep.BytesSaved += discarded
			return rep, err
		}
	}
}

// contextReader checks cancellation between streaming reads without buffering pixels.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func stripWebP(ctx context.Context, r io.Reader, w io.Writer) (Report, error) {
	rep := Report{Format: "webp"}
	r = &contextReader{ctx: ctx, reader: r}
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return rep, err
	}
	if string(header[:4]) != "RIFF" || string(header[8:]) != "WEBP" {
		return rep, errors.New("not a valid WebP")
	}
	size := int64(binary.LittleEndian.Uint32(header[4:8]))
	if size < 4 || size > (1<<32)-10 || size%2 != 0 {
		return rep, errors.New("invalid WebP RIFF size")
	}
	tmp, err := os.CreateTemp("", "webp-strip-*.webp")
	if err != nil {
		return rep, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := tmp.Write(header[:]); err != nil {
		return rep, err
	}
	chunks := &io.LimitedReader{R: r, N: size - 4}
	newSize := uint32(4)
	hasImage, hasExtended := false, false
	for chunks.N > 0 {
		if chunks.N < 8 {
			return rep, io.ErrUnexpectedEOF
		}
		var chunk [8]byte
		if _, err := io.ReadFull(chunks, chunk[:]); err != nil {
			return rep, err
		}
		kind := string(chunk[:4])
		length := int64(binary.LittleEndian.Uint32(chunk[4:]))
		padded := length + length%2
		if padded > chunks.N {
			return rep, errors.New("WebP chunk exceeds RIFF boundary")
		}
		keep := false
		switch kind {
		case "VP8X", "ICCP", "ANIM", "ANMF", "ALPH", "VP8 ", "VP8L":
			keep = true
		}
		if !keep {
			rep.StrippedChunks = append(rep.StrippedChunks, strings.TrimSpace(kind))
			rep.BytesSaved += 8 + padded
			if _, err := io.CopyN(io.Discard, chunks, padded); err != nil {
				return rep, err
			}
			continue
		}
		if kind == "VP8X" {
			if hasExtended || newSize != 4 || length != 10 {
				return rep, errors.New("invalid VP8X header")
			}
			hasExtended = true
			var data [10]byte
			if _, err := io.ReadFull(chunks, data[:]); err != nil {
				return rep, err
			}
			// Remove EXIF and XMP flags; retain alpha, animation and color profiles.
			data[0] &^= 0x0c
			if _, err := tmp.Write(chunk[:]); err != nil {
				return rep, err
			}
			if _, err := tmp.Write(data[:]); err != nil {
				return rep, err
			}
		} else {
			if (kind == "VP8 " || kind == "VP8L" || kind == "ANMF") && length > 0 {
				hasImage = true
			}
			if _, err := tmp.Write(chunk[:]); err != nil {
				return rep, err
			}
			if _, err := io.CopyN(tmp, chunks, length); err != nil {
				return rep, err
			}
			if padded != length {
				var pad [1]byte
				if _, err := io.ReadFull(chunks, pad[:]); err != nil {
					return rep, err
				}
				if pad[0] != 0 {
					return rep, errors.New("invalid WebP chunk padding")
				}
				if _, err := tmp.Write(pad[:]); err != nil {
					return rep, err
				}
			}
		}
		// The original RIFF boundary bounds this sum within uint32.
		newSize += uint32(8 + padded)
	}
	if !hasImage {
		return rep, errors.New("WebP has no image data")
	}
	// Never interpret appended data as additional chunks.
	discarded, err := io.Copy(io.Discard, r)
	rep.BytesSaved += discarded
	if err != nil {
		return rep, err
	}
	binary.LittleEndian.PutUint32(header[4:8], newSize)
	if _, err := tmp.WriteAt(header[4:8], 4); err != nil {
		return rep, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return rep, err
	}
	_, err = io.Copy(w, &contextReader{ctx: ctx, reader: tmp})
	return rep, err
}
