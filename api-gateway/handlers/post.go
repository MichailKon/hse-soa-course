package handlers

import (
	"api-gateway/models"
	"api-gateway/proto"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PostHandler struct {
	client proto.PostServiceClient
}

func NewPostHandler(client proto.PostServiceClient) *PostHandler {
	return &PostHandler{client: client}
}

func convertProtoToPost(p *proto.Post) models.Post {
	post := models.Post{
		Title:       p.Title,
		Description: p.Description,
		CreatorID:   p.CreatorId,
		IsPrivate:   p.IsPrivate,
		Tags:        p.Tags,
	}
	post.ID = uint(p.Id)
	post.CreatedAt = p.CreatedAt.AsTime()
	post.UpdatedAt = p.UpdatedAt.AsTime()
	return post
}

func handleGRPCError(c *gin.Context, err error) {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
		case codes.InvalidArgument:
			c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
		case codes.PermissionDenied:
			c.JSON(http.StatusFound, gin.H{"error": st.Message()})
		case codes.Unauthenticated:
			c.JSON(http.StatusUnauthorized, gin.H{"error": st.Message()})
		default:
			c.JSON(http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("Unknown error %v; error: %v", st.Code(), st.Message())})
		}
	}
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	var req models.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "userId is required"})
		return
	}
	grpcReq := &proto.CreatePostRequest{
		Title:       req.Title,
		Description: req.Description,
		CreatorId:   fmt.Sprint(userId.(int)),
		IsPrivate:   req.IsPrivate,
		Tags:        req.Tags,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	post, err := h.client.CreatePost(ctx, grpcReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, convertProtoToPost(post))
}

func (h *PostHandler) GetPost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is required"})
		return
	}
	var intId uint64
	var err error
	if intId, err = strconv.ParseUint(id, 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	grpcReq := &proto.GetPostRequest{
		Id:          intId,
		RequesterId: strconv.Itoa(userId.(int)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	post, err := h.client.GetPost(ctx, grpcReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, convertProtoToPost(post))
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is required"})
		return
	}
	var req models.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var intId uint64
	var err error
	if intId, err = strconv.ParseUint(id, 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	grpcReq := &proto.UpdatePostRequest{
		Id:          intId,
		Title:       req.Title,
		Description: req.Description,
		IsPrivate:   req.IsPrivate,
		Tags:        req.Tags,
		UpdaterId:   strconv.Itoa(userId.(int)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	post, err := h.client.UpdatePost(ctx, grpcReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, convertProtoToPost(post))
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is required"})
		return
	}
	var intId uint64
	var err error
	if intId, err = strconv.ParseUint(id, 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	grpcReq := &proto.DeletePostRequest{
		Id:        intId,
		DeleterId: strconv.Itoa(userId.(int)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := h.client.DeletePost(ctx, grpcReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}

func (h *PostHandler) ListPosts(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is required"})
		return
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	creatorId := c.Query("creatorId")
	tagsStr := c.Query("tags")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page is not provided or invalid"})
		return
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page size is not provided or invalid"})
		return
	}
	var tags []string
	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}
	grpcReq := &proto.ListPostsRequest{
		Page:        int32(page),
		PageSize:    int32(pageSize),
		RequesterId: strconv.Itoa(userId.(int)),
		CreatorId:   creatorId,
		Tags:        tags,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := h.client.ListPosts(ctx, grpcReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	posts := make([]models.Post, len(response.Posts))
	for i, post := range response.Posts {
		posts[i] = convertProtoToPost(post)
	}
	c.JSON(http.StatusOK, models.ListPostsResponse{
		Posts:      posts,
		TotalCount: response.TotalCount,
		TotalPages: response.TotalPages,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
}
