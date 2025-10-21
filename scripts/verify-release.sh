#!/bin/bash
# scripts/verify-release.sh
# 验证发布产物的完整性和签名

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS] VERSION

Verify Edge-Link release artifacts.

ARGUMENTS:
    VERSION             Release version to verify (e.g., v1.2.3)

OPTIONS:
    -d, --download-dir DIR  Directory to download artifacts (default: ./downloads)
    -s, --skip-download     Skip downloading, verify existing files
    --verify-gpg            Verify GPG signatures (requires public key)
    -h, --help              Show this help message

EXAMPLES:
    # Download and verify release v1.2.3
    $0 v1.2.3

    # Verify existing downloads without re-downloading
    $0 -s v1.2.3

    # Verify with GPG signature check
    $0 --verify-gpg v1.2.3

EOF
}

# 默认值
DOWNLOAD_DIR="./downloads"
SKIP_DOWNLOAD=false
VERIFY_GPG=false
VERSION=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--download-dir)
            DOWNLOAD_DIR="$2"
            shift 2
            ;;
        -s|--skip-download)
            SKIP_DOWNLOAD=true
            shift
            ;;
        --verify-gpg)
            VERIFY_GPG=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            VERSION="$1"
            shift
            ;;
    esac
done

# 检查版本参数
if [ -z "$VERSION" ]; then
    log_error "Version is required"
    usage
    exit 1
fi

# 验证版本格式
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    log_error "Invalid version format: $VERSION (expected: v1.2.3 or v1.2.3-beta.1)"
    exit 1
fi

# GitHub 仓库信息
REPO_OWNER="edgelink"
REPO_NAME="edge-link"
BASE_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}"

# 创建下载目录
mkdir -p "$DOWNLOAD_DIR"
cd "$DOWNLOAD_DIR"

log_info "Edge-Link Release Verification"
log_info "Version: $VERSION"
log_info "Download directory: $(pwd)"
echo ""

# 下载文件
if [ "$SKIP_DOWNLOAD" = false ]; then
    log_step "Downloading release artifacts..."

    # 下载校验和文件
    log_info "Downloading SHA256SUMS.txt..."
    wget -q --show-progress "${BASE_URL}/SHA256SUMS.txt" || {
        log_error "Failed to download SHA256SUMS.txt"
        exit 1
    }

    # 下载 GPG 签名 (如果启用)
    if [ "$VERIFY_GPG" = true ]; then
        log_info "Downloading SHA256SUMS.txt.sig..."
        wget -q --show-progress "${BASE_URL}/SHA256SUMS.txt.sig" || {
            log_warn "GPG signature not available"
            VERIFY_GPG=false
        }
    fi

    # 读取要下载的文件列表
    log_info "Downloading artifacts..."
    while IFS= read -r line; do
        # 提取文件名 (校验和文件格式: HASH  FILENAME)
        filename=$(echo "$line" | awk '{print $2}')

        if [ ! -f "$filename" ]; then
            log_info "Downloading $filename..."
            wget -q --show-progress "${BASE_URL}/${filename}" || {
                log_warn "Failed to download $filename"
            }
        else
            log_info "Skipping $filename (already exists)"
        fi
    done < SHA256SUMS.txt

    echo ""
fi

# 验证 GPG 签名
if [ "$VERIFY_GPG" = true ]; then
    log_step "Verifying GPG signature..."

    if [ ! -f "SHA256SUMS.txt.sig" ]; then
        log_error "SHA256SUMS.txt.sig not found"
        exit 1
    fi

    if command -v gpg &> /dev/null; then
        if gpg --verify SHA256SUMS.txt.sig SHA256SUMS.txt 2>&1; then
            log_info "✅ GPG signature verified"
        else
            log_error "❌ GPG signature verification failed"
            log_warn "You may need to import the signing key first:"
            log_warn "  gpg --keyserver keyserver.ubuntu.com --recv-keys <KEY_ID>"
            exit 1
        fi
    else
        log_warn "GPG not installed, skipping signature verification"
    fi

    echo ""
fi

# 验证 SHA256 校验和
log_step "Verifying SHA256 checksums..."

if [ ! -f "SHA256SUMS.txt" ]; then
    log_error "SHA256SUMS.txt not found"
    exit 1
fi

# 检查是否有 sha256sum 或 shasum
if command -v sha256sum &> /dev/null; then
    CHECKSUM_CMD="sha256sum"
elif command -v shasum &> /dev/null; then
    CHECKSUM_CMD="shasum -a 256"
else
    log_error "Neither sha256sum nor shasum found"
    exit 1
fi

# 验证每个文件
VERIFIED_COUNT=0
FAILED_COUNT=0
MISSING_COUNT=0

while IFS= read -r line; do
    # 提取文件名和期望的校验和
    expected_hash=$(echo "$line" | awk '{print $1}')
    filename=$(echo "$line" | awk '{print $2}')

    if [ ! -f "$filename" ]; then
        log_warn "Missing: $filename"
        MISSING_COUNT=$((MISSING_COUNT + 1))
        continue
    fi

    # 计算实际校验和
    actual_hash=$($CHECKSUM_CMD "$filename" | awk '{print $1}')

    if [ "$expected_hash" = "$actual_hash" ]; then
        log_info "✅ $filename"
        VERIFIED_COUNT=$((VERIFIED_COUNT + 1))
    else
        log_error "❌ $filename (checksum mismatch)"
        log_error "   Expected: $expected_hash"
        log_error "   Actual:   $actual_hash"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
done < SHA256SUMS.txt

echo ""
log_step "Verification Summary"
echo "Verified: $VERIFIED_COUNT"
echo "Failed:   $FAILED_COUNT"
echo "Missing:  $MISSING_COUNT"

if [ $FAILED_COUNT -gt 0 ]; then
    log_error "❌ Verification failed! Some files have incorrect checksums."
    exit 1
elif [ $MISSING_COUNT -gt 0 ]; then
    log_warn "⚠️  Some files are missing but all existing files are valid."
    exit 0
else
    log_info "✅ All artifacts verified successfully!"
    exit 0
fi
