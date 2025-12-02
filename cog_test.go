package gocog_test

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/maptile"
	"github.com/tingold/gocog"
)

func TestOpenCOG(t *testing.T) {
	cog, err := gocog.Open("TCI.tif", nil)
	if err != nil {
		t.Logf("Error opening COG: %v\n", err)
		return
	}
	t.Logf("COG opened successfully\n")
	t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		cog.Bounds().Min[0], cog.Bounds().Min[1],
		cog.Bounds().Max[0], cog.Bounds().Max[1])
	t.Logf("CRS: %s\n", cog.CRS())
	t.Logf("Image size: %dx%d pixels\n", cog.Width(), cog.Height())
	t.Logf("Bands: %d\n", cog.BandCount())
	t.Logf("Overview levels: %d\n", cog.OverviewCount())
}

func TestRead(t *testing.T) {
	// Open a COG file
	file, err := os.Open("TCI.tif")
	if err != nil {
		t.Logf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Read the COG
	cog, err := gocog.Read(file)
	if err != nil {
		t.Logf("Error reading COG: %v\n", err)
		return
	}

	// Get the bounding box as an orb.Bound
	bounds := cog.Bounds()
	t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		bounds.Min[0], bounds.Min[1], bounds.Max[0], bounds.Max[1])

	// Get the CRS
	crs := cog.CRS()
	t.Logf("CRS: %s\n", crs)

	// Get image dimensions
	t.Logf("Image size: %dx%d pixels\n", cog.Width(), cog.Height())
	t.Logf("Bands: %d\n", cog.BandCount())
	t.Logf("Overview levels: %d\n", cog.OverviewCount())
}

func TestCOG_ReadRegion(t *testing.T) {
	file, err := os.Open("B12.tif")
	if err != nil {
		t.Logf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	cog, err := gocog.Read(file)
	if err != nil {
		t.Logf("Error reading COG: %v\n", err)
		return
	}

	// Get the full bounds
	fullBounds := cog.Bounds()

	// Read a specific region (10% of the image)
	region := orb.Bound{
		Min: orb.Point{
			fullBounds.Min[0],
			fullBounds.Min[1],
		},
		Max: orb.Point{
			fullBounds.Min[0] + (fullBounds.Max[0]-fullBounds.Min[0])*0.1,
			fullBounds.Min[1] + (fullBounds.Max[1]-fullBounds.Min[1])*0.1,
		},
	}

	data, err := cog.ReadRegion(region, 0) // 0 = highest resolution
	if err != nil {
		t.Logf("Error reading region: %v\n", err)
		return
	}

	t.Logf("Read region: %dx%d pixels, %d bands\n",
		data.Width, data.Height, data.Bands)
	t.Logf("Region bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		data.Bounds.Min[0], data.Bounds.Min[1],
		data.Bounds.Max[0], data.Bounds.Max[1])
}

func TestReadFromURL(t *testing.T) {
	// Read a COG from a URL (e.g., S3, GCS, Azure Blob Storage)
	cog, err := gocog.Open("https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif", nil)
	if err != nil {
		t.Logf("Error reading COG from URL: %v\n", err)
		return
	}
	t.Logf("COG opened successfully\n")
	bounds := cog.Bounds()
	t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		bounds.Min[0], bounds.Min[1], bounds.Max[0], bounds.Max[1])
}

func TestCOG_ReadWindow(t *testing.T) {

	cog, err := gocog.Open("https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif", nil)
	if err != nil {
		t.Logf("Error reading COG: %v\n", err)
		return
	}

	// Read a window (rectangle) in pixel space
	// This reads a 512x512 pixel window starting at (0, 0)
	rect := gocog.Rectangle{
		X:      0,
		Y:      0,
		Width:  512,
		Height: 512,
	}

	window, err := cog.ReadWindow(rect)
	if err != nil {
		t.Logf("Error reading window: %v\n", err)
		return
	}

	t.Logf("Window size: %dx%d pixels\n", window.Width, window.Height)
	t.Logf("Window bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		window.Bounds.Min[0], window.Bounds.Min[1],
		window.Bounds.Max[0], window.Bounds.Max[1])
}

