FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy proto and generate
COPY proto/ ./proto/
RUN python -m grpc_tools.protoc \
    -I./proto \
    --python_out=. \
    --grpc_python_out=. \
    ./proto/router.proto && \
    mv router_pb2.py embedding_pb2.py && \
    mv router_pb2_grpc.py embedding_pb2_grpc.py

# Copy application
COPY main.py .

# Pre-download model during build (optional, speeds up startup)
ARG MODEL_NAME=Qwen/Qwen3-Embedding-0.6B
RUN python -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('${MODEL_NAME}')"

EXPOSE 50052

CMD ["python", "main.py"]