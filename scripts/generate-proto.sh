#!/bin/bash
# scripts/generate-proto.sh

set -e

PROTO_DIR="proto"
GO_OUT="services/router/pkg/pb"
PY_OUT="services/embedding/src/pb"

# Get paths from go.mod in router service
GATEWAY_PATH=$(cd services/router && go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/v2)

echo "Generating Go proto..."
mkdir -p $GO_OUT
mkdir -p docs/api
protoc \
    --proto_path=$PROTO_DIR \
    --proto_path=$GATEWAY_PATH \
    --go_out=$GO_OUT \
    --go_opt=paths=source_relative \
    --go-grpc_out=$GO_OUT \
    --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out=$GO_OUT \
    --grpc-gateway_opt=paths=source_relative \
    --grpc-gateway_opt=generate_unbound_methods=true \
    --openapiv2_out=docs/api \
    --openapiv2_opt=allow_merge=true,merge_file_name=router \
    $PROTO_DIR/router.proto

echo "Generating Python proto..."
mkdir -p $PY_OUT
python -m grpc_tools.protoc \
    -I$PROTO_DIR \
    --python_out=$PY_OUT \
    --grpc_python_out=$PY_OUT \
    $PROTO_DIR/router.proto

# Fix Python imports
sed -i '' 's/import router_pb2/from . import router_pb2/' $PY_OUT/router_pb2_grpc.py

# Create __init__.py
cat > $PY_OUT/__init__.py << 'EOF'
from .router_pb2 import *
from .router_pb2_grpc import *
EOF

echo "Proto generation complete"