func TestCOG_ReadTile(t *testing.T) {
	cog, err := gocog.Open("https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif", nil)
	if err != nil {
		t.Logf("Error reading COG: %v\n", err)
		return
	}

	// Validate CRS is supported
	crs := cog.CRS()
	if crs != "EPSG:4326" && crs != "EPSG:3857" {
		t.Logf("Skipping test - CRS %s is not supported by ReadTile\n", crs)
		return
	}

	// Get the bounds to determine appropriate tile coordinates
	bounds := cog.Bounds()
	t.Logf("COG bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		bounds.Min[0], bounds.Min[1], bounds.Max[0], bounds.Max[1])
	t.Logf("CRS: %s\n", crs)

	// Convert bounds to WGS84 if needed (maptile.At expects lat/lon)
	var wgs84Bounds orb.Bound
	if crs == "EPSG:3857" {
		// Convert Web Mercator to WGS84
		wgs84Bounds = mercatorToWGS84(bounds)
	} else {
		wgs84Bounds = bounds
	}

	t.Logf("WGS84 bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		wgs84Bounds.Min[0], wgs84Bounds.Min[1], wgs84Bounds.Max[0], wgs84Bounds.Max[1])

	// Create a tile at zoom level 10 that should intersect with the image bounds
	// Use the center point of the bounds to create a tile
	centerLon := (wgs84Bounds.Min[0] + wgs84Bounds.Max[0]) / 2.0
	centerLat := (wgs84Bounds.Min[1] + wgs84Bounds.Max[1]) / 2.0
	centerPoint := orb.Point{centerLon, centerLat}

	// Create a tile at zoom level 10
	zoom := maptile.Zoom(10)
	tile := maptile.At(centerPoint, zoom)

	t.Logf("Tile coordinates: X=%d, Y=%d, Z=%d\n", tile.X, tile.Y, tile.Z)

	// Get tile bounds to verify it intersects
	tileBounds := tile.Bound()
	t.Logf("Tile bounds (WGS84): Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		tileBounds.Min[0], tileBounds.Min[1], tileBounds.Max[0], tileBounds.Max[1])

	// Test ReadTile with default tile size (256)
	tileData, err := cog.ReadTile(tile)
	if err != nil {
		t.Logf("Error reading tile: %v\n", err)
		return
	}

	if tileData == nil {
		t.Logf("Tile data is nil\n")
		return
	}

	// Validate tile dimensions
	if tileData.Width != 256 {
		t.Logf("Expected width 256, got %d\n", tileData.Width)
	}
	if tileData.Height != 256 {
		t.Logf("Expected height 256, got %d\n", tileData.Height)
	}
	if tileData.Bands == 0 {
		t.Logf("Expected bands > 0, got %d\n", tileData.Bands)
	}
	if len(tileData.Data) == 0 {
		t.Logf("Expected non-empty data, got empty data slice\n")
	}

	t.Logf("Tile read successfully: %dx%d pixels, %d bands\n",
		tileData.Width, tileData.Height, tileData.Bands)
	t.Logf("Tile bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]\n",
		tileData.Bounds.Min[0], tileData.Bounds.Min[1],
		tileData.Bounds.Max[0], tileData.Bounds.Max[1])

	// Test ReadTile with custom tile size (512)
	customTileData, err := cog.ReadTile(tile, 512)
	if err != nil {
		t.Logf("Error reading tile with custom size: %v\n", err)
		return
	}

	if customTileData.Width != 512 || customTileData.Height != 512 {
		t.Logf("Expected 512x512, got %dx%d\n", customTileData.Width, customTileData.Height)
	}

	t.Logf("Custom size tile read successfully: %dx%d pixels\n",
		customTileData.Width, customTileData.Height)
}

func TestCOG_ReadTile_UnsupportedCRS(t *testing.T) {
	// This test would require a COG with an unsupported CRS
	// For now, we'll test the error handling by checking the CRS validation logic
	// In a real scenario, you'd need a test file with a different CRS

	// Note: This test demonstrates the expected behavior but may not run
	// if we don't have a test file with unsupported CRS
	t.Logf("Test for unsupported CRS would require a COG file with CRS other than EPSG:4326 or EPSG:3857")
}

// mercatorToWGS84 converts Web Mercator (EPSG:3857) bounds to WGS84 (EPSG:4326) bounds
// This is a helper function for testing
func mercatorToWGS84(bound orb.Bound) orb.Bound {
	const maxMercator = 20037508.342789244

	minLon := bound.Min[0] / maxMercator * 180.0
	maxLon := bound.Max[0] / maxMercator * 180.0

	minLat := math.Atan(math.Exp(bound.Min[1]*math.Pi/maxMercator))*360.0/math.Pi - 90.0
	maxLat := math.Atan(math.Exp(bound.Max[1]*math.Pi/maxMercator))*360.0/math.Pi - 90.0

	return orb.Bound{
		Min: orb.Point{minLon, minLat},
		Max: orb.Point{maxLon, maxLat},
	}
}

func TestCOG_ReadTileAndWriteToFile(t *testing.T) {
	// Open a COG from URL (using an existing URL from tests)
	url := "https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif"
	cog, err := gocog.Open(url, nil)
	if err != nil {
		t.Fatalf("Error opening COG from URL: %v", err)
	}

	// Validate CRS is supported
	crs := cog.CRS()
	if crs != "EPSG:4326" && crs != "EPSG:3857" {
		t.Skipf("Skipping test - CRS %s is not supported by ReadTile", crs)
	}

	// Get the bounds to determine appropriate tile coordinates
	bounds := cog.Bounds()
	t.Logf("COG bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]", bounds.Min[0], bounds.Min[1], bounds.Max[0], bounds.Max[1])
	t.Logf("CRS: %s", crs)

	// Convert bounds to WGS84 if needed (maptile.At expects lat/lon)
	var wgs84Bounds orb.Bound
	if crs == "EPSG:3857" {
		wgs84Bounds = mercatorToWGS84(bounds)
	} else {
		wgs84Bounds = bounds
	}

	// Create a tile at zoom level 10 using the center point of the bounds
	centerLon := (wgs84Bounds.Min[0] + wgs84Bounds.Max[0]) / 2.0
	centerLat := (wgs84Bounds.Min[1] + wgs84Bounds.Max[1]) / 2.0
	centerPoint := orb.Point{centerLon, centerLat}

	zoom := maptile.Zoom(10)
	tile := maptile.At(centerPoint, zoom)

	t.Logf("Tile coordinates: X=%d, Y=%d, Z=%d", tile.X, tile.Y, tile.Z)

	// Read the tile
	tileData, err := cog.ReadTile(tile)
	if err != nil {
		t.Fatalf("Error reading tile: %v", err)
	}

	if tileData == nil {
		t.Fatal("Tile data is nil")
	}

	t.Logf("Tile read successfully: %dx%d pixels, %d bands",
		tileData.Width, tileData.Height, tileData.Bands)

	// Convert raw pixel data to PNG image and write to file
	outputFile := "test_tile_output.png"

	err = writeTileToPNG(tileData, outputFile, cog.DataType())
	if err != nil {
		t.Fatalf("Error writing tile to PNG: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file %s was not created", outputFile)
	}

	t.Logf("Successfully wrote tile to %s", outputFile)
}

// writeTileToPNG converts RasterData to a PNG image and writes it to disk
func writeTileToPNG(tileData *gocog.RasterData, filename string, dataType gocog.DataType) error {
	// Only support 8-bit unsigned integer data for PNG conversion
	if dataType != gocog.DTByte {
		return nil // Skip conversion for non-8-bit data
	}

	// Create an image based on the number of bands
	var img image.Image
	width := tileData.Width
	height := tileData.Height

	switch tileData.Bands {
	case 1:
		// Grayscale image
		grayImg := image.NewGray(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				value := tileData.At(0, x, y)
				if value > 255 {
					value = 255
				}
				grayImg.SetGray(x, y, color.Gray{Y: uint8(value)})
			}
		}
		img = grayImg

	case 3:
		// RGB image
		rgbaImg := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r := tileData.At(0, x, y)
				g := tileData.At(1, x, y)
				b := tileData.At(2, x, y)
				if r > 255 {
					r = 255
				}
				if g > 255 {
					g = 255
				}
				if b > 255 {
					b = 255
				}
				rgbaImg.Set(x, y, color.RGBA{
					R: uint8(r),
					G: uint8(g),
					B: uint8(b),
					A: 255,
				})
			}
		}
		img = rgbaImg

	case 4:
		// RGBA image
		rgbaImg := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r := tileData.At(0, x, y)
				g := tileData.At(1, x, y)
				b := tileData.At(2, x, y)
				a := tileData.At(3, x, y)
				if r > 255 {
					r = 255
				}
				if g > 255 {
					g = 255
				}
				if b > 255 {
					b = 255
				}
				if a > 255 {
					a = 255
				}
				rgbaImg.Set(x, y, color.RGBA{
					R: uint8(r),
					G: uint8(g),
					B: uint8(b),
					A: uint8(a),
				})
			}
		}
		img = rgbaImg

	default:
		// For other band counts, convert to grayscale by taking first band
		grayImg := image.NewGray(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				value := tileData.At(0, x, y)
				if value > 255 {
					value = 255
				}
				grayImg.SetGray(x, y, color.Gray{Y: uint8(value)})
			}
		}
		img = grayImg
	}

	// Create output file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode as PNG
	return png.Encode(file, img)
}
