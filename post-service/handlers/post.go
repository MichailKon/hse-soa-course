package handlers

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"social-network/common/kafka"
	"social-network/common/proto"
	"social-network/post-service/models"
	"social-network/post-service/repositories"
	"time"
)

type PostHandler struct {
	proto.UnimplementedPostServiceServer
	postRepo      *repositories.PostRepository
	commentRepo   *repositories.CommentRepository
	viewRepo      *repositories.ViewRepository
	likeRepo      *repositories.LikeRepository
	kafkaProducer *kafka.Producer
}

func NewPostHandler(
	postRepo *repositories.PostRepository,
	commentRepo *repositories.CommentRepository,
	viewRepo *repositories.ViewRepository,
	likeRepo *repositories.LikeRepository,
	producer *kafka.Producer,
) *PostHandler {
	return &PostHandler{
		postRepo:      postRepo,
		commentRepo:   commentRepo,
		viewRepo:      viewRepo,
		likeRepo:      likeRepo,
		kafkaProducer: producer,
	}
}

func convertPostToProto(post *models.Post) *proto.Post {
	protoPost := &proto.Post{
		Title:       post.Title,
		Description: post.Description,
		CreatorId:   post.CreatorID,
		IsPrivate:   post.IsPrivate,
		Tags:        make([]string, len(post.Tags)),
	}
	protoPost.Id = uint64(post.ID)
	protoPost.CreatedAt = timestamppb.New(post.CreatedAt)
	protoPost.UpdatedAt = timestamppb.New(post.UpdatedAt)
	for i, tag := range post.Tags {
		protoPost.Tags[i] = tag.Name
	}
	return protoPost
}

func (h *PostHandler) CreatePost(ctx context.Context, req *proto.CreatePostRequest) (*proto.Post, error) {
	if req.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Post title is required")
	}
	if req.CreatorId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Post creatorId is required")
	}
	var tags []models.Tag
	for _, tagName := range req.Tags {
		tags = append(tags, models.Tag{Name: tagName})
	}
	post := &models.Post{
		Title:       req.Title,
		Description: req.Description,
		CreatorID:   req.CreatorId,
		IsPrivate:   req.IsPrivate,
		Tags:        tags,
	}
	if err := h.postRepo.CreatePost(ctx, post); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create post: %v", err)
	}
	return convertPostToProto(post), nil
}

func (h *PostHandler) GetPost(ctx context.Context, req *proto.GetPostRequest) (*proto.Post, error) {
	post, err := h.postRepo.GetPostByID(ctx, req.Id)
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
	existingPost, err := h.postRepo.GetPostByID(ctx, req.Id)
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
	if err = h.postRepo.UpdatePost(ctx, existingPost, req.Tags); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update post: %v", err)
	}
	return convertPostToProto(existingPost), nil
}

func (h *PostHandler) DeletePost(ctx context.Context, req *proto.DeletePostRequest) (*proto.DeletePostResponse, error) {
	existingPost, err := h.postRepo.GetPostByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if existingPost == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	if existingPost.CreatorID != req.DeleterId {
		return nil, status.Errorf(codes.PermissionDenied, "You don't have permission to delete this post")
	}
	if err = h.postRepo.DeletePost(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete post: %v", err)
	}
	return &proto.DeletePostResponse{Success: true}, nil
}

func (h *PostHandler) ListPosts(ctx context.Context, req *proto.ListPostsRequest) (*proto.ListPostsResponse, error) {
	page := int(req.Page)
	if page < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "Page must be greater than 0")
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "Page size must be greater than 0")
	}
	includePrivate := req.RequesterId == req.CreatorId && req.CreatorId != ""
	posts, totalCount, err :=
		h.postRepo.ListPosts(ctx, page, pageSize, req.CreatorId, req.Tags, includePrivate, req.RequesterId)
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
