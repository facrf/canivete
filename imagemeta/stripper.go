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
					return rep, nil
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
					continue
				}
				b = b2
				break
			} else {
				if _, err := w.Write([]byte{b}); err != nil {
					return rep, err
				}
			}
		}

		marker := b

	processMarker:
		if (marker >= 0xD0 && marker <= 0xD9) || marker == 0x01 {
			if _, err := w.Write([]byte{0xFF, marker}); err != nil {
				return rep, err
			}
			if marker == 0xD9 {
				// EOI reached. We intentionally DROP any trailing data
				// to prevent appended metadata chunks.
				skipped, _ := io.Copy(io.Discard, br)
				rep.BytesSaved += skipped
				return rep, nil
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
		} else {
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
				chunk, err := br.ReadSlice(0xFF)

				if err == bufio.ErrBufferFull {
					// 0xFF not found in this buffer segment
					if _, wErr := w.Write(chunk); wErr != nil {
						return rep, wErr
					}
					continue
				} else if err != nil && err != io.EOF {
					return rep, err
				}

				hasFF := len(chunk) > 0 && chunk[len(chunk)-1] == 0xFF

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
						return rep, nil
					}
				}

				if hasFF {
					var b2 byte
					for {
						b2, err = br.ReadByte()
						if err != nil {
							if err == io.EOF {
								_, wErr := w.Write([]byte{0xFF})
								return rep, wErr
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
				break
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
			io.Copy(w, r)
			break
		}
	}
	return rep, nil
}

func stripWebP(ctx context.Context, r io.Reader, w io.Writer) (Report, error) {
	rep := Report{Format: "webp"}

	tmpFile, err := os.CreateTemp("", "webp-strip-*.webp")
	if err != nil {
		return rep, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	riffHeader := make([]byte, 12)
	if _, err := io.ReadFull(r, riffHeader); err != nil {
		return rep, err
	}
	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WEBP" {
		return rep, errors.New("not a valid WebP")
	}

	if _, err := tmpFile.Write(riffHeader); err != nil {
		return rep, err
	}

	newRiffSize := uint32(4)

	for {
		if err := ctx.Err(); err != nil {
			return rep, err
		}

		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(r, chunkHeader); err != nil {
			if err == io.EOF {
				break
			}
			return rep, err
		}

		chunkType := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		padding := uint32(0)
		if chunkSize%2 != 0 {
			padding = 1
		}

		strip := false
		switch chunkType {
		case "EXIF", "XMP ", "C2PA":
			strip = true
		}

		if strip {
			rep.StrippedChunks = append(rep.StrippedChunks, strings.TrimSpace(chunkType))
			rep.BytesSaved += int64(8 + chunkSize + padding)

			if _, err := io.CopyN(io.Discard, r, int64(chunkSize+padding)); err != nil {
				return rep, err
			}
		} else {
			if _, err := tmpFile.Write(chunkHeader); err != nil {
				return rep, err
			}
			if _, err := io.CopyN(tmpFile, r, int64(chunkSize+padding)); err != nil {
				return rep, err
			}
			newRiffSize += 8 + chunkSize + padding
		}
	}

	if _, err := tmpFile.Seek(4, io.SeekStart); err != nil {
		return rep, err
	}
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, newRiffSize)
	if _, err := tmpFile.Write(sizeBuf); err != nil {
		return rep, err
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return rep, err
	}
	if _, err := io.Copy(w, tmpFile); err != nil {
		return rep, err
	}

	return rep, nil
}
