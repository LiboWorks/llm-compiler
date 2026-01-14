#!/bin/bash
# run-demo.sh - Compile and run the example workflow
#
# Usage:
#   ./run-demo.sh              # Normal run
#   ./run-demo.sh --parallel   # Run with subprocess workers (parallel LLM)
#   ./run-demo.sh --clean      # Clean and rebuild

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$SCRIPT_DIR/output"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}▶${NC} $1"; }
log_warn() { echo -e "${YELLOW}▶${NC} $1"; }
log_error() { echo -e "${RED}▶${NC} $1"; }

# Parse arguments
PARALLEL=false
CLEAN=false
for arg in "$@"; do
    case $arg in
        --parallel) PARALLEL=true ;;
        --clean) CLEAN=true ;;
        --help|-h)
            echo "Usage: $0 [--parallel] [--clean]"
            echo "  --parallel  Run with LLMC_SUBPROCESS=1 for parallel LLM execution"
            echo "  --clean     Clean output directory before building"
            exit 0
            ;;
    esac
done

cd "$REPO_ROOT"

# Check if llmc CLI exists
if [[ ! -f "./llmc" ]]; then
    log_info "Building llmc CLI..."
    go build ./cmd/llmc
fi

# Clean if requested
if [[ "$CLEAN" == true ]]; then
    log_info "Cleaning output directory..."
    rm -rf "$OUTPUT_DIR"
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Compile the workflow
log_info "Compiling demo/example.yaml..."
./llmc compile -i demo/example.yaml -o "$OUTPUT_DIR" --keep-source

# Run the compiled binary
log_info "Running compiled workflow..."
if [[ "$PARALLEL" == true ]]; then
    log_info "Using subprocess worker mode (LLMC_SUBPROCESS=1)"
    LLMC_SUBPROCESS=1 "$OUTPUT_DIR/example"
else
    "$OUTPUT_DIR/example"
fi

# Show summary
echo ""
log_info "Demo complete! Output files:"
ls -la "$OUTPUT_DIR"

echo ""
log_info "View the JSON output:"
echo "  cat demo/output/example_run.json"
