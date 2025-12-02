package gocog_test

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/tingold/gocog"
)

// TestCompareGDALvsGoCOG compares the speed of opening a COG and printing info
// using gocog vs GDAL's gdalinfo utility.
func TestCompareGDALvsGoCOG(t *testing.T) {
	testFiles := []string{"TCI.tif", "B12.tif"}

	for _, testFile := range testFiles {
		t.Run(testFile, func(t *testing.T) {
			// Check if gdalinfo is available
			if _, err := exec.LookPath("gdalinfo"); err != nil {
				t.Skipf("gdalinfo not found in PATH, skipping comparison test")
			}

			t.Logf("=== Comparing performance for %s ===\n", testFile)

			// --- gocog timing ---
			gocogStart := time.Now()
			cog, err := gocog.Open(testFile, nil)
			gocogDuration := time.Since(gocogStart)

			if err != nil {
				t.Logf("gocog: Error opening file: %v", err)
			} else {
				t.Logf("\n--- gocog output ---")
				t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]",
					cog.Bounds().Min[0], cog.Bounds().Min[1],
					cog.Bounds().Max[0], cog.Bounds().Max[1])
				t.Logf("CRS: %s", cog.CRS())
				t.Logf("Image size: %dx%d pixels", cog.Width(), cog.Height())
				t.Logf("Bands: %d", cog.BandCount())
				t.Logf("Data type: %v", cog.DataType())
				t.Logf("Overview levels: %d", cog.OverviewCount())
			}

			// --- gdalinfo timing ---
			gdalStart := time.Now()
			cmd := exec.Command("gdalinfo", testFile)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			gdalErr := cmd.Run()
			gdalDuration := time.Since(gdalStart)

			if gdalErr != nil {
				t.Logf("gdalinfo: Error running command: %v\nStderr: %s", gdalErr, stderr.String())
			} else {
				t.Logf("\n--- gdalinfo output (first 1000 chars) ---")
				output := stdout.String()
				if len(output) > 1000 {
					output = output[:1000] + "...\n(truncated)"
				}
				t.Logf("%s", output)
			}

			// --- Results comparison ---
			t.Logf("\n=== TIMING RESULTS for %s ===", testFile)
			t.Logf("gocog:    %v", gocogDuration)
			t.Logf("gdalinfo: %v", gdalDuration)

			if gdalDuration > 0 {
				speedup := float64(gdalDuration) / float64(gocogDuration)
				if speedup > 1 {
					t.Logf("gocog is %.2fx FASTER than gdalinfo", speedup)
				} else {
					t.Logf("gdalinfo is %.2fx faster than gocog", 1/speedup)
				}
			}
		})
	}
}

// BenchmarkGoCOG_OpenInfo benchmarks opening a COG and reading its info with gocog
func BenchmarkGoCOG_OpenInfo(b *testing.B) {
	testFiles := []string{"TCI.tif", "B12.tif"}

	for _, testFile := range testFiles {
		b.Run(testFile, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				cog, err := gocog.Open(testFile, nil)
				if err != nil {
					b.Fatalf("Failed to open COG: %v", err)
				}

				// Access all the metadata (simulating what gdalinfo does)
				_ = cog.Bounds()
				_ = cog.CRS()
				_ = cog.Width()
				_ = cog.Height()
				_ = cog.BandCount()
				_ = cog.DataType()
				_ = cog.OverviewCount()
			}
		})
	}
}

// BenchmarkGDALInfo benchmarks running gdalinfo
func BenchmarkGDALInfo(b *testing.B) {
	// Check if gdalinfo is available
	if _, err := exec.LookPath("gdalinfo"); err != nil {
		b.Skip("gdalinfo not found in PATH, skipping benchmark")
	}

	testFiles := []string{"TCI.tif", "B12.tif"}

	for _, testFile := range testFiles {
		b.Run(testFile, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				cmd := exec.Command("gdalinfo", testFile)
				var stdout bytes.Buffer
				cmd.Stdout = &stdout
				if err := cmd.Run(); err != nil {
					b.Fatalf("gdalinfo failed: %v", err)
				}
				// Read the output to ensure the command completed
				_ = stdout.Bytes()
			}
		})
	}
}

