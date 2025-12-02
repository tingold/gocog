#!/usr/bin/env python3
"""
Benchmark script for comparing rasterio performance with gocog.
This script performs the same operations as the Go benchmarks.
"""

import json
import sys
import time
from pathlib import Path

try:
    import rasterio
    from rasterio.warp import transform_bounds
except ImportError:
    print("ERROR: rasterio not installed. Install with: pip install rasterio", file=sys.stderr)
    sys.exit(1)


def benchmark_open_info(file_path):
    """Benchmark opening a COG and reading metadata with rasterio."""
    start = time.perf_counter()
    
    try:
        with rasterio.open(file_path) as src:
            # Access all metadata (same as Go benchmark)
            bounds = src.bounds
            crs = src.crs
            width = src.width
            height = src.height
            count = src.count
            dtype = src.dtypes[0]
            overviews = src.overviews(1) if src.overviews(1) else []
            overview_count = len(overviews)
            
            # Ensure we actually read the data (not just open)
            _ = src.meta
            
    except Exception as e:
        return {"error": str(e), "duration_ns": 0}
    
    duration_ns = int((time.perf_counter() - start) * 1e9)
    
    return {
        "duration_ns": duration_ns,
        "bounds": {
            "min": [bounds.left, bounds.bottom],
            "max": [bounds.right, bounds.top]
        },
        "crs": str(crs),
        "width": width,
        "height": height,
        "band_count": count,
        "data_type": str(dtype),
        "overview_count": overview_count
    }


def benchmark_open_info_url(url):
    """Benchmark opening a remote COG and reading metadata with rasterio."""
    start = time.perf_counter()
    
    try:
        with rasterio.open(url) as src:
            bounds = src.bounds
            crs = src.crs
            width = src.width
            height = src.height
            count = src.count
            dtype = src.dtypes[0]
            overviews = src.overviews(1) if src.overviews(1) else []
            overview_count = len(overviews)
            
            _ = src.meta
            
    except Exception as e:
        return {"error": str(e), "duration_ns": 0}
    
    duration_ns = int((time.perf_counter() - start) * 1e9)
    
    return {
        "duration_ns": duration_ns,
        "bounds": {
            "min": [bounds.left, bounds.bottom],
            "max": [bounds.right, bounds.top]
        },
        "crs": str(crs),
        "width": width,
        "height": height,
        "band_count": count,
        "data_type": str(dtype),
        "overview_count": overview_count
    }


def run_benchmark(file_path, iterations=100):
    """Run benchmark multiple times and return statistics."""
    durations = []
    errors = []
    
    for i in range(iterations):
        result = benchmark_open_info(file_path)
        if "error" in result:
            errors.append(result["error"])
        else:
            durations.append(result["duration_ns"])
    
    if not durations:
        return {
            "error": f"All {iterations} iterations failed",
            "errors": errors[:5]  # Show first 5 errors
        }
    
    durations.sort()
    avg = sum(durations) / len(durations)
    median = durations[len(durations) // 2]
    min_dur = durations[0]
    max_dur = durations[-1]
    
    return {
        "iterations": iterations,
        "successful": len(durations),
        "failed": len(errors),
        "avg_ns": int(avg),
        "median_ns": int(median),
        "min_ns": int(min_dur),
        "max_ns": int(max_dur),
        "durations_ns": durations if len(durations) <= 10 else durations[:10]  # Sample
    }


def run_benchmark_url(url, iterations=100):
    """Run benchmark multiple times for a URL and return statistics."""
    durations = []
    errors = []
    
    for i in range(iterations):
        result = benchmark_open_info_url(url)
        if "error" in result:
            errors.append(result["error"])
        else:
            durations.append(result["duration_ns"])
    
    if not durations:
        return {
            "error": f"All {iterations} iterations failed",
            "errors": errors[:5]  # Show first 5 errors
        }
    
    durations.sort()
    avg = sum(durations) / len(durations)
    median = durations[len(durations) // 2]
    min_dur = durations[0]
    max_dur = durations[-1]
    
    return {
        "iterations": iterations,
        "successful": len(durations),
        "failed": len(errors),
        "avg_ns": int(avg),
        "median_ns": int(median),
        "min_ns": int(min_dur),
        "max_ns": int(max_dur),
        "durations_ns": durations if len(durations) <= 10 else durations[:10]  # Sample
    }


def main():
    if len(sys.argv) < 2:
        print("Usage: python benchmark_rasterio.py <command> [args...]", file=sys.stderr)
        print("Commands:", file=sys.stderr)
        print("  single <file_path>     - Single benchmark run", file=sys.stderr)
        print("  benchmark <file_path> [iterations] - Multiple iterations", file=sys.stderr)
        print("  url <url>              - Single benchmark run for remote COG", file=sys.stderr)
        print("  benchmark_url <url> [iterations] - Multiple iterations for remote COG", file=sys.stderr)
        sys.exit(1)
    
    command = sys.argv[1]
    
    if command == "single":
        if len(sys.argv) < 3:
            print("ERROR: single command requires file_path", file=sys.stderr)
            sys.exit(1)
        file_path = sys.argv[2]
        result = benchmark_open_info(file_path)
        print(json.dumps(result, indent=2))
        
    elif command == "benchmark":
        if len(sys.argv) < 3:
            print("ERROR: benchmark command requires file_path", file=sys.stderr)
            sys.exit(1)
        file_path = sys.argv[2]
        iterations = int(sys.argv[3]) if len(sys.argv) > 3 else 100
        result = run_benchmark(file_path, iterations)
        print(json.dumps(result, indent=2))
        
    elif command == "url":
        if len(sys.argv) < 3:
            print("ERROR: url command requires url", file=sys.stderr)
            sys.exit(1)
        url = sys.argv[2]
        result = benchmark_open_info_url(url)
        print(json.dumps(result, indent=2))
        
    elif command == "benchmark_url":
        if len(sys.argv) < 3:
            print("ERROR: benchmark_url command requires url", file=sys.stderr)
            sys.exit(1)
        url = sys.argv[2]
        iterations = int(sys.argv[3]) if len(sys.argv) > 3 else 100
        result = run_benchmark_url(url, iterations)
        print(json.dumps(result, indent=2))
        
    else:
        print(f"ERROR: Unknown command: {command}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()

