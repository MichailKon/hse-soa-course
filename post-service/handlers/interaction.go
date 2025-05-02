package handlers

import (
	"context"
	"fmt"
	"log"
	"social-network/common/kafka"
	"social-network/common/proto"
	"social-network/post-service/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *PostHandler) ViewPost(ctx context.Context, req *proto.ViewPostRequest) (*proto.ViewPostResponse, error) {
	post, err := h.postRepo.GetPostByID(nil, req.PostId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if post == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	if post.IsPrivate && post.CreatorID != fmt.Sprint(req.ViewerId) {
		return nil, status.Errorf(codes.PermissionDenied, "You don't have permission to view this post")
	}
	view := &models.View{
		PostID:   uint(req.PostId),
		ViewerID: req.ViewerId,
	}
	if err = h.viewRepo.RecordView(ctx, view); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to record view: %v", err)
	}
	event := kafka.NewEvent("post_viewed", fmt.Sprint(req.ViewerId), uint(req.PostId), nil)
	if err = h.kafkaProducer.SendEvent("post_views", event); err != nil {
		log.Printf("Failed to send post_viewed event to Kafka: %v", err)
	}
	return &proto.ViewPostResponse{
		Success: true,
		Post:    convertPostToProto(post),
	}, nil
}

func (h *PostHandler) LikePost(ctx context.Context, req *proto.LikePostRequest) (*proto.LikePostResponse, error) {
	post, err := h.postRepo.GetPostByID(ctx, req.PostId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if post == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	if post.IsPrivate && post.CreatorID != fmt.Sprint(req.LikerId) {
		return nil, status.Errorf(codes.PermissionDenied, "You don't have permission to like this post")
	}
	like := &models.Like{
		PostID:  uint(req.PostId),
		LikerID: req.LikerId,
	}
	added, err := h.likeRepo.ToggleLike(ctx, like)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to toggle like: %v", err)
	}
	likeCount, err := h.likeRepo.LikesCountByPostID(ctx, req.PostId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get like count: %v", err)
	}
	if added {
		event := kafka.NewEvent("post_liked", fmt.Sprint(req.LikerId), uint(req.PostId), nil)
		if err = h.kafkaProducer.SendEvent("post_likes", event); err != nil {
			log.Printf("Failed to send post_liked event to Kafka: %v", err)
		}
	}
	return &proto.LikePostResponse{
		Success:    true,
		TotalLikes: int32(likeCount),
	}, nil
}

func (h *PostHandler) CommentPost(ctx context.Context, req *proto.CommentPostRequest) (*proto.Comment, error) {
	post, err := h.postRepo.GetPostByID(ctx, req.PostId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if post == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	if post.IsPrivate && post.CreatorID != fmt.Sprint(req.AuthorId) {
		return nil, status.Errorf(codes.PermissionDenied, "You don't have permission to comment on this post")
	}
	if req.Content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Comment content cannot be empty")
	}
	comment := &models.Comment{
		PostID:   uint(req.PostId),
		AuthorID: req.AuthorId,
		Content:  req.Content,
	}
	if err = h.commentRepo.CreateComment(ctx, comment); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create comment: %v", err)
	}
	event := kafka.NewEvent(
		"post_commented",
		fmt.Sprint(req.AuthorId),
		uint(req.PostId),
		map[string]interface{}{
			"comment_id": comment.ID,
			"content":    req.Content,
		})
	if err = h.kafkaProducer.SendEvent("post_comments", event); err != nil {
		log.Printf("Failed to send post_commented event to Kafka: %v", err)
	}
	return &proto.Comment{
		Id:        uint64(comment.ID),
		PostId:    req.PostId,
		AuthorId:  comment.AuthorID,
		Content:   comment.Content,
		CreatedAt: timestamppb.New(comment.CreatedAt),
	}, nil
}

func (h *PostHandler) ListComments(ctx context.Context, req *proto.ListCommentsRequest) (*proto.ListCommentsResponse, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "Page must be greater than 0")
	}
	if pageSize < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "Page size must be greater than 0")
	}
	post, err := h.postRepo.GetPostByID(ctx, req.PostId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post: %v", err)
	}
	if post == nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}
	comments, count, err := h.commentRepo.GetCommentsForPostByID(ctx, int(req.PostId), page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get comments: %v", err)
	}
	protoComments := make([]*proto.Comment, len(comments))
	for i, comment := range comments {
		protoComments[i] = &proto.Comment{
			Id:        uint64(comment.ID),
			PostId:    uint64(comment.PostID),
			AuthorId:  comment.AuthorID,
			Content:   comment.Content,
			CreatedAt: timestamppb.New(comment.CreatedAt),
		}
	}
	totalPages := (count + int64(pageSize) - 1) / int64(pageSize)
	return &proto.ListCommentsResponse{
		Comments:   protoComments,
		TotalCount: int32(count),
		TotalPages: int32(totalPages),
	}, nil
}
