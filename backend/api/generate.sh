#!/bin/bash
set -e

PROTO_DIR="./proto"
OUT_DIR="./gen/pim/v1"

mkdir -p "$OUT_DIR"

echo "📦 Generating Go code from Protocol Buffer definitions..."

protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$OUT_DIR" \
  --go_opt=paths=source_relative \
  "$PROTO_DIR"/*.proto

echo "✅ Proto generation complete: $OUT_DIR"

