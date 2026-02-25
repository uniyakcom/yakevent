#!/bin/bash
# yakevent 性能回归守卫
#
# ═══ 核心策略: 绝对值对比 ═══
#
# yakevent 无同类竞品基准作为参照，直接对比当前 ns/op 与本地基线 ns/op。
# 守卫目标（单线程，稳定性高）：
#   BenchmarkImplSync   — 同步模式，~15 ns/op
#   BenchmarkImplAsync  — 异步模式，~33 ns/op
#   BenchmarkImplFlow   — 流水线模式，~70 ns/op
#
# 工作流:
#   1. 运行目标基准测试（count=5 取中位数，抑制 GC 噪声）
#   2. 提取各目标 ns/op 中位数
#   3. 与基线对比，超阈值 → exit 1
#
# 用法:
#   ./bench_guard.sh                     # 回归检查（无基线时自动创建）
#   ./bench_guard.sh force               # 强制重建基线
#   BENCH_THRESHOLD=20 ./bench_guard.sh  # 自定义阈值（默认 15%）
#
# 注意: 跨 Runner / 不同硬件的对比不可靠，本脚本仅用于同环境回归检测。
#       跨环境对比请使用 bench.yml + benchstat。
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS_NAME" in
    mingw*|msys*|cygwin*) OS_NAME="windows" ;;
esac
BASELINE="$SCRIPT_DIR/bench_guard_baseline_${OS_NAME}.txt"
ABS_BASELINE_RAW="$SCRIPT_DIR/bench_guard_raw_${OS_NAME}.txt"   # 每次覆盖：测试数据 + 分析结果
THRESHOLD="${BENCH_THRESHOLD:-15}"   # ns/op 回归阈值百分比
BENCHTIME="${BENCH_TIME:-2s}"
COUNT="${BENCH_COUNT:-5}"

# 守卫的基准测试目标（单线程，稳定）
BENCH_PATTERN='^Benchmark(ImplSync|ImplAsync|ImplFlow)$'

# ── 帮助函数 ──
die() { echo "ERROR: $*" >&2; exit 1; }

# extract_median: 从 benchmark 输出中提取中位数 ns/op
# 参数: <BenchmarkName> <file>
extract_median() {
    local name="$1"
    local file="$2"
    local values
    values=$(grep "^${name}-" "$file" | awk '{for(i=1;i<=NF;i++) if($i=="ns/op") print $(i-1)}' | sort -n)
    local n
    n=$(echo "$values" | grep -c . || true)
    if [[ $n -eq 0 ]]; then
        echo "0"
        return
    fi
    local mid=$(( (n + 1) / 2 ))
    echo "$values" | sed -n "${mid}p"
}

# extract_allocs: 提取 allocs/op 中位数
extract_allocs() {
    local name="$1"
    local file="$2"
    local values
    values=$(grep "^${name}-" "$file" | awk '{for(i=1;i<=NF;i++) if($i=="allocs/op") print $(i-1)}' | sort -n)
    local n
    n=$(echo "$values" | grep -c . || true)
    if [[ $n -eq 0 ]]; then
        echo "0"
        return
    fi
    local mid=$(( (n + 1) / 2 ))
    echo "$values" | sed -n "${mid}p"
}

print_cpu_info() {
    echo "=== CPU topology ==="
    if command -v lscpu &>/dev/null; then
        local sockets cores_per_socket threads_per_core logical
        sockets=$(lscpu | awk '/^Socket\(s\)/{print $2}')
        cores_per_socket=$(lscpu | awk '/^Core\(s\) per socket/{print $NF}')
        threads_per_core=$(lscpu | awk '/^Thread\(s\) per core/{print $NF}')
        logical=$(lscpu | awk '/^CPU\(s\):/{print $2; exit}')
        local physical=$(( sockets * cores_per_socket ))
        echo "  physical cores : $physical  (${sockets} socket × ${cores_per_socket} core/socket)"
        echo "  threads (logical): $logical  (${threads_per_core} thread/core)"
        echo "  model : $(lscpu | awk '/^Model name/{sub(/.*: */,""); print; exit}')"
    elif command -v sysctl &>/dev/null; then
        local physical logical
        physical=$(sysctl -n hw.physicalcpu 2>/dev/null || echo '?')
        logical=$(sysctl -n hw.logicalcpu 2>/dev/null || echo '?')
        echo "  physical cores   : $physical"
        echo "  threads (logical): $logical"
        echo "  model : $(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo 'unknown')"
    else
        echo "  (lscpu / sysctl not available)"
    fi
    echo ""
}

run_bench() {
    local outfile="$1"
    print_cpu_info
    echo "Running benchmarks (pattern=$BENCH_PATTERN, benchtime=$BENCHTIME, count=$COUNT)..."
    cd "$SCRIPT_DIR"
    go test . \
        -bench="$BENCH_PATTERN" \
        -benchmem \
        -benchtime="$BENCHTIME" \
        -count="$COUNT" \
        -run='^$' \
        -timeout=300s 2>&1 \
        | grep --line-buffered '^Benchmark' \
        | tee "$outfile"
    if [[ ! -s "$outfile" ]]; then
        echo "ERROR: No benchmark results collected." >&2
        exit 1
    fi
    echo "  $(wc -l < "$outfile") result lines"
}

