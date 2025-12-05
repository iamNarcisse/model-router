# LLM Router

Semantic router for directing user queries to optimal LLM models based on task type using vector similarity.

## Overview

Routes user queries to specialized models (code generation, creative writing, analysis, etc.) by:
1. Embedding the query using sentence transformers
2. Finding nearest route via vector similarity (Qdrant)
3. Returning the best-matched model with confidence score

## Architecture

- **Router Service** (Go): gRPC server handling route requests, caching, and orchestration
- **Embedding Service** (Python): Generates embeddings via sentence-transformers
- **Qdrant**: Vector database storing route embeddings
- **Routes Config**: `configs/routes.yaml` - define models, utterances, and metadata

## Quick Start

### Prerequisites
- Go 1.21+
- Python 3.12+
- Docker & Docker Compose

### Run

```bash
# Start all services
make run-all

# Or run individually
make run-deps      # Start Qdrant + embedding service
make run-router    # Build and run router (local)

# Seed routes to Qdrant
make seed
```

### Test

```bash
# Unit tests
make test

# Integration test
make test-integration
```

## Key Components

### Routes Configuration
`configs/routes.yaml` - Define routes with:
- `name`: Route identifier
- `model`: Target LLM model
- `provider`: Model provider (openai, anthropic, google, deepseek)
- `utterances`: Example queries for this route
- `metadata`: Cost tier, use case, recommendations

### gRPC Services

**RouterService** (exposed to clients):
- `Route(query, top_k, score_threshold)` → route, model, confidence
- `HealthCheck()` → service status
- `ReloadRoutes()` → reload routes without restart

**EmbeddingService** (internal):
- `Embed(text)` → vector
- `EmbedBatch(texts[])` → vectors[]

### Endpoints

Default ports (Docker):
- Router gRPC: `50051`
- Router HTTP/REST: `8080`
- Embedding gRPC: `50052`
- Qdrant: `6333` (HTTP), `6334` (gRPC)

## REST API

The router exposes a REST API via grpc-gateway for easy HTTP access.

### Route a Query

```bash
curl -X POST http://localhost:8080/v1/route \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Write a Python function to sort a list",
    "top_k": 3,
    "score_threshold": 0.5
  }'
```

**Response:**
```json
{
  "route": "code_generation",
  "model": "deepseek-coder-v2",
  "confidence": 1.21,
  "totalLatencyMs": 128,
  "latencyBreakdown": {
    "embeddingMs": 80.07,
    "vectorSearchMs": 9,
    "cacheLookupMs": 0
  },
  "metadata": {
    "provider": "deepseek",
    "cost_tier": "low",
    "use_case": "Software development and code debugging"
  }
}
```

### Health Check

```bash
curl http://localhost:8080/v1/health
```

**Response:**
```json
{
  "healthy": true,
  "embeddingServiceStatus": "healthy",
  "qdrantStatus": "healthy"
}
```

### OpenAPI Documentation

OpenAPI/Swagger spec available at: `docs/api/router.swagger.json`

Import into [Swagger Editor](https://editor.swagger.io/) or [Postman](https://www.postman.com/) for interactive API docs.

## Commands

```bash
make proto              # Generate proto files
make build              # Build Go router binary
make build-docker       # Build Docker images
make run-all           # Start all services
make seed              # Seed routes to Qdrant
make test              # Run tests
make clean             # Clean up binaries and containers
```

## Project Structure

```
llm-router/
├── configs/           # YAML configurations
├── proto/            # gRPC protobuf definitions
├── services/
│   ├── router/       # Go router service
│   └── embedding/    # Python embedding service
├── scripts/          # Seed and utility scripts
├── deployments/      # Docker compose files
└── bin/             # Compiled binaries
```
