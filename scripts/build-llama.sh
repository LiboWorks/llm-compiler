#!/bin/bash
# build-llama.sh - Build llama.cpp for llm-compiler
# Usage: ./scripts/build-llama.sh [--clean] [--cuda] [--vulkan]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
LLAMA_DIR="$REPO_ROOT/third_party/llama.cpp"
BUILD_DIR="$LLAMA_DIR/build"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default options
CLEAN=false
USE_CUDA=false
USE_VULKAN=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        --cuda)
            USE_CUDA=true
            shift
            ;;
        --vulkan)
            USE_VULKAN=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --clean    Remove existing build directory before building"
            echo "  --cuda     Enable CUDA backend (Linux/Windows, requires CUDA toolkit)"
            echo "  --vulkan   Enable Vulkan backend (Linux/Windows)"
            echo "  --help     Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*)    echo "macos" ;;
        Linux*)     echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)          echo "unknown" ;;
    esac
}

OS=$(detect_os)
echo -e "${GREEN}🔍 Detected OS: $OS${NC}"

# Check if llama.cpp submodule exists
if [ ! -d "$LLAMA_DIR" ] || [ ! -f "$LLAMA_DIR/CMakeLists.txt" ]; then
    echo -e "${YELLOW}📦 llama.cpp submodule not found, initializing...${NC}"
    cd "$REPO_ROOT"
    git submodule update --init --recursive
fi

# Clean if requested
if [ "$CLEAN" = true ] && [ -d "$BUILD_DIR" ]; then
    echo -e "${YELLOW}🧹 Cleaning existing build directory...${NC}"
    rm -rf "$BUILD_DIR"
fi

# Create build directory
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Detect number of CPU cores
if [ "$OS" = "macos" ]; then
    NPROC=$(sysctl -n hw.ncpu)
elif [ "$OS" = "linux" ]; then
    NPROC=$(nproc)
else
    NPROC=${NUMBER_OF_PROCESSORS:-4}
fi

echo -e "${GREEN}🔧 Building with $NPROC parallel jobs${NC}"

# Configure cmake based on OS
configure_cmake() {
    local CMAKE_ARGS=(
        "-DCMAKE_BUILD_TYPE=Release"
        "-DBUILD_SHARED_LIBS=OFF"
    )

    case "$OS" in
        macos)
            echo -e "${GREEN}🍎 Configuring for macOS with Metal backend...${NC}"
            CMAKE_ARGS+=(
                "-DGGML_METAL=ON"
                "-DGGML_BLAS=ON"
                "-DGGML_BLAS_VENDOR=Apple"
            )
            ;;
        linux)
            echo -e "${GREEN}🐧 Configuring for Linux...${NC}"
            if [ "$USE_CUDA" = true ]; then
                echo -e "${GREEN}   CUDA backend enabled${NC}"
                CMAKE_ARGS+=("-DGGML_CUDA=ON")
            elif [ "$USE_VULKAN" = true ]; then
                echo -e "${GREEN}   Vulkan backend enabled${NC}"
                CMAKE_ARGS+=("-DGGML_VULKAN=ON")
            else
                echo -e "${GREEN}   CPU backend (default)${NC}"
                CMAKE_ARGS+=(
                    "-DGGML_METAL=OFF"
                    "-DGGML_BLAS=OFF"
                )
            fi
            CMAKE_ARGS+=("-DLLAMA_CURL=OFF")
            ;;
        windows)
            echo -e "${GREEN}🪟 Configuring for Windows...${NC}"
            if [ "$USE_CUDA" = true ]; then
                echo -e "${GREEN}   CUDA backend enabled${NC}"
                CMAKE_ARGS+=("-DGGML_CUDA=ON")
            elif [ "$USE_VULKAN" = true ]; then
                echo -e "${GREEN}   Vulkan backend enabled${NC}"
                CMAKE_ARGS+=("-DGGML_VULKAN=ON")
            else
                echo -e "${GREEN}   CPU backend (default)${NC}"
                CMAKE_ARGS+=(
                    "-DGGML_METAL=OFF"
                    "-DGGML_BLAS=OFF"
                    "-DGGML_OPENMP=OFF"
                )
            fi
            CMAKE_ARGS+=(
                "-DLLAMA_CURL=OFF"
                "-DLLAMA_BUILD_COMMON=OFF"
            )
            # Use MinGW Makefiles on Windows
            CMAKE_ARGS+=("-G" "MinGW Makefiles")
            ;;
        *)
            echo -e "${RED}❌ Unsupported OS: $OS${NC}"
            exit 1
            ;;
    esac

    echo -e "${GREEN}📋 CMake args: ${CMAKE_ARGS[*]}${NC}"
    cmake .. "${CMAKE_ARGS[@]}"
}

# Build
build_llama() {
    echo -e "${GREEN}🔨 Building llama.cpp...${NC}"
    cmake --build . --config Release -j"$NPROC"
}

# Main
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  llm-compiler: Building llama.cpp                          ${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

configure_cmake
build_llama

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ llama.cpp built successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "Next steps:"
echo -e "  1. Build llm-compiler: ${YELLOW}go build ./cmd/llmc${NC}"
echo -e "  2. Compile a workflow: ${YELLOW}./llmc compile -i example.yaml -o ./build${NC}"
echo -e "  3. Run the binary:     ${YELLOW}./build/example${NC}"
