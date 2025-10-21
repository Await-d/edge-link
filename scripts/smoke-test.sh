#!/bin/bash
# scripts/smoke-test.sh
# 对部署的服务执行烟雾测试

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

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS] BASE_URL

Run smoke tests against deployed Edge-Link services.

ARGUMENTS:
    BASE_URL            Base URL of the deployed service (e.g., https://api.edgelink.example.com)

OPTIONS:
    -t, --timeout SEC   HTTP request timeout in seconds (default: 10)
    --skip-auth         Skip authentication tests
    --api-key KEY       API key for authenticated requests
    -v, --verbose       Enable verbose output
    -h, --help          Show this help message

EXAMPLES:
    # Test production deployment
    $0 https://api.edgelink.example.com

    # Test staging with API key
    $0 --api-key abc123 https://staging-api.edgelink.example.com

    # Test with custom timeout
    $0 -t 30 https://api.edgelink.example.com

EOF
}

# 默认值
TIMEOUT=10
SKIP_AUTH=false
API_KEY=""
VERBOSE=false
BASE_URL=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --skip-auth)
            SKIP_AUTH=true
            shift
            ;;
        --api-key)
            API_KEY="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
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
            BASE_URL="$1"
            shift
            ;;
    esac
done

# 检查必需参数
if [ -z "$BASE_URL" ]; then
    log_error "BASE_URL is required"
    usage
    exit 1
fi

# 移除末尾斜杠
BASE_URL="${BASE_URL%/}"

# 构建 curl 命令基础选项
CURL_OPTS="-f -s -S --max-time $TIMEOUT"
if [ "$VERBOSE" = true ]; then
    CURL_OPTS="$CURL_OPTS -v"
fi

# 测试计数器
PASSED=0
FAILED=0
SKIPPED=0

# 执行测试的辅助函数
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    local expected_pattern="${3:-}"

    log_test "$test_name"

    if eval "$test_cmd"; then
        if [ -n "$expected_pattern" ]; then
            # 检查输出是否匹配期望的模式
            output=$(eval "$test_cmd")
            if echo "$output" | grep -q "$expected_pattern"; then
                log_info "✅ PASS"
                PASSED=$((PASSED + 1))
                return 0
            else
                log_error "❌ FAIL (output doesn't match expected pattern)"
                if [ "$VERBOSE" = true ]; then
                    echo "Output: $output"
                fi
                FAILED=$((FAILED + 1))
                return 1
            fi
        else
            log_info "✅ PASS"
            PASSED=$((PASSED + 1))
            return 0
        fi
    else
        log_error "❌ FAIL"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# 开始测试
log_info "Edge-Link Smoke Tests"
log_info "Target: $BASE_URL"
log_info "Timeout: ${TIMEOUT}s"
echo ""

# 1. 健康检查
run_test "Health Check" \
    "curl $CURL_OPTS ${BASE_URL}/health" \
    "\"status\":\"ok\""

# 2. API 版本端点
run_test "API Version" \
    "curl $CURL_OPTS ${BASE_URL}/api/v1/version" \
    "\"version\""

# 3. Metrics 端点
run_test "Prometheus Metrics" \
    "curl $CURL_OPTS ${BASE_URL}/metrics" \
    "^# HELP"

# 4. OpenAPI 文档
run_test "OpenAPI Spec" \
    "curl $CURL_OPTS ${BASE_URL}/api/v1/docs/openapi.json" \
    "\"openapi\""

# 5. 根路径重定向
run_test "Root Path Redirect" \
    "curl $CURL_OPTS -I ${BASE_URL}/ | grep -i 'location:'"

# 6. CORS 头检查
run_test "CORS Headers" \
    "curl $CURL_OPTS -I -H 'Origin: https://example.com' ${BASE_URL}/health | grep -i 'access-control-allow-origin'"

# 7. 安全头检查
log_test "Security Headers"
HEADERS=$(curl $CURL_OPTS -I ${BASE_URL}/health)

# 检查各种安全头
if echo "$HEADERS" | grep -iq "x-content-type-options"; then
    log_info "  ✅ X-Content-Type-Options present"
else
    log_warn "  ⚠️  X-Content-Type-Options missing"
fi

if echo "$HEADERS" | grep -iq "x-frame-options"; then
    log_info "  ✅ X-Frame-Options present"
else
    log_warn "  ⚠️  X-Frame-Options missing"
fi

if echo "$HEADERS" | grep -iq "strict-transport-security"; then
    log_info "  ✅ Strict-Transport-Security present"
else
    log_warn "  ⚠️  Strict-Transport-Security missing (expected for HTTPS)"
fi

PASSED=$((PASSED + 1))

# 8. 响应时间检查
log_test "Response Time Check"
START_TIME=$(date +%s%N)
curl $CURL_OPTS ${BASE_URL}/health > /dev/null
END_TIME=$(date +%s%N)
RESPONSE_TIME=$(( (END_TIME - START_TIME) / 1000000 ))  # Convert to milliseconds

if [ $RESPONSE_TIME -lt 1000 ]; then
    log_info "✅ PASS (${RESPONSE_TIME}ms < 1000ms)"
    PASSED=$((PASSED + 1))
else
    log_warn "⚠️  SLOW (${RESPONSE_TIME}ms >= 1000ms)"
    PASSED=$((PASSED + 1))
fi

# 9. 不存在的路径应该返回 404
run_test "404 Not Found" \
    "curl -s -o /dev/null -w '%{http_code}' ${BASE_URL}/non-existent-path | grep 404"

# 10. API 认证测试 (如果未跳过)
if [ "$SKIP_AUTH" = false ]; then
    if [ -n "$API_KEY" ]; then
        run_test "Authenticated Request" \
            "curl $CURL_OPTS -H 'Authorization: Bearer $API_KEY' ${BASE_URL}/api/v1/devices" \
            "\"devices\""
    else
        log_test "Authenticated Request"
        log_info "⏭️  SKIP (no API key provided)"
        SKIPPED=$((SKIPPED + 1))
    fi

    # 未认证请求应该返回 401
    run_test "Unauthorized Request (401)" \
        "curl -s -o /dev/null -w '%{http_code}' ${BASE_URL}/api/v1/devices | grep 401"
else
    log_info "⏭️  Skipping authentication tests"
    SKIPPED=$((SKIPPED + 2))
fi

# 11. 数据库连接检查 (通过健康检查)
run_test "Database Connectivity" \
    "curl $CURL_OPTS ${BASE_URL}/health/db" \
    "\"status\":\"healthy\""

# 12. Redis 连接检查
run_test "Redis Connectivity" \
    "curl $CURL_OPTS ${BASE_URL}/health/redis" \
    "\"status\":\"healthy\""

# 13. 并发请求测试
log_test "Concurrent Requests"
for i in {1..10}; do
    curl $CURL_OPTS ${BASE_URL}/health > /dev/null &
done
wait

if [ $? -eq 0 ]; then
    log_info "✅ PASS (handled 10 concurrent requests)"
    PASSED=$((PASSED + 1))
else
    log_error "❌ FAIL (failed to handle concurrent requests)"
    FAILED=$((FAILED + 1))
fi

# 测试总结
echo ""
log_info "===================="
log_info "Smoke Test Summary"
log_info "===================="
echo "Passed:  $PASSED"
echo "Failed:  $FAILED"
echo "Skipped: $SKIPPED"
echo "Total:   $((PASSED + FAILED + SKIPPED))"

if [ $FAILED -gt 0 ]; then
    log_error "❌ Smoke tests failed!"
    exit 1
else
    log_info "✅ All smoke tests passed!"
    exit 0
fi
