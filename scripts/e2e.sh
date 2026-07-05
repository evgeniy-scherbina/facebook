#!/bin/bash
set -e

# Run the end-to-end tests against the real cluster (via the ingress ELB).
#
# Usage:
#   ./scripts/e2e.sh              # run all e2e tests
#   ./scripts/e2e.sh TestFanOut   # run only tests matching a name
#
# Notes:
#   - -tags e2e   : required; the e2e files are behind a //go:build e2e tag
#   - -count=1    : disables Go's test cache (the cluster changes but the cache can't see it)

RUN="${1:-.}"   # default: run everything

go test -tags e2e ./test/e2e/ -run "$RUN" -v -count=1 -timeout 120s
