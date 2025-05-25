package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"social-network/common/proto"
	"social-network/statistics-service/handlers"
	"social-network/statistics-service/kafka"
	"social-network/statistics-service/repositories"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	chHost := getEnv("CLICKHOUSE_HOST", "clickhouse")
	chPort := getEnv("CLICKHOUSE_PORT", "9000")
	chDatabase := getEnv("CLICKHOUSE_DATABASE", "statistics")
	chUser := getEnv("CLICKHOUSE_USER", "default")
	chPassword := getEnv("CLICKHOUSE_PASSWORD", "password")

	chRepo, err := repositories.NewClickHouseRepository(chHost, chPort, chDatabase, chUser, chPassword)
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	kafkaServers := getEnv("KAFKA_BOOTSTRAP_SERVERS", "kafka:9092")
	consumer := kafka.NewConsumer(kafkaServers, chRepo)
	handler := handlers.NewStatisticsHandler(chRepo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer.ConsumeEvents(ctx)
	port := getEnv("GRPC_PORT", "50052")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterStatisticsServiceServer(s, handler)
	reflection.Register(s)
	log.Printf("Statistics service gRPC server listening on port %s", port)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Println("Shutting down gRPC server...")
		s.GracefulStop()
		cancel()
		consumer.Close()
		log.Println("Server stopped")
	}()
	if err = s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
