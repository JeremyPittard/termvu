#!/usr/bin/env bash
# Phase 1 verification script
# Runs termvu with synthetic audio and checks for stable render

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_ROOT/bin/termvu"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Check binary exists
if [[ ! -f "$BINARY" ]]; then
    log_error "Binary not found: $BINARY"
    log_info "Run 'make build' first"
    exit 1
fi

# Check if we can generate synthetic audio
if ! command -v speaker-test &> /dev/null; then
    log_warn "speaker-test not found (alsa-utils). Skipping synthetic audio test."
    log_info "Install alsa-utils or run manually with audio playing."
    exit 0
fi

log_info "Starting Phase 1 verification..."

# Generate synthetic audio in background (440Hz sine wave)
speaker-test -t sine -f 440 -l 1 &
SPEAKER_PID=$!

# Give it a moment to start
sleep 1

# Run termvu for 10 seconds with timeout
log_info "Running termvu for 10 seconds..."
timeout 12s "$BINARY" -backend malgo 2>&1 | tee /tmp/termvu_verify.log &
TERMVU_PID=$!

# Wait for termvu to finish or timeout
wait $TERMVU_PID 2>/dev/null || true

# Kill speaker-test if still running
kill $SPEAKER_PID 2>/dev/null || true
wait $SPEAKER_PID 2>/dev/null || true

# Analyze output
log_info "Analyzing output..."

# Check for crashes
if grep -q "panic\|fatal\|runtime error" /tmp/termvu_verify.log; then
    log_error "Crash detected in output"
    cat /tmp/termvu_verify.log
    exit 1
fi

# Check for "prints lines" failure mode (repeated output without AltScreen)
# This would show many lines of output scrolling
LINE_COUNT=$(wc -l < /tmp/termvu_verify.log)
if [[ $LINE_COUNT -gt 100 ]]; then
    log_warn "High line count ($LINE_COUNT) - possible 'prints lines' issue"
    log_info "Check that AltScreen=true and tea.NewView are used"
fi

# Check for spectrum characters in output
if grep -q -E "[▁▂▃▄▅▆▇█]" /tmp/termvu_verify.log; then
    log_info "Spectrum block characters detected - rendering working"
else
    log_warn "No spectrum characters found in output"
fi

# Check for metadata header
if grep -q -E "Live System Audio|Unknown Track" /tmp/termvu_verify.log; then
    log_info "Metadata fallback label detected"
fi

log_info "Phase 1 verification complete"
log_info "Check /tmp/termvu_verify.log for full output"