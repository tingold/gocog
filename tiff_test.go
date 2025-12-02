package gocog

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

// createSimpleTIFF creates a minimal valid TIFF file for testing
func createSimpleTIFF() []byte {
	var buf bytes.Buffer

	// TIFF header
	binary.Write(&buf, binary.LittleEndian, uint16(0x4949)) // Little-endian magic
	binary.Write(&buf, binary.LittleEndian, uint16(42))     // Version
	binary.Write(&buf, binary.LittleEndian, uint32(8))      // First IFD offset

	// IFD: 1 tag
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // Tag count

	// Tag: ImageWidth (256) = 100
	binary.Write(&buf, binary.LittleEndian, uint16(256)) // Tag ID
	binary.Write(&buf, binary.LittleEndian, uint16(4))   // Type: LONG
	binary.Write(&buf, binary.LittleEndian, uint32(1))   // Count
	binary.Write(&buf, binary.LittleEndian, uint32(100)) // Value (inline)

	// Next IFD offset (0 = no more IFDs)
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	return buf.Bytes()
}

func TestTIFFReader(t *testing.T) {
	data := createSimpleTIFF()
	reader := bytes.NewReader(data)

	tr, err := NewTIFFReader(reader)
	if err != nil {
		t.Fatalf("Failed to create TIFF reader: %v", err)
	}

	if tr.IFDCount() != 1 {
		t.Errorf("Expected 1 IFD, got %d", tr.IFDCount())
	}

	ifd := tr.GetIFD(0)
	if ifd == nil {
		t.Fatal("IFD 0 is nil")
	}

	// Check ImageWidth tag
	tag := ifd.Tags[256]
	if tag == nil {
		t.Fatal("ImageWidth tag not found")
	}

	width, ok := tag.Value.(uint32)
	if !ok {
		t.Fatalf("Expected uint32, got %T", tag.Value)
	}

	if width != 100 {
		t.Errorf("Expected width 100, got %d", width)
	}
}

func TestTIFFReaderBigEndian(t *testing.T) {
	var buf bytes.Buffer

	// Big-endian TIFF header
	binary.Write(&buf, binary.BigEndian, uint16(0x4D4D)) // Big-endian magic
	binary.Write(&buf, binary.BigEndian, uint16(42))     // Version
	binary.Write(&buf, binary.BigEndian, uint32(8))      // First IFD offset

	// IFD: 1 tag
	binary.Write(&buf, binary.BigEndian, uint16(1)) // Tag count

	// Tag: ImageWidth (256) = 200
	binary.Write(&buf, binary.BigEndian, uint16(256)) // Tag ID
	binary.Write(&buf, binary.BigEndian, uint16(4))   // Type: LONG
	binary.Write(&buf, binary.BigEndian, uint32(1))   // Count
	binary.Write(&buf, binary.BigEndian, uint32(200)) // Value (inline)

	// Next IFD offset
	binary.Write(&buf, binary.BigEndian, uint32(0))

	reader := bytes.NewReader(buf.Bytes())
	tr, err := NewTIFFReader(reader)
	if err != nil {
		t.Fatalf("Failed to create TIFF reader: %v", err)
	}

	ifd := tr.GetIFD(0)
	if ifd == nil {
		t.Fatal("IFD 0 is nil")
	}

	if ifd.ByteOrder != binary.BigEndian {
		t.Error("Expected big-endian byte order")
	}

	tag := ifd.Tags[256]
	if tag == nil {
		t.Fatal("ImageWidth tag not found")
	}

	width := tag.Value.(uint32)
	if width != 200 {
		t.Errorf("Expected width 200, got %d", width)
	}
}

func TestHTTPRangeReader(t *testing.T) {
	// This test would require a mock HTTP server
	// For now, we'll just test the structure with a real HTTP client
	// Note: This will fail if there's no network, but that's acceptable for a basic test
	client := &fasthttp.Client{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	rr := NewHTTPRangeReader("https://data.source.coop/earthgenome/earthindeximagery/18SUJ_2024-01-01_2025-01-01/TCI.tif", client)
	if rr == nil {
		t.Fatal("Failed to create HTTPRangeReader")
	}

	// Test Seek
	pos, err := rr.Seek(0, io.SeekStart)
	if err != nil {
		t.Errorf("Seek failed: %v", err)
	}
	if pos != 0 {
		t.Errorf("Expected position 0, got %d", pos)
	}
}
