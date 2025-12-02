# gocog

A pure Go library for reading Cloud Optimized GeoTIFF (COG) files without any C/C++/Rust dependencies. This library provides efficient access to geospatial raster data and exposes geometry information using types from [github.com/paulmach/orb](https://github.com/paulmach/orb).

## Features

- **Pure Go Implementation** - No external C/C++/Rust dependencies
- **Cloud Optimized GeoTIFF Support** - Efficient reading of COG files from cloud storage
- **HTTP Range Request Support** - Optimized for reading specific regions without downloading entire files
- **Automatic URL/File Detection** - The `Open()` function automatically detects URLs vs file paths
- **Geospatial Metadata** - Extracts CRS, georeferencing, and projection information
- **Geometry Integration** - Uses [orb](https://github.com/paulmach/orb) types for geometry representation
- **Overview Support** - Access to multiple resolution levels (pyramids) with automatic selection
- **Tiled and Stripped Images** - Efficient access to both tiled and stripped TIFF formats
- **Compression Support** - Supports multiple compression formats: None, LZW, Deflate/ZIP, and JPEG
- **Multiple Data Types** - Supports various pixel data types (8/16/32-bit integers, floats, signed/unsigned)
- **Map Tile Support** - Read standard map tiles (XYZ tiles) from COG files with automatic resampling
- **Pixel Space Windows** - Read rectangular regions in pixel coordinates with automatic overview selection
- **Optimized Metadata Reading** - Efficient metadata extraction using single-buffer reads and lazy loading

## Installation

```bash
go get github.com/tingold/gocog
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/tingold/gocog"
    "github.com/paulmach/orb"
    "github.com/paulmach/orb/maptile"
)

func main() {
    // Open a COG file (automatically detects URLs vs file paths)
    cog, err := gocog.Open("example.cog.tif", nil)
    if err != nil {
        panic(err)
    }
    
    // Get the bounding box as an orb.Bound
    bounds := cog.Bounds()
    fmt.Printf("Bounds: %+v\n", bounds)
    
    // Get the CRS
    crs := cog.CRS()
    fmt.Printf("CRS: %s\n", crs)
    
    // Read a specific region
    region := orb.Bound{
        Min: orb.Point{bounds.Min[0], bounds.Min[1]},
        Max: orb.Point{bounds.Min[0] + 0.1, bounds.Min[1] + 0.1},
    }
    
    data, err := cog.ReadRegion(region, 0) // 0 = highest resolution
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Read region: %dx%d pixels\n", data.Width, data.Height)
    
    // Read a map tile (XYZ tile format)
    // Create a tile at zoom level 10 using the center point of the bounds
    centerPoint := orb.Point{
        (bounds.Min[0] + bounds.Max[0]) / 2,
        (bounds.Min[1] + bounds.Max[1]) / 2,
    }
    zoom := maptile.Zoom(10)
    tile := maptile.At(centerPoint, zoom)
    
    tileData, err := cog.ReadTile(tile) // Defaults to 256x256 pixels
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Read tile: %dx%d pixels, %d bands\n", 
        tileData.Width, tileData.Height, tileData.Bands)
    
    // Read a tile with custom size (512x512)
    customTileData, err := cog.ReadTile(tile, 512)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Read custom size tile: %dx%d pixels\n", 
        customTileData.Width, customTileData.Height)
}
```

## API Overview

### Reading COG Files

- `Open(pathOrURL string, client *fasthttp.Client) (*COG, error)` - Open a COG from a file path or URL (automatically detects which). Validates the file can be opened and reads metadata only.
- `Read(io.ReadSeeker) (*COG, error)` - Read a COG from an io.ReadSeeker (file or HTTP range reader)
- `ReadFromURL(url string, client *fasthttp.Client) (*COG, error)` - Read a COG from a URL using fasthttp

### Accessing Metadata

- `Bounds() orb.Bound` - Get the geographic bounding box
- `CRS() string` - Get the Coordinate Reference System (e.g., "EPSG:4326")
- `Width() int` - Get image width in pixels
- `Height() int` - Get image height in pixels
- `BandCount() int` - Get number of bands
- `DataType() DataType` - Get pixel data type
- `OverviewCount() int` - Get the number of overview levels available
- `GetOverview(level int) *GeoTIFFMetadata` - Get metadata for a specific overview level (0 = highest resolution overview)

### Reading Data

- `ReadRegion(bound orb.Bound, overview int) (*RasterData, error)` - Read a geographic region from the specified overview level (0 = main image)
- `ReadWindow(rect Rectangle) (*RasterData, error)` - Read a window (rectangle) in pixel space. Automatically selects the appropriate overview level to minimize data transfer while maintaining reasonable resolution.
- `ReadTile(tile maptile.Tile, tileSize ...int) (*RasterData, error)` - Read a map tile from the COG. Supports EPSG:4326 and EPSG:3857 CRS. The tile will be resampled to the specified size (defaults to 256x256 if not provided).

### Types

- `Rectangle` - Represents a rectangle in pixel space with fields: `X`, `Y`, `Width`, `Height`
- `RasterData` - Contains raster data with fields: `Data [][][]uint64` (3D slice: [band][x][y]), `Width int`, `Height int`, `Bands int`, `Bounds orb.Bound`
- `DataType` - Represents pixel data types: `DTByte`, `DTSByte`, `DTSShort`, `DTSShortS`, `DTSLong`, `DTSLongS`, `DTFloat`, `DTDouble`, `DTRational`, `DTSRational`, `DTASCII`, `DTUndefined`

### Compression Support

The library supports reading COG files with the following compression formats:
- **None** (uncompressed)
- **LZW** (TIFF compression)
- **Deflate/ZIP** (ZIP compression)
- **JPEG** (JPEG compression for tiles/strips)

Compression is automatically detected and handled transparently when reading pixel data.

## License

MIT