# compute_baseline: 提取各目标中位数写入 baseline 文件
# 格式: <Name> <ns/op> <allocs/op>
compute_baseline() {
    local file="$1"
    for name in BenchmarkImplSync BenchmarkImplAsync BenchmarkImplFlow; do
        local ns allocs
        ns=$(extract_median "$name" "$file")
        allocs=$(extract_allocs "$name" "$file")
        if [[ "$ns" != "0" ]]; then
            echo "$name $ns $allocs"
        fi
    done
}

# ── 模式: 更新基线 ──
if [[ "${1:-}" == "force" ]]; then
    run_bench "$ABS_BASELINE_RAW"

    echo ""
    echo "=== Computing baseline ==="
    compute_baseline "$ABS_BASELINE_RAW" > "$BASELINE"

    echo ""
    echo "Baseline updated: $BASELINE"
    echo "Raw results:      $ABS_BASELINE_RAW"
    echo ""
    echo "Baseline values:"
    while IFS=' ' read -r name ns allocs; do
        printf "  %-30s %8s ns/op   %s allocs/op\n" "$name" "$ns" "$allocs"
    done < "$BASELINE"
    exit 0
fi

# ── 模式: CI 回归检查 ──
if [[ ! -f "$BASELINE" ]]; then
    echo "No baseline found, creating automatically..."
    run_bench "$ABS_BASELINE_RAW"
    compute_baseline "$ABS_BASELINE_RAW" > "$BASELINE"
    echo ""
    echo "Baseline created: $BASELINE"
    echo "Baseline values:"
    while IFS=' ' read -r name ns allocs; do
        printf "  %-30s %8s ns/op   %s allocs/op\n" "$name" "$ns" "$allocs"
    done < "$BASELINE"
    echo ""
    echo "✅ Baseline initialized. Run again to check for regressions."
    exit 0
fi

run_bench "$ABS_BASELINE_RAW"

# 分析结果同步写入文件末尾（tee -a 追加到已有测试数据后）
{
    echo ""
    echo "=== Performance regression check (threshold: ${THRESHOLD}%) ==="
    echo ""

    regression_found=false

    while IFS=' ' read -r name base_ns base_allocs; do
        cur_ns=$(extract_median "$name" "$ABS_BASELINE_RAW")
        cur_allocs=$(extract_allocs "$name" "$ABS_BASELINE_RAW")

        if [[ "$cur_ns" == "0" || -z "$cur_ns" ]]; then
            echo "⚠️  $name: no current data (skipped)"
            continue
        fi

        # 计算 ns/op 变化百分比
        pct_change=$(awk "BEGIN{printf \"%.2f\", ($cur_ns - $base_ns) / $base_ns * 100}")
        abs_pct=$(echo "$pct_change" | tr -d '-')
        is_regression=$(awk "BEGIN{print ($pct_change > 0) ? 1 : 0}")
        exceeds=$(awk "BEGIN{print ($abs_pct > $THRESHOLD) ? 1 : 0}")

        if [[ "$is_regression" == "1" && "$exceeds" == "1" ]]; then
            printf "❌ %-30s %8s → %8s ns/op  (%+.1f%% > %s%%)\n" \
                "$name" "$base_ns" "$cur_ns" "$pct_change" "$THRESHOLD"
            regression_found=true
        else
            status="✅"
            [[ "$is_regression" == "1" ]] && status="⚠️ "
            printf "%s %-30s %8s → %8s ns/op  (%+.1f%%)\n" \
                "$status" "$name" "$base_ns" "$cur_ns" "$pct_change"
        fi

        # allocs/op 严格检测（仅报告，不阻塞）
        if [[ "$base_allocs" != "0" && "$cur_allocs" != "$base_allocs" ]]; then
            alloc_change=$(awk "BEGIN{printf \"%.1f\", ($cur_allocs - $base_allocs) / $base_allocs * 100}")
            printf "   ↳ allocs/op: %s → %s (%+.1f%%)\n" "$base_allocs" "$cur_allocs" "$alloc_change"
        fi
        if [[ "$base_allocs" == "0" && "$cur_allocs" != "0" ]]; then
            printf "   ↳ ⚠️  allocs/op: was 0, now %s\n" "$cur_allocs"
        fi
    done < "$BASELINE"

    # ── 可选: benchstat 绝对值展示 ──
    if command -v benchstat &>/dev/null; then
        echo ""
        echo "=== benchstat (informational) ==="
        benchstat "$ABS_BASELINE_RAW" 2>&1 || true
    fi

    echo ""
    if $regression_found; then
        echo "❌ Performance regression detected (ns/op threshold: ${THRESHOLD}%)"
        echo ""
        echo "   To update baseline: $0 force"
    else
        echo "✅ No performance regression (ns/op threshold: ${THRESHOLD}%)"
    fi
} | tee -a "$ABS_BASELINE_RAW"

# tee 完成后从文件检测是否有回归行，决定退出码
if grep -q '^❌ Performance' "$ABS_BASELINE_RAW" 2>/dev/null; then
    exit 1
fi
