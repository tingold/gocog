package gocog

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/rand"
	"os"
	"sync"
	"testing"

	"github.com/paulmach/orb"
)

// Benchmark data generation helpers

// generateTestTileData creates synthetic tile data for benchmarking
func generateTestTileData(width, height, bands int, dataType DataType) []byte {
	bytesPerSample := getBytesPerSampleStatic(dataType)
	size := width * height * bands * bytesPerSample
	data := make([]byte, size)
	
	// Fill with random data
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	return data
}

// getBytesPerSampleStatic is a static version for benchmarks
func getBytesPerSampleStatic(dt DataType) int {
	switch dt {
	case DTByte, DTASCII, DTSByte, DTUndefined:
		return 1
	case DTSShort, DTSShortS:
		return 2
	case DTSLong, DTSLongS, DTFloat:
		return 4
	case DTRational, DTSRational, DTDouble:
		return 8
	default:
		return 1
	}
}

// createBenchmarkTIFF creates a minimal TIFF with specified dimensions for benchmarking
func createBenchmarkTIFF(width, height, bands int) []byte {
	var buf bytes.Buffer

	// TIFF header (8 bytes)
	binary.Write(&buf, binary.LittleEndian, uint16(0x4949)) // Little-endian magic
	binary.Write(&buf, binary.LittleEndian, uint16(42))     // Version
	binary.Write(&buf, binary.LittleEndian, uint32(8))      // First IFD offset

	// IFD with essential tags
	tagCount := uint16(8)
	binary.Write(&buf, binary.LittleEndian, tagCount)

	// Tag: ImageWidth (256)
	writeTag(&buf, 256, 4, 1, uint32(width))
	// Tag: ImageLength (257)
	writeTag(&buf, 257, 4, 1, uint32(height))
	// Tag: BitsPerSample (258)
	writeTag(&buf, 258, 3, 1, uint32(8))
	// Tag: Compression (259) = None
	writeTag(&buf, 259, 3, 1, uint32(1))
	// Tag: PhotometricInterpretation (262) = RGB
	writeTag(&buf, 262, 3, 1, uint32(2))
	// Tag: SamplesPerPixel (277)
	writeTag(&buf, 277, 3, 1, uint32(bands))
	// Tag: RowsPerStrip (278)
	writeTag(&buf, 278, 4, 1, uint32(height))
	// Tag: StripByteCounts (279)
	stripSize := uint32(width * height * bands)
	writeTag(&buf, 279, 4, 1, stripSize)

	// Next IFD offset (0 = none)
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	// Add dummy strip data
	stripData := make([]byte, stripSize)
	for i := range stripData {
		stripData[i] = byte(i % 256)
	}
	buf.Write(stripData)

	return buf.Bytes()
}

func writeTag(buf *bytes.Buffer, id, typ uint16, count, value uint32) {
	binary.Write(buf, binary.LittleEndian, id)
	binary.Write(buf, binary.LittleEndian, typ)
	binary.Write(buf, binary.LittleEndian, count)
	binary.Write(buf, binary.LittleEndian, value)
}

// =============================================================================
// Benchmarks for decodeBytesToFlat - memory layout performance
// =============================================================================

func BenchmarkDecodeBytesToFlat_Small(b *testing.B) {
	benchmarkDecodeBytesToFlat(b, 64, 64, 3, DTByte)
}

func BenchmarkDecodeBytesToFlat_Medium(b *testing.B) {
	benchmarkDecodeBytesToFlat(b, 256, 256, 3, DTByte)
}

func BenchmarkDecodeBytesToFlat_Large(b *testing.B) {
	benchmarkDecodeBytesToFlat(b, 512, 512, 3, DTByte)
}

func BenchmarkDecodeBytesToFlat_SingleBand(b *testing.B) {
	benchmarkDecodeBytesToFlat(b, 256, 256, 1, DTByte)
}

func BenchmarkDecodeBytesToFlat_FourBands(b *testing.B) {
	benchmarkDecodeBytesToFlat(b, 256, 256, 4, DTByte)
}

func BenchmarkDecodeBytesToFlat_16bit(b *testing.B) {
	benchmarkDecodeBytesToFlat(b, 256, 256, 3, DTSShort)
}

func BenchmarkDecodeBytesToFlat_32bit(b *testing.B) {
	benchmarkDecodeBytesToFlat(b, 256, 256, 3, DTSLong)
}

func benchmarkDecodeBytesToFlat(b *testing.B, width, height, bands int, dataType DataType) {
	data := generateTestTileData(width, height, bands, dataType)
	cog := &COG{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = cog.decodeBytesToFlat(data, width, height, bands, dataType, binary.LittleEndian, 2)
	}
}

// =============================================================================
// Benchmarks for pixel access patterns (flat array with accessor methods)
// =============================================================================

func BenchmarkPixelAccess_Sequential(b *testing.B) {
	width, height, bands := 256, 256, 3
	data := generateTestTileData(width, height, bands, DTByte)
	cog := &COG{}
	decodedData := cog.decodeBytesToFlat(data, width, height, bands, DTByte, binary.LittleEndian, 2)
	decoded := &RasterData{Data: decodedData, Width: width, Height: height, Bands: bands}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum = 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				for band := 0; band < bands; band++ {
					sum += decoded.AtUnchecked(band, x, y)
				}
			}
		}
	}
	_ = sum
}

