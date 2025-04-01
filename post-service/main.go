package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"post-service/handlers"
	"post-service/models"
	"post-service/proto"
	"post-service/repositories"

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

	var count int64
	if err = db.Model(&models.Post{}).Count(&count).Error; err != nil {
		log.Fatalf("Failed to count posts: %v", err)
	}
	log.Printf("Current posts: %v", count)

	repo := repository.NewPostRepository(db)
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
