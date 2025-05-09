package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"social-network/common/proto"
	"social-network/post-service/handlers"
	"social-network/post-service/models"
	"social-network/post-service/repositories"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=%s sslmode=disable password=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err = db.AutoMigrate(&models.Post{}, &models.Tag{}, &models.PostTag{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	repo := repositories.NewPostRepository(db)
	handler := handlers.NewPostHandler(repo)
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterPostServiceServer(s, handler)
	reflection.Register(s)
	log.Printf("Post service gRPC server listening on port %s", port)
	if err = s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
