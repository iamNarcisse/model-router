#!/bin/bash
# scripts/test-route.sh

set -e

ROUTER_ADDR="${ROUTER_ADDR:-localhost:50051}"

echo "Testing route: code generation query"
grpcurl -plaintext -d '{
  "query": "write a python function to sort a list",
  "top_k": 3
}' $ROUTER_ADDR router.RouterService/Route

echo ""
echo "Testing route: summarization query"
grpcurl -plaintext -d '{
  "query": "give me the key points of this document",
  "top_k": 3
}' $ROUTER_ADDR router.RouterService/Route

echo ""
echo "Testing health check"
grpcurl -plaintext $ROUTER_ADDR router.RouterService/HealthCheck