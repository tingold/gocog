#!/bin/bash
# Comparison script that runs both Go and Python benchmarks and compares results

set -e

echo "=========================================="
echo "  GoCOG vs Rasterio Benchmark Comparison"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if test files exist
if [ ! -f "TCI.tif" ] && [ ! -f "B12.tif" ]; then
    echo -e "${YELLOW}Warning: Test files (TCI.tif, B12.tif) not found in current directory${NC}"
    echo "Some benchmarks may be skipped."
    echo ""
    # todo: download the files
fi


echo "Running Go benchmarks (local files)..."
echo "--------------------------------------"
go test -bench=BenchmarkGoCOG_OpenInfo -benchmem -run=^$ ./... | grep -E "(Benchmark|TCI|B12)" || true
echo ""

echo "Running Go benchmarks (remote URLs)..."
echo "--------------------------------------"
go test -bench=BenchmarkGoCOG_OpenInfo_URL -benchmem -run=^$ ./... | grep -E "(Benchmark|URL)" || true
echo ""

echo "Running Python benchmarks (local files)..."
echo "------------------------------------------"
for file in TCI.tif B12.tif; do
    if [ -f "$file" ]; then
        echo "Benchmarking $file..."
        uv run benchmark_rasterio.py benchmark "$file" 1000 2>/dev/null | uv run python -m json.tool || echo "Failed to benchmark $file"
        echo ""
    fi
done

echo "Running Python benchmarks (remote URLs)..."
echo "------------------------------------------"
test_url="https://f004.backblazeb2.com/file/ei-imagery/s2/18SUJ_2024-01-01_2025-01-01/TCI.tif"
echo "Benchmarking $test_url..."
uv run benchmark_rasterio.py benchmark_url "$test_url" 1000 2>/dev/null | uv run python -m json.tool || echo "Failed to benchmark URL"
echo ""

echo "Running comparison test..."
echo "-------------------------"
go test -v -run=TestPrintRasterioSummary ./... || echo "Comparison test failed"

echo ""
echo "=========================================="
echo "  Comparison Complete"
echo "=========================================="

