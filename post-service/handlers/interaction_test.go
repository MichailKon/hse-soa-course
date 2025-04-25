package handlers

import (
	"context"
	"social-network/common/kafka"
	"social-network/common/proto"
	"social-network/post-service/models"
	"social-network/post-service/repositories"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) SendEvent(topic string, event *kafka.Event) error {
	args := m.Called(topic, event)
	return args.Error(0)
}

func (m *MockKafkaProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func setupInteractionTestFixture(t *testing.T) (*PostHandler, *gorm.DB, *MockKafkaProducer) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&models.Post{}, &models.Tag{}, &models.PostTag{},
		&models.Comment{}, &models.Like{}, &models.View{})
	require.NoError(t, err)
	postRepo := repositories.NewPostRepository(db)
	commentRepo := repositories.NewCommentRepository(db)
	likeRepo := repositories.NewLikeRepository(db)
	viewRepo := repositories.NewViewRepository(db)
	kafkaProducer := new(MockKafkaProducer)
	handler := NewPostHandler(postRepo, commentRepo, viewRepo, likeRepo, kafkaProducer)

	return handler, db, kafkaProducer
}

func createTestPost(db *gorm.DB, isPrivate bool) (*models.Post, error) {
	post := &models.Post{
		Title:       "Test Post",
		Description: "Test Description",
		CreatorID:   "123",
		IsPrivate:   isPrivate,
		Tags:        []models.Tag{{Name: "test"}},
	}
	return post, db.Create(post).Error
}

func TestViewPost(t *testing.T) {
	t.Run("successfully view public post", func(t *testing.T) {
		handler, db, mockProducer := setupInteractionTestFixture(t)
		post, err := createTestPost(db, false)
		require.NoError(t, err)
		mockProducer.On("SendEvent", "post_views", mock.Anything).Return(nil)
		req := &proto.ViewPostRequest{
			PostId:   uint64(post.ID),
			ViewerId: 123,
		}
		resp, err := handler.ViewPost(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Post)
		assert.Equal(t, post.Title, resp.Post.Title)
		mockProducer.AssertExpectations(t)
		var count int64
		db.Model(&models.View{}).Where("post_id = ? AND viewer_id = ?", post.ID, req.ViewerId).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("cannot view private post", func(t *testing.T) {
		handler, db, _ := setupInteractionTestFixture(t)
		post, err := createTestPost(db, true)
		require.NoError(t, err)
		req := &proto.ViewPostRequest{
			PostId:   uint64(post.ID),
			ViewerId: 456,
		}
		_, err = handler.ViewPost(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("post not found", func(t *testing.T) {
		handler, _, _ := setupInteractionTestFixture(t)
		req := &proto.ViewPostRequest{
			PostId:   999,
			ViewerId: 123,
		}
		_, err := handler.ViewPost(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestLikePost(t *testing.T) {
	t.Run("add like to post", func(t *testing.T) {
		handler, db, mockProducer := setupInteractionTestFixture(t)
		post, err := createTestPost(db, false)
		require.NoError(t, err)
		mockProducer.On("SendEvent", "post_likes", mock.Anything).Return(nil)
		req := &proto.LikePostRequest{
			PostId:  uint64(post.ID),
			LikerId: 123,
		}
		resp, err := handler.LikePost(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(1), resp.TotalLikes)
		mockProducer.AssertExpectations(t)
		var count int64
		db.Model(&models.Like{}).Where("post_id = ? AND liker_id = ?", post.ID, req.LikerId).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("toggle like (remove existing like)", func(t *testing.T) {
		handler, db, _ := setupInteractionTestFixture(t)
		post, err := createTestPost(db, false)
		require.NoError(t, err)
		like := &models.Like{
			PostID:  post.ID,
			LikerID: 123,
		}
		require.NoError(t, db.Create(like).Error)
		req := &proto.LikePostRequest{
			PostId:  uint64(post.ID),
			LikerId: 123,
		}
		resp, err := handler.LikePost(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, int32(0), resp.TotalLikes) // Count should be 0 after removing the like

		// Verify like was removed
		var count int64
		db.Model(&models.Like{}).Where("post_id = ? AND liker_id = ?", post.ID, req.LikerId).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("cannot like private post", func(t *testing.T) {
		handler, db, _ := setupInteractionTestFixture(t)
		post, err := createTestPost(db, true)
		require.NoError(t, err)
		req := &proto.LikePostRequest{
			PostId:  uint64(post.ID),
			LikerId: 456,
		}
		_, err = handler.LikePost(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})
}

func TestCommentPost(t *testing.T) {
	t.Run("add comment to post", func(t *testing.T) {
		handler, db, mockProducer := setupInteractionTestFixture(t)
		post, err := createTestPost(db, false)
		require.NoError(t, err)
		mockProducer.On("SendEvent", "post_comments", mock.Anything).Return(nil)
		req := &proto.CommentPostRequest{
			PostId:   uint64(post.ID),
			AuthorId: 123,
			Content:  "Test comment",
		}
		resp, err := handler.CommentPost(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, req.Content, resp.Content)
		assert.Equal(t, req.AuthorId, resp.AuthorId)
		mockProducer.AssertExpectations(t)
		var count int64
		db.Model(&models.Comment{}).Where("post_id = ? AND author_id = ?", post.ID, req.AuthorId).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("empty comment content", func(t *testing.T) {
		handler, db, _ := setupInteractionTestFixture(t)
		post, err := createTestPost(db, false)
		require.NoError(t, err)
		req := &proto.CommentPostRequest{
			PostId:   uint64(post.ID),
			AuthorId: 123,
			Content:  "",
		}
		_, err = handler.CommentPost(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("cannot comment on private post", func(t *testing.T) {
		handler, db, _ := setupInteractionTestFixture(t)
		post, err := createTestPost(db, true)
		require.NoError(t, err)
		req := &proto.CommentPostRequest{
			PostId:   uint64(post.ID),
			AuthorId: 456,
			Content:  "Test comment",
		}
		_, err = handler.CommentPost(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})
}

func TestListComments(t *testing.T) {
	t.Run("list comments with pagination", func(t *testing.T) {
		handler, db, _ := setupInteractionTestFixture(t)
		post, err := createTestPost(db, false)
		require.NoError(t, err)
		for i := 0; i < 15; i++ {
			comment := &models.Comment{
				PostID:   post.ID,
				AuthorID: 123,
				Content:  "Comment " + string(rune(i+48)),
			}
			require.NoError(t, db.Create(comment).Error)
		}
		req := &proto.ListCommentsRequest{
			PostId:   uint64(post.ID),
			Page:     1,
			PageSize: 10,
		}
		resp, err := handler.ListComments(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Comments, 10)
		assert.Equal(t, int32(15), resp.TotalCount)
		assert.Equal(t, int32(2), resp.TotalPages)
		req.Page = 2
		resp, err = handler.ListComments(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Comments, 5)
	})

	t.Run("invalid pagination parameters", func(t *testing.T) {
		handler, db, _ := setupInteractionTestFixture(t)
		post, err := createTestPost(db, false)
		require.NoError(t, err)
		req := &proto.ListCommentsRequest{
			PostId:   uint64(post.ID),
			Page:     0,
			PageSize: 10,
		}
		_, err = handler.ListComments(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		req.Page = 1
		req.PageSize = 0
		_, err = handler.ListComments(context.Background(), req)
		require.Error(t, err)
		st, ok = status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}
