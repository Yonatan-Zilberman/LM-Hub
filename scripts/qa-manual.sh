#!/usr/bin/env bash
set -e

echo "[=] LM Hub QA Smoke Test"

echo "-> Building binary..."
go build -o lmh-test ./cmd/lmh

echo "-> Running unit tests..."
go test ./... -short -count=1

echo "-> Testing headless info..."
./lmh-test info

echo "-> Testing headless models list..."
./lmh-test models list

echo "-> Testing headless memory center..."
./lmh-test memory

echo "[ok] Smoke test passed."
rm -f lmh-test
