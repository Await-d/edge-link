#!/bin/bash
# 生成gRPC和Protobuf代码

set -e

# 确保protoc已安装
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc not found. Please install Protocol Buffer Compiler."
    echo "  Ubuntu/Debian: apt-get install -y protobuf-compiler"
    echo "  macOS: brew install protobuf"
    exit 1
fi

# 确保Go插件已安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "Generating Protobuf and gRPC code..."

# 生成device.proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I="${SCRIPT_DIR}" \
    "${SCRIPT_DIR}/device.proto"

# 生成topology.proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I="${SCRIPT_DIR}" \
    "${SCRIPT_DIR}/topology.proto"

# 生成nat.proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I="${SCRIPT_DIR}" \
    "${SCRIPT_DIR}/nat.proto"

echo "✅ Protobuf code generation complete!"
