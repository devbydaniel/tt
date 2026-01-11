#!/bin/bash
#
# End-to-end sync tests for tt
# Tests synchronization between two clients and a server
#

set -e  # Exit on error

# Configuration
TEST_DIR="./test-sync"
API_KEY="test-api-key-e2e"
SERVER_PORT=18080
SERVER_PID=""
PASSED=0
FAILED=0
TOTAL=10

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Cleanup function (called on EXIT)
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -rf "$TEST_DIR"
    echo "Done."
}
trap cleanup EXIT

# Helper: run command as client1
client1() {
    TT_DATA_DIR="$TEST_DIR/client1" \
    TT_CLIENT_ID="test-client-1" \
    TT_SYNC_URL="http://localhost:$SERVER_PORT" \
    TT_SYNC_API_KEY="$API_KEY" \
    ./tt "$@"
}

# Helper: run command as client2
client2() {
    TT_DATA_DIR="$TEST_DIR/client2" \
    TT_CLIENT_ID="test-client-2" \
    TT_SYNC_URL="http://localhost:$SERVER_PORT" \
    TT_SYNC_API_KEY="$API_KEY" \
    ./tt "$@"
}

# Wait for server to be ready
wait_for_server() {
    local max_attempts=30
    local attempt=0
    while ! curl -s "http://localhost:$SERVER_PORT/health" > /dev/null 2>&1; do
        attempt=$((attempt + 1))
        if [ $attempt -ge $max_attempts ]; then
            echo -e "${RED}Server failed to start after $max_attempts attempts${NC}"
            exit 1
        fi
        sleep 0.1
    done
}

# Assert task exists with title on a client
assert_task_exists() {
    local client_func=$1
    local title=$2
    if $client_func list --json 2>/dev/null | jq -e ".[] | select(.title == \"$title\")" > /dev/null 2>&1; then
        return 0
    else
        echo -e "  ${RED}FAIL: Task '$title' not found${NC}"
        return 1
    fi
}

# Assert task does NOT exist with title on a client
assert_task_not_exists() {
    local client_func=$1
    local title=$2
    if $client_func list --json 2>/dev/null | jq -e ".[] | select(.title == \"$title\")" > /dev/null 2>&1; then
        echo -e "  ${RED}FAIL: Task '$title' should not exist${NC}"
        return 1
    else
        return 0
    fi
}

# Assert area exists on a client
assert_area_exists() {
    local client_func=$1
    local name=$2
    if $client_func area list --json 2>/dev/null | jq -e ".[] | select(.name == \"$name\")" > /dev/null 2>&1; then
        return 0
    else
        echo -e "  ${RED}FAIL: Area '$name' not found${NC}"
        return 1
    fi
}

# Assert project exists on a client
assert_project_exists() {
    local client_func=$1
    local title=$2
    if $client_func project list --json 2>/dev/null | jq -e ".[] | select(.title == \"$title\")" > /dev/null 2>&1; then
        return 0
    else
        echo -e "  ${RED}FAIL: Project '$title' not found${NC}"
        return 1
    fi
}

# Assert task is completed (in log)
assert_task_completed() {
    local client_func=$1
    local title=$2
    if $client_func log --json 2>/dev/null | jq -e ".[] | select(.title == \"$title\")" > /dev/null 2>&1; then
        return 0
    else
        echo -e "  ${RED}FAIL: Completed task '$title' not found in log${NC}"
        return 1
    fi
}

# Assert task has expected area
assert_task_has_area() {
    local client_func=$1
    local title=$2
    local area=$3
    if $client_func list --json 2>/dev/null | jq -e ".[] | select(.title == \"$title\" and .areaName == \"$area\")" > /dev/null 2>&1; then
        return 0
    else
        echo -e "  ${RED}FAIL: Task '$title' should have area '$area'${NC}"
        return 1
    fi
}

# Assert task has expected title (for updates)
assert_task_has_title() {
    local client_func=$1
    local id=$2
    local expected_title=$3
    local actual_title
    actual_title=$($client_func list --json 2>/dev/null | jq -r ".[] | select(.id == $id) | .title")
    if [ "$actual_title" = "$expected_title" ]; then
        return 0
    else
        echo -e "  ${RED}FAIL: Task $id should have title '$expected_title' but has '$actual_title'${NC}"
        return 1
    fi
}

# Get task count
get_task_count() {
    local client_func=$1
    $client_func list --json 2>/dev/null | jq 'length'
}

# Get task ID by title
get_task_id() {
    local client_func=$1
    local title=$2
    $client_func list --json 2>/dev/null | jq -r ".[] | select(.title == \"$title\") | .id" | head -1
}

# Run a test
run_test() {
    local name=$1
    local test_func=$2
    echo -e "${BLUE}[TEST $((PASSED + FAILED + 1))/$TOTAL]${NC} $name"
    if $test_func; then
        echo -e "  ${GREEN}PASS${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "  ${RED}FAIL${NC}"
        FAILED=$((FAILED + 1))
    fi
}

# ============================================================================
# Test 1: Basic sync
# ============================================================================
test_basic_sync() {
    client1 add "Basic sync task" > /dev/null
    client1 sync > /dev/null
    client2 sync > /dev/null
    assert_task_exists client2 "Basic sync task"
}

