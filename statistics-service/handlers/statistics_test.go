package handlers

import (
	"context"
	"social-network/common/proto"
	"social-network/statistics-service/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockClickHouseRepository struct {
	mock.Mock
}

func (m *MockClickHouseRepository) GetPostStats(postID uint) (*models.PostStat, error) {
	args := m.Called(postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PostStat), args.Error(1)
}

func (m *MockClickHouseRepository) GetPostTimeline(postID uint, eventType models.EventType, startDate, endDate time.Time) ([]models.TimelinePoint, error) {
	args := m.Called(postID, eventType, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TimelinePoint), args.Error(1)
}

func (m *MockClickHouseRepository) GetTopPosts(eventType models.EventType, limit int) ([]models.TopItem, error) {
	args := m.Called(eventType, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TopItem), args.Error(1)
}

func (m *MockClickHouseRepository) GetTopUsers(eventType models.EventType, limit int) ([]models.TopItem, error) {
	args := m.Called(eventType, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TopItem), args.Error(1)
}

func (m *MockClickHouseRepository) SaveEvent(event *models.Event) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *MockClickHouseRepository) TestConnection() error {
	args := m.Called()
	return args.Error(0)
}

func TestGetPostStats(t *testing.T) {
	mockRepo := new(MockClickHouseRepository)
	handler := NewStatisticsHandler(mockRepo)
	ctx := context.Background()

	t.Run("Successful stats retrieval", func(t *testing.T) {
		mockStat := &models.PostStat{
			PostID:        1,
			ViewsCount:    100,
			LikesCount:    50,
			CommentsCount: 25,
		}
		mockRepo.On("GetPostStats", uint(1)).Return(mockStat, nil).Once()

		req := &proto.GetPostStatsRequest{PostId: 1}
		resp, err := handler.GetPostStats(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, uint64(1), resp.PostId)
		assert.Equal(t, int32(100), resp.ViewsCount)
		assert.Equal(t, int32(50), resp.LikesCount)
		assert.Equal(t, int32(25), resp.CommentsCount)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error getting stats", func(t *testing.T) {
		mockError := status.Error(codes.Internal, "database error")
		mockRepo.On("GetPostStats", uint(2)).Return(nil, mockError).Once()

		req := &proto.GetPostStatsRequest{PostId: 2}
		resp, err := handler.GetPostStats(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "Failed to get post stats")

		mockRepo.AssertExpectations(t)
	})
}

func TestGetPostViewsTimeline(t *testing.T) {
	mockRepo := new(MockClickHouseRepository)
	handler := NewStatisticsHandler(mockRepo)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour).UTC()
	yesterday := today.AddDate(0, 0, -1).UTC()
	twoDaysAgo := today.AddDate(0, 0, -2).UTC()

	t.Run("Successful views timeline retrieval", func(t *testing.T) {
		mockTimeline := []models.TimelinePoint{
			{Date: twoDaysAgo, Count: 10},
			{Date: yesterday, Count: 20},
			{Date: today, Count: 30},
		}

		startDate := twoDaysAgo
		endDate := today

		mockRepo.On("GetPostTimeline", uint(1), models.EventTypeView, startDate, endDate).
			Return(mockTimeline, nil).Once()

		req := &proto.GetPostTimelineRequest{
			PostId:    1,
			StartDate: twoDaysAgo.Format("2006-01-02"),
			EndDate:   today.Format("2006-01-02"),
		}
		resp, err := handler.GetPostViewsTimeline(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, uint64(1), resp.PostId)
		assert.Len(t, resp.Points, 3)
		assert.Equal(t, twoDaysAgo.Format("2006-01-02"), resp.Points[0].Date)
		assert.Equal(t, int32(10), resp.Points[0].Count)
		assert.Equal(t, today.Format("2006-01-02"), resp.Points[2].Date)
		assert.Equal(t, int32(30), resp.Points[2].Count)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid date format", func(t *testing.T) {
		req := &proto.GetPostTimelineRequest{
			PostId:    1,
			StartDate: "invalid-date",
			EndDate:   today.Format("2006-01-02"),
		}
		resp, err := handler.GetPostViewsTimeline(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "Invalid date range")
	})

	t.Run("Error getting timeline", func(t *testing.T) {
		mockError := status.Error(codes.Internal, "database error")

		startDate := twoDaysAgo
		endDate := today

		mockRepo.On("GetPostTimeline", uint(2), models.EventTypeView, startDate, endDate).
			Return(nil, mockError).Once()

		req := &proto.GetPostTimelineRequest{
			PostId:    2,
			StartDate: twoDaysAgo.Format("2006-01-02"),
			EndDate:   today.Format("2006-01-02"),
		}
		resp, err := handler.GetPostViewsTimeline(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "Failed to get views timeline")

		mockRepo.AssertExpectations(t)
	})
}

