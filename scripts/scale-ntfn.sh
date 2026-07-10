#!/bin/bash
set -e

# Scale the real-time-ntfn deployment without editing any YAML.
#
# Usage:
#   ./scripts/scale-ntfn.sh       # scale to 1 replica (working state)
#   ./scripts/scale-ntfn.sh 2     # scale to 2 replicas (reproduces the fan-out bug)

REPLICAS="${1:-1}"

kubectl scale deploy/real-time-ntfn-service -n chat-app --replicas="$REPLICAS"
kubectl rollout status deploy/real-time-ntfn-service -n chat-app --timeout=90s
kubectl get pods -n chat-app -l app=real-time-ntfn-service -o wide
