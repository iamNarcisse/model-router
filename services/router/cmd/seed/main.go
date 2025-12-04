// cmd/seed/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"

	pb "github.com/iamnarcisse/llm-router/proto"
)

type RoutesConfig struct {
	Routes []RouteDefinition `yaml:"routes"`
}

type RouteDefinition struct {
	Name       string            `yaml:"name"`
	Model      string            `yaml:"model"`
	Provider   string            `yaml:"provider"`
	Utterances []string          `yaml:"utterances"`
	Metadata   map[string]string `yaml:"metadata"`
}

func main() {
	ctx := context.Background()

	// Config
	routesPath := getEnv("ROUTES_PATH", "configs/routes.yaml")
	embeddingAddr := getEnv("EMBEDDING_ADDR", "localhost:50052")
	qdrantHost := getEnv("QDRANT_HOST", "localhost")
	qdrantPort := 6334
	collectionName := getEnv("COLLECTION_NAME", "llm_routes")

	// Load routes
	log.Printf("Loading routes from %s", routesPath)
	routesData, err := os.ReadFile(routesPath)
	if err != nil {
		log.Fatalf("Failed to read routes file: %v", err)
	}

	var config RoutesConfig
	if err := yaml.Unmarshal(routesData, &config); err != nil {
		log.Fatalf("Failed to parse routes: %v", err)
	}

	// Connect to embedding service
	log.Printf("Connecting to embedding service at %s", embeddingAddr)
	conn, err := grpc.Dial(embeddingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to embedding service: %v", err)
	}
	defer conn.Close()
	embedClient := pb.NewEmbeddingServiceClient(conn)

	// Connect to Qdrant
	log.Printf("Connecting to Qdrant at %s:%d", qdrantHost, qdrantPort)
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: qdrantHost,
		Port: qdrantPort,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer qdrantClient.Close()

	// Get dimensions from a test embedding
	testResp, err := embedClient.Embed(ctx, &pb.EmbedRequest{Text: "test"})
	if err != nil {
		log.Fatalf("Failed to get test embedding: %v", err)
	}
	dimensions := uint64(len(testResp.Vector))
	log.Printf("Embedding dimensions: %d", dimensions)

	// Recreate collection
	log.Printf("Creating collection: %s", collectionName)
	exists, _ := qdrantClient.CollectionExists(ctx, collectionName)
	if exists {
		qdrantClient.DeleteCollection(ctx, collectionName)
	}

	err = qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     dimensions,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	// Process routes
	var points []*qdrant.PointStruct
	totalUtterances := 0

	for _, route := range config.Routes {
		log.Printf("Processing route '%s' with %d utterances", route.Name, len(route.Utterances))

		// Batch embed utterances
		batchResp, err := embedClient.EmbedBatch(ctx, &pb.EmbedBatchRequest{
			Texts: route.Utterances,
		})
		if err != nil {
			log.Fatalf("Failed to embed utterances for route %s: %v", route.Name, err)
		}

		for i, utterance := range route.Utterances {
			vector := batchResp.Vectors[i].Values

			payload := map[string]*qdrant.Value{
				"route":     qdrant.NewValue(route.Name),
				"model":     qdrant.NewValue(route.Model),
				"provider":  qdrant.NewValue(route.Provider),
				"utterance": qdrant.NewValue(utterance),
			}

			// Add metadata
			for k, v := range route.Metadata {
				payload[k] = qdrant.NewValue(v)
			}

			points = append(points, &qdrant.PointStruct{
				Id:      qdrant.NewID(uuid.New().String()),
				Vectors: qdrant.NewVectors(vector...),
				Payload: payload,
			})
			totalUtterances++
		}
	}

	// Upsert points in batches
	log.Printf("Upserting %d points to Qdrant", len(points))
	batchSize := 100
	for i := 0; i < len(points); i += batchSize {
		end := i + batchSize
		if end > len(points) {
			end = len(points)
		}

		_, err := qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: collectionName,
			Points:         points[i:end],
		})
		if err != nil {
			log.Fatalf("Failed to upsert points: %v", err)
		}
	}

	// Verify
	time.Sleep(500 * time.Millisecond) // Let Qdrant index
	info, err := qdrantClient.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		log.Fatalf("Failed to get collection info: %v", err)
	}

	log.Printf("Collection '%s' now has %d points", collectionName, info.PointsCount)
	log.Printf("Seeded %d utterances across %d routes", totalUtterances, len(config.Routes))
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
