#! /usr/bin/env bash

set -e

PROTOFILE_URL="https://raw.githubusercontent.com/google/perfetto/refs/heads/main/protos/perfetto/trace/perfetto_trace.proto"

# download protobuf file
curl --silent --fail \
	 -o internal/proto/perfetto_trace.proto "$PROTOFILE_URL" || \
	{ echo "Download failed, aborting."; exit; };
echo "Downloaded protofile"

# generate go code
cd internal/proto/
protoc --go_out=. --go_opt=paths=source_relative perfetto_trace.proto
echo "Generated Go code"

# run package tests
echo "Running tests..."
cd -
go test -v .
