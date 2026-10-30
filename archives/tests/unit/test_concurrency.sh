#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_concurrency.sh
# Unit tests for the parallel task pool and concurrent execution engine.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/concurrency.sh"

echo "  Running Concurrency & Parallel Execution Tests..."

# Test 1: CPU Cores Detection
cores=$(get_cpu_cores)
if ((cores >= 1)); then
	pass "get_cpu_cores detected valid CPU core count: $cores"
else
	fail "get_cpu_cores returned invalid core count: $cores"
fi

# Test 2: Successful Concurrent Task Pool Execution
export MAX_PARALLEL_JOBS=4
start_time=$(date +%s)

pool_output=$(run_parallel_task_pool "Test Pool" \
	"Sleep Task 1:::sleep 0.1" \
	"Sleep Task 2:::sleep 0.1" \
	"Sleep Task 3:::sleep 0.1" \
	"Sleep Task 4:::sleep 0.1" 2>&1)
exit_code=$?

end_time=$(date +%s)
duration=$((end_time - start_time))

assert_exit_code 0 "$exit_code" "Parallel task pool finished successfully" &&
	assert_contains "$pool_output" "All 4 parallel tasks in 'Test Pool' finished successfully" "Success message printed" &&
	pass "run_parallel_task_pool executed 4 concurrent jobs in ${duration}s and returned exit code 0"

# Test 3: Failure Isolation and Exit Code Propagation
fail_output=$(run_parallel_task_pool "Failing Pool" \
	"Succeeding Task:::true" \
	"Failing Task:::sh -c 'echo \"intentional parallel error output\"; exit 42'" 2>&1)
fail_code=$?

if ((fail_code != 0)); then
	assert_contains "$fail_output" "intentional parallel error output" "Captured stdout/stderr from failed worker" &&
		pass "run_parallel_task_pool properly captured task failure and isolated error output"
else
	fail "run_parallel_task_pool failed to report worker exit code failure"
fi