func BenchmarkPixelAccess_SequentialDirect(b *testing.B) {
	width, height, bands := 256, 256, 3
	data := generateTestTileData(width, height, bands, DTByte)
	cog := &COG{}
	decoded := cog.decodeBytesToFlat(data, width, height, bands, DTByte, binary.LittleEndian, 2)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum = 0
		// Direct array access is fastest
		for _, v := range decoded {
			sum += v
		}
	}
	_ = sum
}

func BenchmarkPixelAccess_Random(b *testing.B) {
	width, height, bands := 256, 256, 3
	data := generateTestTileData(width, height, bands, DTByte)
	cog := &COG{}
	decodedData := cog.decodeBytesToFlat(data, width, height, bands, DTByte, binary.LittleEndian, 2)
	decoded := &RasterData{Data: decodedData, Width: width, Height: height, Bands: bands}
	
	// Pre-generate random coordinates
	coords := make([][3]int, 10000)
	for i := range coords {
		coords[i] = [3]int{rand.Intn(bands), rand.Intn(width), rand.Intn(height)}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum = 0
		for _, c := range coords {
			sum += decoded.AtUnchecked(c[0], c[1], c[2])
		}
	}
	_ = sum
}

// =============================================================================
// Benchmarks for decompression
// =============================================================================

func BenchmarkDecompressTile_None(b *testing.B) {
	width, height, bands := 256, 256, 3
	data := generateTestTileData(width, height, bands, DTByte)
	cog := &COG{}
	ifd := &IFD{Tags: make(map[uint16]*Tag), ByteOrder: binary.LittleEndian}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = cog.decompressTile(data, CompressionNone, ifd, width, height, bands, DTByte)
	}
}

// =============================================================================
// Benchmarks for full COG operations (requires test files)
// =============================================================================

func BenchmarkCOG_Open_LocalFile(b *testing.B) {
	// Skip if test file doesn't exist
	if _, err := os.Stat("TCI.tif"); os.IsNotExist(err) {
		b.Skip("TCI.tif not found, skipping benchmark")
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		cog, err := Open("TCI.tif", nil)
		if err != nil {
			b.Fatalf("Failed to open COG: %v", err)
		}
		_ = cog
	}
}

func BenchmarkCOG_ReadRegion_Small(b *testing.B) {
	if _, err := os.Stat("TCI.tif"); os.IsNotExist(err) {
		b.Skip("TCI.tif not found, skipping benchmark")
	}
	
	file, err := os.Open("TCI.tif")
	if err != nil {
		b.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()
	
	cog, err := Read(file)
	if err != nil {
		b.Fatalf("Failed to read COG: %v", err)
	}
	
	bounds := cog.Bounds()
	// Read 5% of the image
	region := orb.Bound{
		Min: orb.Point{bounds.Min[0], bounds.Min[1]},
		Max: orb.Point{
			bounds.Min[0] + (bounds.Max[0]-bounds.Min[0])*0.05,
			bounds.Min[1] + (bounds.Max[1]-bounds.Min[1])*0.05,
		},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		file.Seek(0, io.SeekStart)
		_, err := cog.ReadRegion(region, 0)
		if err != nil {
			b.Fatalf("ReadRegion failed: %v", err)
		}
	}
}

func BenchmarkCOG_ReadWindow_Small(b *testing.B) {
	if _, err := os.Stat("TCI.tif"); os.IsNotExist(err) {
		b.Skip("TCI.tif not found, skipping benchmark")
	}
	
	file, err := os.Open("TCI.tif")
	if err != nil {
		b.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()
	
	cog, err := Read(file)
	if err != nil {
		b.Fatalf("Failed to read COG: %v", err)
	}
	
	rect := Rectangle{X: 0, Y: 0, Width: 256, Height: 256}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := cog.ReadWindow(rect)
		if err != nil {
			b.Fatalf("ReadWindow failed: %v", err)
		}
	}
}

func BenchmarkCOG_ReadWindow_Large(b *testing.B) {
	if _, err := os.Stat("TCI.tif"); os.IsNotExist(err) {
		b.Skip("TCI.tif not found, skipping benchmark")
	}
	
	file, err := os.Open("TCI.tif")
	if err != nil {
		b.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()
	
	cog, err := Read(file)
	if err != nil {
		b.Fatalf("Failed to read COG: %v", err)
	}
	
	rect := Rectangle{X: 0, Y: 0, Width: 1024, Height: 1024}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := cog.ReadWindow(rect)
		if err != nil {
			b.Fatalf("ReadWindow failed: %v", err)
		}
	}
}

// =============================================================================
// Benchmarks for memory allocation patterns
// =============================================================================

func BenchmarkAllocation_NestedSlice(b *testing.B) {
	width, height, bands := 256, 256, 3
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		result := make([][][]uint64, bands)
		for band := 0; band < bands; band++ {
			result[band] = make([][]uint64, width)
			for x := 0; x < width; x++ {
				result[band][x] = make([]uint64, height)
			}
		}
		_ = result
	}
}