// TestCompareGDALvsGoCOG_URL compares performance for remote COGs (HTTP)
func TestCompareGDALvsGoCOG_URL(t *testing.T) {
	testURL := "https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif"

	// Check if gdalinfo is available
	if _, err := exec.LookPath("gdalinfo"); err != nil {
		t.Skip("gdalinfo not found in PATH, skipping comparison test")
	}

	t.Logf("=== Comparing performance for remote COG ===")
	t.Logf("URL: %s", testURL)

	// --- gocog timing ---
	gocogStart := time.Now()
	cog, err := gocog.Open(testURL, nil)
	gocogDuration := time.Since(gocogStart)

	if err != nil {
		t.Logf("gocog: Error opening URL: %v", err)
	} else {
		t.Logf("\n--- gocog output ---")
		t.Logf("Bounds: Min=[%.6f, %.6f], Max=[%.6f, %.6f]",
			cog.Bounds().Min[0], cog.Bounds().Min[1],
			cog.Bounds().Max[0], cog.Bounds().Max[1])
		t.Logf("CRS: %s", cog.CRS())
		t.Logf("Image size: %dx%d pixels", cog.Width(), cog.Height())
		t.Logf("Bands: %d", cog.BandCount())
		t.Logf("Overview levels: %d", cog.OverviewCount())
	}

	// --- gdalinfo timing (with /vsicurl/ prefix for HTTP access) ---
	gdalURL := fmt.Sprintf("/vsicurl/%s", testURL)
	gdalStart := time.Now()
	cmd := exec.Command("gdalinfo", gdalURL)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	gdalErr := cmd.Run()
	gdalDuration := time.Since(gdalStart)

	if gdalErr != nil {
		t.Logf("gdalinfo: Error running command: %v\nStderr: %s", gdalErr, stderr.String())
	} else {
		t.Logf("\n--- gdalinfo output (first 1000 chars) ---")
		output := stdout.String()
		if len(output) > 1000 {
			output = output[:1000] + "...\n(truncated)"
		}
		t.Logf("%s", output)
	}

	// --- Results comparison ---
	t.Logf("\n=== TIMING RESULTS for remote COG ===")
	t.Logf("gocog:    %v", gocogDuration)
	t.Logf("gdalinfo: %v", gdalDuration)

	if gdalDuration > 0 && gocogDuration > 0 {
		speedup := float64(gdalDuration) / float64(gocogDuration)
		if speedup > 1 {
			t.Logf("gocog is %.2fx FASTER than gdalinfo", speedup)
		} else {
			t.Logf("gdalinfo is %.2fx faster than gocog", 1/speedup)
		}
	}
}

// BenchmarkGoCOG_OpenInfo_URL benchmarks opening a remote COG with gocog
func BenchmarkGoCOG_OpenInfo_URL(b *testing.B) {
	testURL := "https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cog, err := gocog.Open(testURL, nil)
		if err != nil {
			b.Fatalf("Failed to open COG: %v", err)
		}

		// Access all the metadata
		_ = cog.Bounds()
		_ = cog.CRS()
		_ = cog.Width()
		_ = cog.Height()
		_ = cog.BandCount()
		_ = cog.DataType()
		_ = cog.OverviewCount()
	}
}

// BenchmarkGDALInfo_URL benchmarks running gdalinfo on a remote COG
func BenchmarkGDALInfo_URL(b *testing.B) {
	// Check if gdalinfo is available
	if _, err := exec.LookPath("gdalinfo"); err != nil {
		b.Skip("gdalinfo not found in PATH, skipping benchmark")
	}

	testURL := "/vsicurl/https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command("gdalinfo", testURL)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			b.Fatalf("gdalinfo failed: %v", err)
		}
		_ = stdout.Bytes()
	}
}

// TestPrintSummary prints a summary table of timing results
func TestPrintSummary(t *testing.T) {
	// Check if gdalinfo is available
	if _, err := exec.LookPath("gdalinfo"); err != nil {
		t.Skip("gdalinfo not found in PATH, skipping summary test")
	}

	testCases := []struct {
		name   string
		path   string
		isURL  bool
	}{
		{"TCI.tif (local)", "TCI.tif", false},
		{"B12.tif (local)", "B12.tif", false},
	}

	t.Logf("\n")
	t.Logf("╔════════════════════════════════════════════════════════════════════════╗")
	t.Logf("║                    GOCOG vs GDALINFO PERFORMANCE                       ║")
	t.Logf("╠═══════════════════════╦═══════════════╦═══════════════╦════════════════╣")
	t.Logf("║ File                  ║ gocog         ║ gdalinfo      ║ Speedup        ║")
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

		// Time gdalinfo
		path := tc.path
		if tc.isURL {
			path = "/vsicurl/" + tc.path
		}
		gdalStart := time.Now()
		cmd := exec.Command("gdalinfo", path)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		gdalErr := cmd.Run()
		gdalDuration := time.Since(gdalStart)

		if gdalErr != nil {
			t.Logf("║ %-21s ║ %-13v ║ ERROR         ║                ║", tc.name, gocogDuration.Round(time.Microsecond))
			continue
		}

		speedup := float64(gdalDuration) / float64(gocogDuration)
		speedupStr := fmt.Sprintf("%.2fx faster", speedup)
		if speedup < 1 {
			speedupStr = fmt.Sprintf("%.2fx slower", 1/speedup)
		}

		t.Logf("║ %-21s ║ %-13v ║ %-13v ║ %-14s ║",
			tc.name,
			gocogDuration.Round(time.Microsecond),
			gdalDuration.Round(time.Microsecond),
			speedupStr)
	}

	t.Logf("╚═══════════════════════╩═══════════════╩═══════════════╩════════════════╝")
}