# ============================================================================
# Test 2: Bidirectional sync
# ============================================================================
test_bidirectional_sync() {
    client1 add "Task from client1" > /dev/null
    client2 add "Task from client2" > /dev/null

    # Sync client1 first (pushes its task)
    client1 sync > /dev/null

    # Sync client2 (pushes its task, pulls client1's task)
    client2 sync > /dev/null

    # Sync client1 again (pulls client2's task)
    client1 sync > /dev/null

    # Both clients should have both tasks
    assert_task_exists client1 "Task from client1" && \
    assert_task_exists client1 "Task from client2" && \
    assert_task_exists client2 "Task from client1" && \
    assert_task_exists client2 "Task from client2"
}

# ============================================================================
# Test 3: Task update sync
# ============================================================================
test_update_sync() {
    client1 add "Original title" > /dev/null
    client1 sync > /dev/null
    client2 sync > /dev/null

    # Get task ID on client1 and update it
    local id
    id=$(get_task_id client1 "Original title")
    client1 edit "$id" --title "Updated title" > /dev/null

    # Sync both
    client1 sync > /dev/null
    client2 sync > /dev/null

    # Client2 should have the updated title
    assert_task_exists client2 "Updated title" && \
    assert_task_not_exists client2 "Original title"
}

# ============================================================================
# Test 4: Task deletion sync
# ============================================================================
test_deletion_sync() {
    client1 add "Task to delete" > /dev/null
    client1 sync > /dev/null
    client2 sync > /dev/null

    # Verify task exists on client2
    assert_task_exists client2 "Task to delete" || return 1

    # Delete on client1
    local id
    id=$(get_task_id client1 "Task to delete")
    client1 delete "$id" > /dev/null

    # Sync both
    client1 sync > /dev/null
    client2 sync > /dev/null

    # Task should be gone from client2
    assert_task_not_exists client2 "Task to delete"
}

# ============================================================================
# Test 5: Task completion sync
# ============================================================================
test_completion_sync() {
    client1 add "Task to complete" > /dev/null
    client1 sync > /dev/null
    client2 sync > /dev/null

    # Complete on client1
    local id
    id=$(get_task_id client1 "Task to complete")
    client1 do "$id" > /dev/null

    # Sync both
    client1 sync > /dev/null
    client2 sync > /dev/null

    # Task should be in client2's log (completed tasks)
    assert_task_completed client2 "Task to complete"
}

# ============================================================================
# Test 6: Area sync
# ============================================================================
test_area_sync() {
    client1 area add "Work" > /dev/null
    client1 sync > /dev/null
    client2 sync > /dev/null

    assert_area_exists client2 "Work"
}

# ============================================================================
# Test 7: Task with area sync
# ============================================================================
test_task_with_area_sync() {
    client1 area add "Health" > /dev/null
    client1 add "Exercise daily" -a "Health" > /dev/null
    client1 sync > /dev/null
    client2 sync > /dev/null

    assert_task_exists client2 "Exercise daily" && \
    assert_task_has_area client2 "Exercise daily" "Health"
}

# ============================================================================
# Test 8: Project sync
# ============================================================================
test_project_sync() {
    # Create a project using the project add command
    client1 project add "My Project" > /dev/null
    client1 add "Project subtask" -p "My Project" > /dev/null
    client1 sync > /dev/null
    client2 sync > /dev/null

    # Verify project and subtask exist on client2
    assert_project_exists client2 "My Project" && \
    assert_task_exists client2 "Project subtask"
}

# ============================================================================
# Test 9: Empty sync (no changes)
# ============================================================================
test_empty_sync() {
    # Run sync with no pending changes
    local output1 output2
    output1=$(client1 sync 2>&1)
    output2=$(client2 sync 2>&1)

    # Should complete without error
    if [[ "$output1" == *"error"* ]] || [[ "$output2" == *"error"* ]]; then
        echo -e "  ${RED}FAIL: Empty sync returned error${NC}"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 10: Reset sync
# ============================================================================
test_reset_sync() {
    client1 add "Reset test task" > /dev/null
    client1 sync > /dev/null

    # Reset sync on client1
    echo "y" | client1 sync reset > /dev/null 2>&1

    # Task should still exist locally (reset only clears sync events)
    assert_task_exists client1 "Reset test task"
}

# ============================================================================
# Main
# ============================================================================
main() {
    echo -e "${BLUE}=== Sync E2E Tests ===${NC}"

    # Build binaries
    echo "Building binaries..."
    make build > /dev/null 2>&1
    make build-sync > /dev/null 2>&1

    # Create test directories
    mkdir -p "$TEST_DIR/server" "$TEST_DIR/client1" "$TEST_DIR/client2"

    # Start sync server
    echo "Starting sync server on port $SERVER_PORT..."
    TT_DATA_DIR="$TEST_DIR/server" \
    TT_SYNC_API_KEY="$API_KEY" \
    PORT="$SERVER_PORT" \
    ./tt-sync > "$TEST_DIR/server.log" 2>&1 &
    SERVER_PID=$!

    wait_for_server
    echo -e "Server ready.\n"

    # Run tests
    run_test "Basic sync" test_basic_sync
    run_test "Bidirectional sync" test_bidirectional_sync
    run_test "Task update sync" test_update_sync
    run_test "Task deletion sync" test_deletion_sync
    run_test "Task completion sync" test_completion_sync
    run_test "Area sync" test_area_sync
    run_test "Task with area sync" test_task_with_area_sync
    run_test "Project sync" test_project_sync
    run_test "Empty sync (no changes)" test_empty_sync
    run_test "Reset sync" test_reset_sync

    # Results
    echo -e "\n${BLUE}=== Results ===${NC}"
    echo -e "Passed: ${GREEN}$PASSED${NC}/$TOTAL"
    if [ $FAILED -gt 0 ]; then
        echo -e "Failed: ${RED}$FAILED${NC}/$TOTAL"
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
    fi
}

main "$@"
