package handlers

import (
	"context"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"post-service/models"
	"post-service/proto"
	"post-service/repositories"
	"time"
)

type PostHandler struct {
	repo *repository.PostRepository
	proto.UnimplementedPostServiceServer
}

func NewPostHandler(repo *repository.PostRepository) *PostHandler {
	return &PostHandler{repo: repo}
}

func convertPostToProto(post *models.Post) *proto.Post {
	protoPost := &proto.Post{
		Id:          post.ID,
		Title:       post.Title,
		Description: post.Description,
		CreatorId:   post.CreatorID,
		CreatedAt:   timestamppb.New(post.CreatedAt),
		UpdatedAt:   timestamppb.New(post.UpdatedAt),
		IsPrivate:   post.IsPrivate,
		Tags:        make([]string, len(post.Tags)),
	}
	for i, tag := range post.Tags {
		protoPost.Tags[i] = tag.Name
	}
	return protoPost
}

func (h *PostHandler) CreatePost(ctx context.Context, req *proto.CreatePostRequest) (*proto.Post, error) {
	log.Printf("Post Service: Received request to create post: %v", req)
	if req.Title == "" {
		log.Printf("Post Service: Title empty")
		return nil, status.Errorf(codes.InvalidArgument, "Post title is required")
	}
	var tags []models.Tag
	for _, tagName := range req.Tags {
		tags = append(tags, models.Tag{Name: tagName})
	}
	post := &models.Post{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		CreatorID:   req.CreatorId,
		IsPrivate:   req.IsPrivate,
		Tags:        tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	log.Printf("Post Service: Creating post")
	if err := h.repo.CreatePost(post); err != nil {
		log.Printf("Post Service: error: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to create post: %v", err)
	}
	log.Printf("Post Service: OK, id=%v", post.ID)
	return convertPostToProto(post), nil
}

func (h *PostHandler) GetPost(ctx context.Context, req *proto.GetPostRequest) (*proto.Post, error) {
	post, err := h.repo.GetPostByID(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if post == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	if post.IsPrivate && post.CreatorID != req.RequesterId {
		return nil, status.Errorf(codes.PermissionDenied, "You don't have permission to view this post")
	}
	return convertPostToProto(post), nil
}

func (h *PostHandler) UpdatePost(ctx context.Context, req *proto.UpdatePostRequest) (*proto.Post, error) {
	existingPost, err := h.repo.GetPostByID(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if existingPost == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	if existingPost.CreatorID != req.UpdaterId {
		return nil, status.Errorf(codes.PermissionDenied, "You don't have permission to update this post")
	}
	if req.Title != "" {
		existingPost.Title = req.Title
	}
	if req.Description != "" {
		existingPost.Description = req.Description
	}
	var tags []models.Tag
	for _, tagName := range req.Tags {
		tags = append(tags, models.Tag{Name: tagName})
	}
	existingPost.IsPrivate = req.IsPrivate
	existingPost.Tags = tags
	existingPost.UpdatedAt = time.Now()
	if err = h.repo.UpdatePost(existingPost, req.Tags); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update post: %v", err)
	}
	return convertPostToProto(existingPost), nil
}

func (h *PostHandler) DeletePost(ctx context.Context, req *proto.DeletePostRequest) (*proto.DeletePostResponse, error) {
	log.Printf("Post Service: Delete post: %v", req.String())
	existingPost, err := h.repo.GetPostByID(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if existingPost == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	if existingPost.CreatorID != req.DeleterId {
		return nil, status.Errorf(codes.PermissionDenied, "You don't have permission to delete this post")
	}
	if err = h.repo.DeletePost(req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete post: %v", err)
	}
	return &proto.DeletePostResponse{Success: true}, nil
}

func (h *PostHandler) ListPosts(ctx context.Context, req *proto.ListPostsRequest) (*proto.ListPostsResponse, error) {
	log.Printf("Post Service: List Posts: %v", req.String())
	page := int(req.Page)
	if page < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "Page must be greater than 0")
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "Page size must be greater than 0")
	}
	includePrivate := req.RequesterId == req.CreatorId && req.CreatorId != ""
	posts, totalCount, err := h.repo.ListPosts(page, pageSize, req.CreatorId, req.Tags, includePrivate, req.RequesterId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list posts: %v", err)
	}
	protoPostsList := make([]*proto.Post, len(posts))
	for i, post := range posts {
		protoPostsList[i] = convertPostToProto(&post)
	}
	totalPages := int32((totalCount + int64(pageSize) - 1) / int64(pageSize))
	return &proto.ListPostsResponse{
		Posts:      protoPostsList,
		TotalCount: int32(totalCount),
		TotalPages: totalPages,
	}, nil
}
