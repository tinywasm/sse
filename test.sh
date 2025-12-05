#!/bin/bash
set -euo pipefail

NOISE='could not unmarshal event: unknown IPAddressSpace value: Loopback'
TMPDIR="/tmp/crudp-test-$$"
mkdir -p "$TMPDIR"

run_tests() {
  name="$1"
  prefix="$2"
  tags="$3"
  
  printf "\n=== %s Tests ===\n" "$name"
  
  out="$TMPDIR/${name}.out"
  cov="$TMPDIR/${name}.cover.out"
  
  # Test con coverage en un solo comando
  rc=0
  if [ -n "$prefix" ]; then
    eval "$prefix go test -coverprofile=$cov ./... $tags" > "$out" 2>&1 || rc=$?
  else
    go test -coverprofile="$cov" ./... $tags > "$out" 2>&1 || rc=$?
  fi
  
  if [ $rc -ne 0 ]; then
    printf "❌ FAILED\n\n"
    if [ -n "$prefix" ]; then
      eval "$prefix go test -v ./... $tags" 2>&1 | grep -v "$NOISE" || true
    else
      go test -v ./... $tags 2>&1 | grep -v "$NOISE" || true
    fi
    return $rc
  fi
  
  # Solo mostrar coverage total
  if [ -f "$cov" ]; then
    printf "✅ Coverage: "
    go tool cover -func="$cov" | tail -1 | awk '{print $NF}'
  else
    printf "✅ (no coverage data)\n"
  fi
}

# Stdlib
run_tests "Stdlib" "" "" || { rm -rf "$TMPDIR"; exit 1; }

# WASM
if ! command -v wasmbrowsertest >/dev/null 2>&1; then
  printf "\n⚠️  wasmbrowsertest not found\nInstall: go install github.com/agnivade/wasmbrowsertest@latest\n"
  rm -rf "$TMPDIR"
  exit 1
fi

run_tests "WASM" "GOOS=js GOARCH=wasm" "-tags wasm" || { rm -rf "$TMPDIR"; exit 1; }

rm -rf "$TMPDIR"
printf "\n✅ All tests passed!\n"