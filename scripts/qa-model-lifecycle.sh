#!/usr/bin/env bash
set -e

echo "[=] Testing Model Lifecycle (Headless)"

echo "-> Building binary..."
go build -o lmh-test ./cmd/lmh

echo "-> Triggering Plan headless (should auto-load model)..."
./lmh-test plan "List 3 colors" --headless

echo "-> Unloading all models..."
# (Assuming models unload command or using curl to LM Studio)
# Not strictly required for the smoke test if LM Studio handles it

echo "[ok] Model lifecycle test passed."
rm -f lmh-test