func TestGetPostLikesTimeline(t *testing.T) {
	mockRepo := new(MockClickHouseRepository)
	handler := NewStatisticsHandler(mockRepo)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour).UTC()
	yesterday := today.AddDate(0, 0, -1).UTC()

	t.Run("Successful likes timeline retrieval", func(t *testing.T) {
		mockTimeline := []models.TimelinePoint{
			{Date: yesterday, Count: 5},
			{Date: today, Count: 10},
		}

		startDate := yesterday
		endDate := today

		mockRepo.On("GetPostTimeline", uint(1), models.EventTypeLike, startDate, endDate).
			Return(mockTimeline, nil).Once()

		req := &proto.GetPostTimelineRequest{
			PostId:    1,
			StartDate: yesterday.Format("2006-01-02"),
			EndDate:   today.Format("2006-01-02"),
		}
		resp, err := handler.GetPostLikesTimeline(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, uint64(1), resp.PostId)
		assert.Len(t, resp.Points, 2)
		assert.Equal(t, yesterday.Format("2006-01-02"), resp.Points[0].Date)
		assert.Equal(t, int32(5), resp.Points[0].Count)
		assert.Equal(t, today.Format("2006-01-02"), resp.Points[1].Date)
		assert.Equal(t, int32(10), resp.Points[1].Count)

		mockRepo.AssertExpectations(t)
	})
}

func TestGetPostCommentsTimeline(t *testing.T) {
	mockRepo := new(MockClickHouseRepository)
	handler := NewStatisticsHandler(mockRepo)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour).UTC()
	yesterday := today.AddDate(0, 0, -1).UTC()

	t.Run("Successful comments timeline retrieval", func(t *testing.T) {
		mockTimeline := []models.TimelinePoint{
			{Date: yesterday, Count: 3},
			{Date: today, Count: 7},
		}

		startDate := yesterday
		endDate := today

		mockRepo.On("GetPostTimeline", uint(1), models.EventTypeComment, startDate, endDate).
			Return(mockTimeline, nil).Once()

		req := &proto.GetPostTimelineRequest{
			PostId:    1,
			StartDate: yesterday.Format("2006-01-02"),
			EndDate:   today.Format("2006-01-02"),
		}
		resp, err := handler.GetPostCommentsTimeline(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, uint64(1), resp.PostId)
		assert.Len(t, resp.Points, 2)
		assert.Equal(t, yesterday.Format("2006-01-02"), resp.Points[0].Date)
		assert.Equal(t, int32(3), resp.Points[0].Count)
		assert.Equal(t, today.Format("2006-01-02"), resp.Points[1].Date)
		assert.Equal(t, int32(7), resp.Points[1].Count)

		mockRepo.AssertExpectations(t)
	})
}

func TestGetTopPosts(t *testing.T) {
	mockRepo := new(MockClickHouseRepository)
	handler := NewStatisticsHandler(mockRepo)
	ctx := context.Background()

	t.Run("Successful top posts retrieval", func(t *testing.T) {
		mockTopPosts := []models.TopItem{
			{ID: 1, Title: "Post 1", Count: 100},
			{ID: 2, Title: "Post 2", Count: 50},
			{ID: 3, Title: "Post 3", Count: 25},
		}

		mockRepo.On("GetTopPosts", models.EventTypeView, 3).
			Return(mockTopPosts, nil).Once()

		req := &proto.GetTopPostsRequest{
			StatType: proto.StatType_VIEWS,
			Limit:    3,
		}
		resp, err := handler.GetTopPosts(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Posts, 3)
		assert.Equal(t, uint64(1), resp.Posts[0].PostId)
		assert.Equal(t, "Post 1", resp.Posts[0].Title)
		assert.Equal(t, int32(100), resp.Posts[0].Count)
		assert.Equal(t, uint64(3), resp.Posts[2].PostId)
		assert.Equal(t, int32(25), resp.Posts[2].Count)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid stat type", func(t *testing.T) {
		req := &proto.GetTopPostsRequest{
			StatType: 999,
			Limit:    3,
		}
		resp, err := handler.GetTopPosts(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "Unknown stat type")
	})

	t.Run("Error getting top posts", func(t *testing.T) {
		mockError := status.Error(codes.Internal, "database error")
		mockRepo.On("GetTopPosts", models.EventTypeLike, 5).
			Return(nil, mockError).Once()

		req := &proto.GetTopPostsRequest{
			StatType: proto.StatType_LIKES,
			Limit:    5,
		}
		resp, err := handler.GetTopPosts(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "Failed to get top posts")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Negative limit", func(t *testing.T) {
		mockTopPosts := []models.TopItem{
			{ID: 1, Title: "Post 1", Count: 100},
			{ID: 2, Title: "Post 2", Count: 50},
		}

		mockRepo.On("GetTopPosts", models.EventTypeView, 10).
			Return(mockTopPosts, nil).Once()

		req := &proto.GetTopPostsRequest{
			StatType: proto.StatType_VIEWS,
			Limit:    -1,
		}
		resp, err := handler.GetTopPosts(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Posts, 2)

		mockRepo.AssertExpectations(t)
	})
}

