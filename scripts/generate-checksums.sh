#!/bin/bash
# scripts/generate-checksums.sh
# 为构建产物生成 SHA256 校验和

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 使用说明
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Generate SHA256 checksums for build artifacts.

OPTIONS:
    -d, --directory DIR     Directory containing artifacts (default: ./release)
    -o, --output FILE       Output checksum file (default: SHA256SUMS.txt)
    -p, --pattern PATTERN   File pattern to include (default: *)
    -v, --verify            Verify existing checksums instead of generating
    -h, --help              Show this help message

EXAMPLES:
    # Generate checksums for all files in release directory
    $0 -d ./release

    # Generate checksums for specific file types
    $0 -d ./release -p "*.tgz"

    # Verify existing checksums
    $0 -d ./release --verify

EOF
}

# 默认值
ARTIFACT_DIR="./release"
OUTPUT_FILE="SHA256SUMS.txt"
FILE_PATTERN="*"
VERIFY_MODE=false

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--directory)
            ARTIFACT_DIR="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -p|--pattern)
            FILE_PATTERN="$2"
            shift 2
            ;;
        -v|--verify)
            VERIFY_MODE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# 检查目录是否存在
if [ ! -d "$ARTIFACT_DIR" ]; then
    log_error "Directory not found: $ARTIFACT_DIR"
    exit 1
fi

# 进入产物目录
cd "$ARTIFACT_DIR"

if [ "$VERIFY_MODE" = true ]; then
    # 验证模式
    log_info "Verifying checksums from $OUTPUT_FILE..."

    if [ ! -f "$OUTPUT_FILE" ]; then
        log_error "Checksum file not found: $OUTPUT_FILE"
        exit 1
    fi

    # 使用 sha256sum 验证
    if sha256sum -c "$OUTPUT_FILE"; then
        log_info "✅ All checksums verified successfully!"
        exit 0
    else
        log_error "❌ Checksum verification failed!"
        exit 1
    fi
else
    # 生成模式
    log_info "Generating checksums for files matching: $FILE_PATTERN"
    log_info "Output file: $OUTPUT_FILE"

    # 查找匹配的文件
    FILES=$(find . -maxdepth 1 -type f -name "$FILE_PATTERN" ! -name "$OUTPUT_FILE" ! -name "*.sha256" | sort)

    if [ -z "$FILES" ]; then
        log_warn "No files found matching pattern: $FILE_PATTERN"
        exit 0
    fi

    # 删除旧的校验和文件
    if [ -f "$OUTPUT_FILE" ]; then
        log_warn "Removing existing checksum file: $OUTPUT_FILE"
        rm "$OUTPUT_FILE"
    fi

    # 生成校验和
    FILE_COUNT=0
    for file in $FILES; do
        # 移除 ./ 前缀
        file=${file#./}

        log_info "Processing: $file"

        # 计算 SHA256
        if command -v sha256sum &> /dev/null; then
            # Linux
            sha256sum "$file" >> "$OUTPUT_FILE"
        elif command -v shasum &> /dev/null; then
            # macOS
            shasum -a 256 "$file" >> "$OUTPUT_FILE"
        else
            log_error "Neither sha256sum nor shasum found!"
            exit 1
        fi

        # 为每个文件生成单独的 .sha256 文件
        if command -v sha256sum &> /dev/null; then
            sha256sum "$file" > "$file.sha256"
        else
            shasum -a 256 "$file" > "$file.sha256"
        fi

        FILE_COUNT=$((FILE_COUNT + 1))
    done

    log_info "✅ Generated checksums for $FILE_COUNT files"
    log_info "Checksum file: $(pwd)/$OUTPUT_FILE"

    # 显示生成的校验和
    echo ""
    echo "=== Generated Checksums ==="
    cat "$OUTPUT_FILE"
    echo "==========================="
fi