func BenchmarkAllocation_FlatSlice(b *testing.B) {
	width, height, bands := 256, 256, 3
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		result := make([]uint64, width*height*bands)
		_ = result
	}
}

// =============================================================================
// Benchmarks for byte buffer operations
// =============================================================================

func BenchmarkByteBufferAlloc(b *testing.B) {
	size := 256 * 256 * 3 // Typical tile size
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		buf := make([]byte, size)
		_ = buf
	}
}

func BenchmarkByteBufferPooled(b *testing.B) {
	size := 256 * 256 * 3 // Typical tile size
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		buf := GetBuffer(size)
		_ = buf
		PutBuffer(buf)
	}
}

func BenchmarkBytesBufferAlloc(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.Grow(256 * 256 * 3)
		_ = buf
	}
}

func BenchmarkBytesBufferPooled(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		buf := GetBytesBuffer()
		buf.Grow(256 * 256 * 3)
		PutBytesBuffer(buf)
	}
}

// =============================================================================
// Benchmarks for resampleImage
// =============================================================================

func BenchmarkResampleImage_256to128(b *testing.B) {
	srcWidth, srcHeight := 256, 256
	dstWidth, dstHeight := 128, 128
	bands := 3
	data := generateTestTileData(srcWidth, srcHeight, bands, DTByte)
	cog := &COG{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = cog.resampleImage(data, srcWidth, srcHeight, dstWidth, dstHeight, bands, DTByte)
	}
}

func BenchmarkResampleImage_512to256(b *testing.B) {
	srcWidth, srcHeight := 512, 512
	dstWidth, dstHeight := 256, 256
	bands := 3
	data := generateTestTileData(srcWidth, srcHeight, bands, DTByte)
	cog := &COG{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = cog.resampleImage(data, srcWidth, srcHeight, dstWidth, dstHeight, bands, DTByte)
	}
}

func BenchmarkResampleImage_1024to256(b *testing.B) {
	srcWidth, srcHeight := 1024, 1024
	dstWidth, dstHeight := 256, 256
	bands := 3
	data := generateTestTileData(srcWidth, srcHeight, bands, DTByte)
	cog := &COG{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = cog.resampleImage(data, srcWidth, srcHeight, dstWidth, dstHeight, bands, DTByte)
	}
}

// =============================================================================
// Benchmark for TIFF parsing
// =============================================================================

func BenchmarkTIFFReader_Parse(b *testing.B) {
	data := createBenchmarkTIFF(1024, 1024, 3)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, err := NewTIFFReader(reader)
		if err != nil {
			b.Fatalf("Failed to create TIFF reader: %v", err)
		}
	}
}

// =============================================================================
// Parallel vs Sequential comparison benchmarks
// =============================================================================

func BenchmarkSequentialTileProcessing(b *testing.B) {
	// Simulate processing multiple tiles sequentially
	tileCount := 16
	tileSize := 256 * 256 * 3
	tiles := make([][]byte, tileCount)
	for i := range tiles {
		tiles[i] = generateTestTileData(256, 256, 3, DTByte)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, tile := range tiles {
			// Simulate tile processing
			sum := uint64(0)
			for _, v := range tile {
				sum += uint64(v)
			}
			_ = sum
		}
		_ = tileSize
	}
}

func BenchmarkParallelTileProcessing(b *testing.B) {
	// Simulate processing multiple tiles in parallel
	tileCount := 16
	tiles := make([][]byte, tileCount)
	for i := range tiles {
		tiles[i] = generateTestTileData(256, 256, 3, DTByte)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		results := make([]uint64, tileCount)
		
		for idx, tile := range tiles {
			wg.Add(1)
			go func(idx int, tile []byte) {
				defer wg.Done()
				sum := uint64(0)
				for _, v := range tile {
					sum += uint64(v)
				}
				results[idx] = sum
			}(idx, tile)
		}
		wg.Wait()
		_ = results
	}
}

// =============================================================================
// Benchmarks comparing COG operations with local file
// =============================================================================

func BenchmarkCOG_ReadRegion_LocalFile(b *testing.B) {
	if _, err := os.Stat("B12.tif"); os.IsNotExist(err) {
		b.Skip("B12.tif not found, skipping benchmark")
	}
	
	file, err := os.Open("B12.tif")
	if err != nil {
		b.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()
	
	cog, err := Read(file)
	if err != nil {
		b.Fatalf("Failed to read COG: %v", err)
	}
	
	bounds := cog.Bounds()
	// Read 10% of the image
	region := orb.Bound{
		Min: orb.Point{bounds.Min[0], bounds.Min[1]},
		Max: orb.Point{
			bounds.Min[0] + (bounds.Max[0]-bounds.Min[0])*0.1,
			bounds.Min[1] + (bounds.Max[1]-bounds.Min[1])*0.1,
		},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		file.Seek(0, io.SeekStart)
		_, err := cog.ReadRegion(region, 0)
		if err != nil {
			b.Fatalf("ReadRegion failed: %v", err)
		}
	}
}

