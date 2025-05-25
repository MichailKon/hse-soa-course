package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"social-network/common/proto"
	"strconv"
	"time"
)

type StatisticsHandler struct {
	client proto.StatisticsServiceClient
}

func NewStatisticsHandler(client proto.StatisticsServiceClient) *StatisticsHandler {
	return &StatisticsHandler{client: client}
}

func (h *StatisticsHandler) GetPostStats(c *gin.Context) {
	postIDStr := c.Param("id")
	if postIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post ID is required"})
		return
	}

	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := h.client.GetPostStats(ctx, &proto.GetPostStatsRequest{PostId: postID})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"post_id":        stats.PostId,
		"views_count":    stats.ViewsCount,
		"likes_count":    stats.LikesCount,
		"comments_count": stats.CommentsCount,
	})
}

func (h *StatisticsHandler) GetPostViewsTimeline(c *gin.Context) {
	postIDStr := c.Param("id")
	if postIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post ID is required"})
		return
	}

	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	timeline, err := h.client.GetPostViewsTimeline(ctx, &proto.GetPostTimelineRequest{
		PostId:    postID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response := formatTimelineResponse(timeline)
	c.JSON(http.StatusOK, response)
}

func (h *StatisticsHandler) GetPostLikesTimeline(c *gin.Context) {
	postIDStr := c.Param("id")
	if postIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post ID is required"})
		return
	}

	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	timeline, err := h.client.GetPostLikesTimeline(ctx, &proto.GetPostTimelineRequest{
		PostId:    postID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response := formatTimelineResponse(timeline)
	c.JSON(http.StatusOK, response)
}

func (h *StatisticsHandler) GetPostCommentsTimeline(c *gin.Context) {
	postIDStr := c.Param("id")
	if postIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post ID is required"})
		return
	}

	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	timeline, err := h.client.GetPostCommentsTimeline(ctx, &proto.GetPostTimelineRequest{
		PostId:    postID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response := formatTimelineResponse(timeline)
	c.JSON(http.StatusOK, response)
}

func (h *StatisticsHandler) GetTopPosts(c *gin.Context) {
	statTypeStr := c.Query("stat_type")
	if statTypeStr == "" {
		statTypeStr = "views"
	}

	var statType proto.StatType
	switch statTypeStr {
	case "views":
		statType = proto.StatType_VIEWS
	case "likes":
		statType = proto.StatType_LIKES
	case "comments":
		statType = proto.StatType_COMMENTS
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stat_type. Must be one of: views, likes, comments"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topPosts, err := h.client.GetTopPosts(ctx, &proto.GetTopPostsRequest{
		StatType: statType,
		Limit:    int32(limit),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response := make([]map[string]interface{}, len(topPosts.Posts))
	for i, post := range topPosts.Posts {
		response[i] = map[string]interface{}{
			"post_id": post.PostId,
			"title":   post.Title,
			"count":   post.Count,
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": response})
}

func (h *StatisticsHandler) GetTopUsers(c *gin.Context) {
	statTypeStr := c.Query("stat_type")
	if statTypeStr == "" {
		statTypeStr = "views"
	}

	var statType proto.StatType
	switch statTypeStr {
	case "views":
		statType = proto.StatType_VIEWS
	case "likes":
		statType = proto.StatType_LIKES
	case "comments":
		statType = proto.StatType_COMMENTS
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stat_type. Must be one of: views, likes, comments"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topUsers, err := h.client.GetTopUsers(ctx, &proto.GetTopUsersRequest{
		StatType: statType,
		Limit:    int32(limit),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response := make([]map[string]interface{}, len(topUsers.Users))
	for i, user := range topUsers.Users {
		response[i] = map[string]interface{}{
			"user_id":  user.UserId,
			"username": user.Username,
			"count":    user.Count,
		}
	}

	c.JSON(http.StatusOK, gin.H{"users": response})
}

func formatTimelineResponse(timeline *proto.GetTimelineResponse) gin.H {
	points := make([]map[string]interface{}, len(timeline.Points))
	for i, point := range timeline.Points {
		points[i] = map[string]interface{}{
			"date":  point.Date,
			"count": point.Count,
		}
	}

	return gin.H{
		"post_id": timeline.PostId,
		"points":  points,
	}
}