func TestGetTopUsers(t *testing.T) {
	mockRepo := new(MockClickHouseRepository)
	handler := NewStatisticsHandler(mockRepo)
	ctx := context.Background()

	t.Run("Successful top users retrieval", func(t *testing.T) {
		mockTopUsers := []models.TopItem{
			{ID: 1, Title: "user1", Count: 100},
			{ID: 2, Title: "user2", Count: 50},
			{ID: 3, Title: "user3", Count: 25},
		}

		mockRepo.On("GetTopUsers", models.EventTypeComment, 3).
			Return(mockTopUsers, nil).Once()

		req := &proto.GetTopUsersRequest{
			StatType: proto.StatType_COMMENTS,
			Limit:    3,
		}
		resp, err := handler.GetTopUsers(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Users, 3)
		assert.Equal(t, int64(1), resp.Users[0].UserId)
		assert.Equal(t, "user1", resp.Users[0].Username)
		assert.Equal(t, int32(100), resp.Users[0].Count)
		assert.Equal(t, int64(3), resp.Users[2].UserId)
		assert.Equal(t, int32(25), resp.Users[2].Count)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid stat type", func(t *testing.T) {
		req := &proto.GetTopUsersRequest{
			StatType: 999,
			Limit:    3,
		}
		resp, err := handler.GetTopUsers(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "Unknown stat type")
	})

	t.Run("Error getting top users", func(t *testing.T) {
		mockError := status.Error(codes.Internal, "database error")
		mockRepo.On("GetTopUsers", models.EventTypeLike, 5).
			Return(nil, mockError).Once()

		req := &proto.GetTopUsersRequest{
			StatType: proto.StatType_LIKES,
			Limit:    5,
		}
		resp, err := handler.GetTopUsers(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "Failed to get top users")

		mockRepo.AssertExpectations(t)
	})
}

func TestParseTimelineRange(t *testing.T) {
	t.Run("Valid date range", func(t *testing.T) {
		startDateStr := "2023-01-01"
		endDateStr := "2023-01-10"

		startDate, endDate, err := parseTimelineRange(startDateStr, endDateStr)

		assert.NoError(t, err)
		assert.Equal(t, "2023-01-01", startDate.Format("2006-01-02"))
		assert.Equal(t, "2023-01-10", endDate.Format("2006-01-02"))
	})

	t.Run("Empty date strings", func(t *testing.T) {
		startDate, endDate, err := parseTimelineRange("", "")

		assert.NoError(t, err)

		expectedStartDate := time.Unix(0, 0).UTC()
		assert.Equal(t, expectedStartDate.Format("2006-01-02"), startDate.Format("2006-01-02"))

		today := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, today.Format("2006-01-02"), endDate.Format("2006-01-02"))
	})

	t.Run("Invalid date format", func(t *testing.T) {
		startDateStr := "invalid-date"
		endDateStr := "2023-01-10"

		_, _, err := parseTimelineRange(startDateStr, endDateStr)

		assert.Error(t, err)
	})
}

func TestConvertStatType(t *testing.T) {
	t.Run("Valid stat types", func(t *testing.T) {
		viewsType, err := convertStatType(proto.StatType_VIEWS)
		assert.NoError(t, err)
		assert.Equal(t, models.EventTypeView, viewsType)

		likesType, err := convertStatType(proto.StatType_LIKES)
		assert.NoError(t, err)
		assert.Equal(t, models.EventTypeLike, likesType)

		commentsType, err := convertStatType(proto.StatType_COMMENTS)
		assert.NoError(t, err)
		assert.Equal(t, models.EventTypeComment, commentsType)
	})

	t.Run("Invalid stat type", func(t *testing.T) {
		_, err := convertStatType(999)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "Unknown stat type")
	})
}
