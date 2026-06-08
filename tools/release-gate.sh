#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${RELEASE_GATE_LOG_DIR:-$ROOT_DIR/build/release-evidence/release-gate}"

if [[ -t 1 ]]; then
  BOLD="$(printf '\033[1m')"
  GREEN="$(printf '\033[32m')"
  RED="$(printf '\033[31m')"
  YELLOW="$(printf '\033[33m')"
  DIM="$(printf '\033[2m')"
  RESET="$(printf '\033[0m')"
else
  BOLD=""
  GREEN=""
  RED=""
  YELLOW=""
  DIM=""
  RESET=""
fi

mkdir -p "$LOG_DIR" || exit 1
cd "$ROOT_DIR" || exit 1

total=0
passed=0
failed=0
failed_steps=()

print_header() {
  printf "%sRelease gate%s\n" "$BOLD" "$RESET"
  printf "%sLogs: %s%s\n\n" "$DIM" "$LOG_DIR" "$RESET"
}

run_step() {
  local id="$1"
  local title="$2"
  shift 2

  local log_file="$LOG_DIR/$id.log"
  local started_at ended_at duration

  total=$((total + 1))
  started_at="$(date +%s)"

  printf "  %-28s" "$title"

  if "$@" >"$log_file" 2>&1; then
    ended_at="$(date +%s)"
    duration=$((ended_at - started_at))
    passed=$((passed + 1))
    printf "%sOK%s %s(%ss)%s\n" "$GREEN" "$RESET" "$DIM" "$duration" "$RESET"
    return 0
  fi

  ended_at="$(date +%s)"
  duration=$((ended_at - started_at))
  failed=$((failed + 1))
  failed_steps+=("$title")
  printf "%sFAIL%s %s(%ss, log: %s)%s\n" "$RED" "$RESET" "$DIM" "$duration" "$log_file" "$RESET"
  return 1
}

print_summary() {
  printf "\n%sSummary%s\n" "$BOLD" "$RESET"
  printf "  Passed: %s\n" "$passed"
  printf "  Failed: %s\n" "$failed"
  printf "  Total:  %s\n" "$total"

  if ((failed > 0)); then
    printf "\n%sFailed steps:%s\n" "$YELLOW" "$RESET"
    local step
    for step in "${failed_steps[@]}"; do
      printf "  - %s\n" "$step"
    done
    printf "\n%sRelease gate failed.%s See logs above for command output.\n" "$RED" "$RESET"
    return 1
  fi

  printf "\n%sRelease gate passed.%s\n" "$GREEN" "$RESET"
  return 0
}

print_header

run_step "check-release-env" "Check release env" make --no-print-directory check-release-env
run_step "release-assets-check" "Release assets check" make --no-print-directory release-assets-check
run_step "go-test" "Go tests" make --no-print-directory go-test
run_step "go-vet" "Go vet" make --no-print-directory go-vet
run_step "govulncheck" "Go vulnerability check" make --no-print-directory govulncheck
run_step "frontend-ci" "Frontend dependencies" make --no-print-directory frontend-ci
run_step "frontend-lint" "Frontend lint" make --no-print-directory frontend-lint
run_step "frontend-test" "Frontend tests" make --no-print-directory frontend-test
run_step "frontend-build" "Frontend build" make --no-print-directory frontend-build
run_step "npm-audit" "NPM audit" make --no-print-directory npm-audit

print_summary
