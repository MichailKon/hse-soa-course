package handlers

import (
	"context"
	"social-network/common/proto"
	"social-network/statistics-service/models"
	"social-network/statistics-service/repositories"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StatisticsHandler struct {
	proto.UnimplementedStatisticsServiceServer
	repo repositories.IRepository
}

func NewStatisticsHandler(repo repositories.IRepository) *StatisticsHandler {
	return &StatisticsHandler{repo: repo}
}

func (h *StatisticsHandler) GetPostStats(ctx context.Context, req *proto.GetPostStatsRequest) (*proto.GetPostStatsResponse, error) {
	stats, err := h.repo.GetPostStats(uint(req.PostId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get post stats: %v", err)
	}

	return &proto.GetPostStatsResponse{
		PostId:        req.PostId,
		ViewsCount:    int32(stats.ViewsCount),
		LikesCount:    int32(stats.LikesCount),
		CommentsCount: int32(stats.CommentsCount),
	}, nil
}

func (h *StatisticsHandler) GetPostViewsTimeline(ctx context.Context, req *proto.GetPostTimelineRequest) (*proto.GetTimelineResponse, error) {
	startDate, endDate, err := parseTimelineRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid date range: %v", err)
	}

	timeline, err := h.repo.GetPostTimeline(uint(req.PostId), models.EventTypeView, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get views timeline: %v", err)
	}

	return buildTimelineResponse(req.PostId, timeline), nil
}

func (h *StatisticsHandler) GetPostLikesTimeline(ctx context.Context, req *proto.GetPostTimelineRequest) (*proto.GetTimelineResponse, error) {
	startDate, endDate, err := parseTimelineRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid date range: %v", err)
	}

	timeline, err := h.repo.GetPostTimeline(uint(req.PostId), models.EventTypeLike, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get likes timeline: %v", err)
	}

	return buildTimelineResponse(req.PostId, timeline), nil
}

func (h *StatisticsHandler) GetPostCommentsTimeline(ctx context.Context, req *proto.GetPostTimelineRequest) (*proto.GetTimelineResponse, error) {
	startDate, endDate, err := parseTimelineRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid date range: %v", err)
	}

	timeline, err := h.repo.GetPostTimeline(uint(req.PostId), models.EventTypeComment, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get comments timeline: %v", err)
	}

	return buildTimelineResponse(req.PostId, timeline), nil
}

func (h *StatisticsHandler) GetTopPosts(ctx context.Context, req *proto.GetTopPostsRequest) (*proto.GetTopPostsResponse, error) {
	eventType, err := convertStatType(req.StatType)
	if err != nil {
		return nil, err
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	topPosts, err := h.repo.GetTopPosts(eventType, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get top posts: %v", err)
	}

	result := &proto.GetTopPostsResponse{
		Posts: make([]*proto.TopPost, len(topPosts)),
	}

	for i, post := range topPosts {
		result.Posts[i] = &proto.TopPost{
			PostId: post.ID,
			Title:  post.Title,
			Count:  int32(post.Count),
		}
	}

	return result, nil
}

func (h *StatisticsHandler) GetTopUsers(ctx context.Context, req *proto.GetTopUsersRequest) (*proto.GetTopUsersResponse, error) {
	eventType, err := convertStatType(req.StatType)
	if err != nil {
		return nil, err
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	topUsers, err := h.repo.GetTopUsers(eventType, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get top users: %v", err)
	}

	result := &proto.GetTopUsersResponse{
		Users: make([]*proto.TopUser, len(topUsers)),
	}

	for i, user := range topUsers {
		result.Users[i] = &proto.TopUser{
			UserId:   int64(user.ID),
			Username: user.Title,
			Count:    int32(user.Count),
		}
	}

	return result, nil
}

func parseTimelineRange(startDateStr, endDateStr string) (time.Time, time.Time, error) {
	var startDate, endDate time.Time
	var err error

	if startDateStr == "" {
		startDate = time.Unix(0, 0)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if endDateStr == "" {
		endDate = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return startDate, endDate, nil
}

func buildTimelineResponse(postID uint64, timeline []models.TimelinePoint) *proto.GetTimelineResponse {
	response := &proto.GetTimelineResponse{
		PostId: postID,
		Points: make([]*proto.TimelinePoint, len(timeline)),
	}

	for i, point := range timeline {
		response.Points[i] = &proto.TimelinePoint{
			Date:  point.Date.Format("2006-01-02"),
			Count: int32(point.Count),
		}
	}

	return response
}

func convertStatType(statType proto.StatType) (models.EventType, error) {
	switch statType {
	case proto.StatType_VIEWS:
		return models.EventTypeView, nil
	case proto.StatType_LIKES:
		return models.EventTypeLike, nil
	case proto.StatType_COMMENTS:
		return models.EventTypeComment, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "Unknown stat type: %v", statType)
	}
}
