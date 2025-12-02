package gocog_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/tingold/gocog"
)

// rasterioResult represents the JSON output from the Python benchmark script
type rasterioResult struct {
	DurationNs int64 `json:"duration_ns"`
	Error      string `json:"error,omitempty"`
	Bounds     struct {
		Min [2]float64 `json:"min"`
		Max [2]float64 `json:"max"`
	} `json:"bounds,omitempty"`
	CRS          string `json:"crs,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	BandCount    int    `json:"band_count,omitempty"`
	DataType     string `json:"data_type,omitempty"`
	OverviewCount int   `json:"overview_count,omitempty"`
}

// rasterioBenchmarkResult represents the JSON output from benchmark runs
type rasterioBenchmarkResult struct {
	Iterations int   `json:"iterations"`
	Successful  int   `json:"successful"`
	Failed      int   `json:"failed"`
	AvgNs       int64 `json:"avg_ns"`
	MedianNs    int64 `json:"median_ns"`
	MinNs       int64 `json:"min_ns"`
	MaxNs       int64 `json:"max_ns"`
	Error       string `json:"error,omitempty"`
}

// runRasterioBenchmark runs the Python benchmark script and returns the result
func runRasterioBenchmark(filePath string) (*rasterioResult, error) {
	scriptPath := filepath.Join(".", "benchmark_rasterio.py")
	
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("benchmark_rasterio.py not found")
	}
	
	// Try uv run first, fall back to python3
	var cmd *exec.Cmd
	if _, err := exec.LookPath("uv"); err == nil {
		cmd = exec.Command("uv", "run", scriptPath, "single", filePath)
	} else {
		cmd = exec.Command("python3", scriptPath, "single", filePath)
	}
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to run rasterio benchmark: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run rasterio benchmark: %w", err)
	}
	
	var result rasterioResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rasterio output: %w", err)
	}
	
	if result.Error != "" {
		return nil, fmt.Errorf("rasterio error: %s", result.Error)
	}
	
	return &result, nil
}

// runRasterioBenchmarkURL runs the Python benchmark script for a URL
func runRasterioBenchmarkURL(url string) (*rasterioResult, error) {
	scriptPath := filepath.Join(".", "benchmark_rasterio.py")
	
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("benchmark_rasterio.py not found")
	}
	
	// Try uv run first, fall back to python3
	var cmd *exec.Cmd
	if _, err := exec.LookPath("uv"); err == nil {
		cmd = exec.Command("uv", "run", scriptPath, "url", url)
	} else {
		cmd = exec.Command("python3", scriptPath, "url", url)
	}
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to run rasterio benchmark: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run rasterio benchmark: %w", err)
	}
	
	var result rasterioResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rasterio output: %w", err)
	}
	
	if result.Error != "" {
		return nil, fmt.Errorf("rasterio error: %s", result.Error)
	}
	
	return &result, nil
}

// TestCompareRasterioVsGoCOG compares the speed of opening a COG and reading info
// using gocog vs rasterio (Python library).
func TestCompareRasterioVsGoCOG(t *testing.T) {
	testFiles := []string{"TCI.tif", "B12.tif"}

	for _, testFile := range testFiles {
		t.Run(testFile, func(t *testing.T) {
			// Check if Python script is available
			if _, err := os.Stat("benchmark_rasterio.py"); os.IsNotExist(err) {
				t.Skip("benchmark_rasterio.py not found, skipping comparison test")
			}
			
			// Check if uv or python3 is available
			if _, err := exec.LookPath("uv"); err != nil {
				if _, err := exec.LookPath("python3"); err != nil {
					t.Skip("neither uv nor python3 found in PATH, skipping comparison test")
				}
			}

			t.Logf("=== Comparing performance for %s ===\n", testFile)

			// --- gocog timing ---
			gocogStart := time.Now()
			cog, err := gocog.Open(testFile, nil)
			gocogDuration := time.Since(gocogStart)

			if err != nil {
				t.Logf("gocog: Error opening file: %v", err)
				return
			}
			
			// Access all metadata
			bounds := cog.Bounds()
			crs := cog.CRS()
			width := cog.Width()
			height := cog.Height()
			bandCount := cog.BandCount()
			dataType := cog.DataType()
			overviewCount := cog.OverviewCount()
			
			t.Logf("\n--- gocog output ---")
			t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]",
				bounds.Min[0], bounds.Min[1],
				bounds.Max[0], bounds.Max[1])
			t.Logf("CRS: %s", crs)
			t.Logf("Image size: %dx%d pixels", width, height)
			t.Logf("Bands: %d", bandCount)
			t.Logf("Data type: %v", dataType)
			t.Logf("Overview levels: %d", overviewCount)

			// --- rasterio timing ---
			rasterioResult, rasterioErr := runRasterioBenchmark(testFile)
			var rasterioDuration time.Duration
			
			if rasterioErr != nil {
				t.Logf("rasterio: Error running benchmark: %v", rasterioErr)
			} else {
				rasterioDuration = time.Duration(rasterioResult.DurationNs) * time.Nanosecond
				t.Logf("\n--- rasterio output ---")
				t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]",
					rasterioResult.Bounds.Min[0], rasterioResult.Bounds.Min[1],
					rasterioResult.Bounds.Max[0], rasterioResult.Bounds.Max[1])
				t.Logf("CRS: %s", rasterioResult.CRS)
				t.Logf("Image size: %dx%d pixels", rasterioResult.Width, rasterioResult.Height)
				t.Logf("Bands: %d", rasterioResult.BandCount)
				t.Logf("Data type: %s", rasterioResult.DataType)
				t.Logf("Overview levels: %d", rasterioResult.OverviewCount)
			}

			// --- Results comparison ---
			t.Logf("\n=== TIMING RESULTS for %s ===", testFile)
			t.Logf("gocog:    %v", gocogDuration)
			if rasterioErr == nil {
				t.Logf("rasterio: %v", rasterioDuration)

				if rasterioDuration > 0 && gocogDuration > 0 {
					speedup := float64(rasterioDuration) / float64(gocogDuration)
					if speedup > 1 {
						t.Logf("gocog is %.2fx FASTER than rasterio", speedup)
					} else {
						t.Logf("rasterio is %.2fx faster than gocog", 1/speedup)
					}
				}
			}
		})
	}
}

// TestCompareRasterioVsGoCOG_URL compares performance for remote COGs (HTTP)
func TestCompareRasterioVsGoCOG_URL(t *testing.T) {
	testURL := "https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif"

	// Check if Python script is available
	if _, err := os.Stat("benchmark_rasterio.py"); os.IsNotExist(err) {
		t.Skip("benchmark_rasterio.py not found, skipping comparison test")
	}
	
	// Check if uv or python3 is available
	if _, err := exec.LookPath("uv"); err != nil {
		if _, err := exec.LookPath("python3"); err != nil {
			t.Skip("neither uv nor python3 found in PATH, skipping comparison test")
		}
	}

	t.Logf("=== Comparing performance for remote COG ===")
	t.Logf("URL: %s", testURL)

	// --- gocog timing ---
	gocogStart := time.Now()
	cog, err := gocog.Open(testURL, nil)
	gocogDuration := time.Since(gocogStart)

	if err != nil {
		t.Logf("gocog: Error opening URL: %v", err)
		return
	}
	
	bounds := cog.Bounds()
	crs := cog.CRS()
	width := cog.Width()
	height := cog.Height()
	bandCount := cog.BandCount()
	overviewCount := cog.OverviewCount()
	
	t.Logf("\n--- gocog output ---")
	t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]",
		bounds.Min[0], bounds.Min[1],
		bounds.Max[0], bounds.Max[1])
	t.Logf("CRS: %s", crs)
	t.Logf("Image size: %dx%d pixels", width, height)
	t.Logf("Bands: %d", bandCount)
	t.Logf("Overview levels: %d", overviewCount)

	// --- rasterio timing ---
	rasterioResult, rasterioErr := runRasterioBenchmarkURL(testURL)
	var rasterioDuration time.Duration
	
	if rasterioErr != nil {
		t.Logf("rasterio: Error running benchmark: %v", rasterioErr)
	} else {
		rasterioDuration = time.Duration(rasterioResult.DurationNs) * time.Nanosecond
		t.Logf("\n--- rasterio output ---")
		t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]",
			rasterioResult.Bounds.Min[0], rasterioResult.Bounds.Min[1],
			rasterioResult.Bounds.Max[0], rasterioResult.Bounds.Max[1])
		t.Logf("CRS: %s", rasterioResult.CRS)
		t.Logf("Image size: %dx%d pixels", rasterioResult.Width, rasterioResult.Height)
		t.Logf("Bands: %d", rasterioResult.BandCount)
		t.Logf("Overview levels: %d", rasterioResult.OverviewCount)
	}

	// --- Results comparison ---
	t.Logf("\n=== TIMING RESULTS for remote COG ===")
	t.Logf("gocog:    %v", gocogDuration)
	if rasterioErr == nil {
		t.Logf("rasterio: %v", rasterioDuration)

		if rasterioDuration > 0 && gocogDuration > 0 {
			speedup := float64(rasterioDuration) / float64(gocogDuration)
			if speedup > 1 {
				t.Logf("gocog is %.2fx FASTER than rasterio", speedup)
			} else {
				t.Logf("rasterio is %.2fx faster than gocog", 1/speedup)
			}
		}
	}
}

// TestPrintRasterioSummary prints a summary table comparing gocog vs rasterio
func TestPrintRasterioSummary(t *testing.T) {
	// Check if Python script is available
	if _, err := os.Stat("benchmark_rasterio.py"); os.IsNotExist(err) {
		t.Skip("benchmark_rasterio.py not found, skipping summary test")
	}
	
	// Check if uv or python3 is available
	if _, err := exec.LookPath("uv"); err != nil {
		if _, err := exec.LookPath("python3"); err != nil {
			t.Skip("neither uv nor python3 found in PATH, skipping summary test")
		}
	}

	testCases := []struct {
		name   string
		path   string
		isURL  bool
	}{
		{"TCI.tif (local)", "TCI.tif", false},
		{"B12.tif (local)", "B12.tif", false},
		{"TCI.tif (remote)", "https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif", true},
	}

	t.Logf("\n")
	t.Logf("╔════════════════════════════════════════════════════════════════════════╗")
	t.Logf("║                  GOCOG vs RASTERIO PERFORMANCE                         ║")
	t.Logf("╠═══════════════════════╦═══════════════╦═══════════════╦════════════════╣")
	t.Logf("║ File                  ║ gocog         ║ rasterio      ║ Speedup        ║")
	t.Logf("╠═══════════════════════╬═══════════════╬═══════════════╬════════════════╣")

	for _, tc := range testCases {
		// Time gocog
		gocogStart := time.Now()
		cog, err := gocog.Open(tc.path, nil)
		gocogDuration := time.Since(gocogStart)
		if err != nil {
			t.Logf("║ %-21s ║ ERROR         ║               ║                ║", tc.name)
			continue
		}
		_ = cog.Bounds()
		_ = cog.CRS()

		// Time rasterio
		var rasterioResult *rasterioResult
		var rasterioErr error
		if tc.isURL {
			rasterioResult, rasterioErr = runRasterioBenchmarkURL(tc.path)
		} else {
			rasterioResult, rasterioErr = runRasterioBenchmark(tc.path)
		}

		if rasterioErr != nil {
			t.Logf("║ %-21s ║ %-13v ║ ERROR         ║                ║", tc.name, gocogDuration.Round(time.Microsecond))
			continue
		}

		rasterioDuration := time.Duration(rasterioResult.DurationNs) * time.Nanosecond
		speedup := float64(rasterioDuration) / float64(gocogDuration)
		speedupStr := fmt.Sprintf("%.2fx faster", speedup)
		if speedup < 1 {
			speedupStr = fmt.Sprintf("%.2fx slower", 1/speedup)
		}

		t.Logf("║ %-21s ║ %-13v ║ %-13v ║ %-14s ║",
			tc.name,
			gocogDuration.Round(time.Microsecond),
			rasterioDuration.Round(time.Microsecond),
			speedupStr)
	}

	t.Logf("╚═══════════════════════╩═══════════════╩═══════════════╩════════════════╝")
}

