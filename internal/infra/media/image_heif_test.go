package media

import (
	"encoding/base64"
	"encoding/binary"
	"testing"
)

func TestDecodeBase64ImageDataSupportsHEICDimensions(t *testing.T) {
	data := buildHEIFTestFile("heic", 1536, 1024, false)

	config, format, _, err := DecodeBase64ImageData(base64.StdEncoding.EncodeToString(data))
	if err != nil {
		t.Fatalf("DecodeBase64ImageData error: %v", err)
	}
	if format != "heic" {
		t.Fatalf("format = %q, want heic", format)
	}
	if config.Width != 1536 || config.Height != 1024 {
		t.Fatalf("config = %dx%d, want 1536x1024", config.Width, config.Height)
	}
}

func TestDecodeImageConfigSupportsHEIFExtendedMetaBox(t *testing.T) {
	data := buildHEIFTestFile("mif1", 832, 1248, true)

	config, format, err := decodeImageConfig(data)
	if err != nil {
		t.Fatalf("decodeImageConfig error: %v", err)
	}
	if format != "heif" {
		t.Fatalf("format = %q, want heif", format)
	}
	if config.Width != 832 || config.Height != 1248 {
		t.Fatalf("config = %dx%d, want 832x1248", config.Width, config.Height)
	}
}

func TestGetMimeTypeByExtensionSupportsHEICAndHEIF(t *testing.T) {
	if got := GetMimeTypeByExtension("heic"); got != "image/heic" {
		t.Fatalf("heic mime = %q, want image/heic", got)
	}
	if got := GetMimeTypeByExtension("HEIF"); got != "image/heif" {
		t.Fatalf("HEIF mime = %q, want image/heif", got)
	}
}

func buildHEIFTestFile(brand string, width int, height int, extendedMeta bool) []byte {
	ftypPayload := append([]byte(brand), 0, 0, 0, 0)
	ftypPayload = append(ftypPayload, []byte(brand)...)

	ispePayload := make([]byte, 12)
	binary.BigEndian.PutUint32(ispePayload[4:8], uint32(width))
	binary.BigEndian.PutUint32(ispePayload[8:12], uint32(height))
	ispe := makeBox("ispe", ispePayload)
	ipco := makeBox("ipco", ispe)
	iprp := makeBox("iprp", ipco)
	metaPayload := append([]byte{0, 0, 0, 0}, iprp...)

	meta := makeBox("meta", metaPayload)
	if extendedMeta {
		meta = makeExtendedBox("meta", metaPayload)
	}

	return append(makeBox("ftyp", ftypPayload), meta...)
}

func makeBox(boxType string, payload []byte) []byte {
	box := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(box)))
	copy(box[4:8], boxType)
	copy(box[8:], payload)
	return box
}

func makeExtendedBox(boxType string, payload []byte) []byte {
	box := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint32(box[0:4], 1)
	copy(box[4:8], boxType)
	binary.BigEndian.PutUint64(box[8:16], uint64(len(box)))
	copy(box[16:], payload)
	return box
}
