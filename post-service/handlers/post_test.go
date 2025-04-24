package handlers

import (
	"context"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"post-service/repositories"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"post-service/models"
	"post-service/proto"
)

func fixtureDb(t *testing.T) *repositories.PostRepository {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	err := db.AutoMigrate(&models.Post{}, &models.Tag{}, &models.PostTag{})
	assert.NoError(t, err)
	return repositories.NewPostRepository(db)
}

func TestCreatePost(t *testing.T) {
	creatorID := "user123"

	t.Run("successful creation", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		req := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		response, err := handler.CreatePost(context.Background(), req)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Id)
		assert.Equal(t, req.Title, response.Title)
		assert.Equal(t, req.Description, response.Description)
		assert.Equal(t, req.CreatorId, response.CreatorId)
		assert.Equal(t, req.IsPrivate, response.IsPrivate)
		assert.ElementsMatch(t, req.Tags, response.Tags)
	})

	t.Run("missing title", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		req := &proto.CreatePostRequest{
			Title:       "",
			Description: "Test Description",
			CreatorId:   creatorID,
		}
		response, err := handler.CreatePost(context.Background(), req)
		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "title is required")
	})

	t.Run("missing creator ID", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		req := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   "",
		}
		response, err := handler.CreatePost(context.Background(), req)
		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "Post creatorId is required")
	})
}

func TestGetPost(t *testing.T) {
	creatorID := "user123"
	requesterID := "user123"

	t.Run("successful get", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id

		expectedPost := &models.Post{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorID:   creatorID,
			IsPrivate:   false,
			Tags:        []models.Tag{{Name: "tag1"}, {Name: "tag2"}},
		}
		getReq := &proto.GetPostRequest{
			Id:          postID,
			RequesterId: requesterID,
		}
		response, err := handler.GetPost(context.Background(), getReq)
		require.NoError(t, err)
		assert.Equal(t, postID, response.Id)
		assert.Equal(t, expectedPost.Title, response.Title)
		assert.Equal(t, expectedPost.Description, response.Description)
		assert.Equal(t, expectedPost.CreatorID, response.CreatorId)
		assert.Equal(t, expectedPost.IsPrivate, response.IsPrivate)
		assert.Len(t, response.Tags, 2)
		assert.Contains(t, response.Tags, "tag1")
		assert.Contains(t, response.Tags, "tag2")
	})

	t.Run("post not found", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		postID := 1
		req := &proto.GetPostRequest{
			Id:          uint64(postID),
			RequesterId: requesterID,
		}
		response, err := handler.GetPost(context.Background(), req)
		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Contains(t, st.Message(), "Post not found")
	})

	t.Run("no access to private post", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		otherRequesterID := "user456"
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   true,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id
		req := &proto.GetPostRequest{
			Id:          postID,
			RequesterId: otherRequesterID,
		}
		response, err := handler.GetPost(context.Background(), req)
		assert.NotNil(t, err)
		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		assert.Contains(t, st.Message(), "don't have permission")
	})
}

func TestUpdatePost(t *testing.T) {
	creatorID := "user123"
	t.Run("successful update", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id

		updatedTags := []string{"test", "updated"}
		req := &proto.UpdatePostRequest{
			Id:          postID,
			Title:       "Updated Title",
			Description: "Updated Description",
			IsPrivate:   true,
			Tags:        updatedTags,
			UpdaterId:   creatorID,
		}
		response, err := handler.UpdatePost(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, postID, response.Id)
		assert.Equal(t, req.Title, response.Title)
		assert.Equal(t, req.Description, response.Description)
		assert.Equal(t, req.IsPrivate, response.IsPrivate)
		assert.ElementsMatch(t, req.Tags, response.Tags)
	})

	t.Run("post not found", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id

		req := &proto.UpdatePostRequest{
			Id:        postID + 1,
			Title:     "Updated Title",
			UpdaterId: creatorID,
		}
		response, err := handler.UpdatePost(context.Background(), req)
		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Contains(t, st.Message(), "Post not found")
	})

	t.Run("not post owner", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id

		otherUserID := "user456"
		req := &proto.UpdatePostRequest{
			Id:        postID,
			Title:     "Updated Title",
			UpdaterId: otherUserID,
		}
		response, err := handler.UpdatePost(context.Background(), req)

		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		assert.Contains(t, st.Message(), "don't have permission")
	})
}

func TestDeletePost(t *testing.T) {
	creatorID := "user123"

	t.Run("successful delete", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id

		req := &proto.DeletePostRequest{
			Id:        postID,
			DeleterId: creatorID,
		}

		response, err := handler.DeletePost(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Success)
	})

	t.Run("post not found", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id

		req := &proto.DeletePostRequest{
			Id:        postID + 1,
			DeleterId: creatorID,
		}

		response, err := handler.DeletePost(context.Background(), req)

		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Contains(t, st.Message(), "Post not found")
	})

	t.Run("not post owner", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		otherUserID := "user456"
		createReq := &proto.CreatePostRequest{
			Title:       "Test Post",
			Description: "Test Description",
			CreatorId:   creatorID,
			IsPrivate:   false,
			Tags:        []string{"tag1", "tag2"},
		}
		resp, err := handler.CreatePost(context.Background(), createReq)
		require.NoError(t, err)
		postID := resp.Id

		req := &proto.DeletePostRequest{
			Id:        postID,
			DeleterId: otherUserID,
		}
		response, err := handler.DeletePost(context.Background(), req)
		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		assert.Contains(t, st.Message(), "don't have permission")
	})
}

func TestListPosts(t *testing.T) {
	requesterID := "user123"
	creatorID := "user456"

	t.Run("successful list", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		posts := []models.Post{
			{
				Title:       "Post 1",
				Description: "Description 1",
				CreatorID:   creatorID,
				IsPrivate:   false,
				Tags:        []models.Tag{{Name: "tag1"}, {Name: "tag2"}},
			},
			{
				Title:       "Post 2",
				Description: "Description 2",
				CreatorID:   creatorID,
				IsPrivate:   false,
				Tags:        []models.Tag{{Name: "tag2"}, {Name: "tag3"}},
			},
		}
		for _, post := range posts {
			newTags := make([]string, len(post.Tags))
			for _, tag := range post.Tags {
				newTags = append(newTags, tag.Name)
			}
			createReq := &proto.CreatePostRequest{
				Title:       post.Title,
				Description: post.Description,
				CreatorId:   creatorID,
				IsPrivate:   false,
				Tags:        newTags,
			}
			_, err := handler.CreatePost(context.Background(), createReq)
			require.NoError(t, err)
		}

		req := &proto.ListPostsRequest{
			Page:        1,
			PageSize:    10,
			RequesterId: requesterID,
			CreatorId:   creatorID,
			Tags:        []string{"tag2"},
		}

		response, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Posts, 2)
		assert.Equal(t, int32(2), response.TotalCount)
		assert.Equal(t, int32(1), response.TotalPages)
	})

	t.Run("empty list", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		req := &proto.ListPostsRequest{
			Page:        1,
			PageSize:    10,
			RequesterId: requesterID,
		}

		response, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Posts, 0)
		assert.Equal(t, int32(0), response.TotalCount)
		assert.Equal(t, int32(0), response.TotalPages)
	})

	t.Run("empty list [by tag]", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		posts := []models.Post{
			{
				Title:       "Post 1",
				Description: "Description 1",
				CreatorID:   creatorID,
				IsPrivate:   false,
				Tags:        []models.Tag{{Name: "tag1"}, {Name: "tag2"}},
			},
			{
				Title:       "Post 2",
				Description: "Description 2",
				CreatorID:   creatorID,
				IsPrivate:   false,
				Tags:        []models.Tag{{Name: "tag2"}, {Name: "tag3"}},
			},
		}
		for _, post := range posts {
			newTags := make([]string, len(post.Tags))
			for _, tag := range post.Tags {
				newTags = append(newTags, tag.Name)
			}
			createReq := &proto.CreatePostRequest{
				Title:       post.Title,
				Description: post.Description,
				CreatorId:   creatorID,
				IsPrivate:   false,
				Tags:        newTags,
			}
			_, err := handler.CreatePost(context.Background(), createReq)
			require.NoError(t, err)
		}

		req := &proto.ListPostsRequest{
			Page:        1,
			PageSize:    10,
			RequesterId: requesterID,
			CreatorId:   creatorID,
			Tags:        []string{"tag4"},
		}

		response, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Posts, 0)
		assert.Equal(t, int32(0), response.TotalCount)
		assert.Equal(t, int32(0), response.TotalPages)
	})

	t.Run("no private", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		posts := []models.Post{
			{
				Title:       "Post 1",
				Description: "Description 1",
				CreatorID:   creatorID,
				IsPrivate:   false,
				Tags:        []models.Tag{{Name: "tag1"}, {Name: "tag2"}},
			},
			{
				Title:       "Post 2",
				Description: "Description 2",
				CreatorID:   creatorID,
				IsPrivate:   true,
				Tags:        []models.Tag{{Name: "tag2"}, {Name: "tag3"}},
			},
		}
		for _, post := range posts {
			newTags := make([]string, len(post.Tags))
			for _, tag := range post.Tags {
				newTags = append(newTags, tag.Name)
			}
			createReq := &proto.CreatePostRequest{
				Title:       post.Title,
				Description: post.Description,
				CreatorId:   creatorID,
				IsPrivate:   post.IsPrivate,
				Tags:        newTags,
			}
			_, err := handler.CreatePost(context.Background(), createReq)
			require.NoError(t, err)
		}

		req := &proto.ListPostsRequest{
			Page:        1,
			PageSize:    10,
			RequesterId: requesterID,
			CreatorId:   creatorID,
			Tags:        []string{"tag2"},
		}

		response, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Posts, 1)
		assert.Equal(t, int32(1), response.TotalCount)
		assert.Equal(t, int32(1), response.TotalPages)
	})

	t.Run("pagination", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		posts := []models.Post{
			{
				Title:       "Post 1",
				Description: "Description 1",
				CreatorID:   creatorID,
				IsPrivate:   false,
				Tags:        []models.Tag{{Name: "tag1"}, {Name: "tag2"}},
			},
		}
		for _, post := range posts {
			for i := range 100 {
				newTags := make([]string, len(post.Tags))
				for _, tag := range post.Tags {
					newTags = append(newTags, tag.Name)
				}
				createReq := &proto.CreatePostRequest{
					Title:       post.Title,
					Description: post.Description,
					CreatorId:   creatorID,
					IsPrivate:   i%2 == 0,
					Tags:        newTags,
				}
				_, err := handler.CreatePost(context.Background(), createReq)
				require.NoError(t, err)
			}
		}

		req := &proto.ListPostsRequest{
			Page:        1,
			PageSize:    10,
			RequesterId: requesterID,
			CreatorId:   creatorID,
			Tags:        []string{"tag2"},
		}

		response, err := handler.ListPosts(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Posts, 10)
		assert.Equal(t, int32(50), response.TotalCount)
		assert.Equal(t, int32(5), response.TotalPages)
	})

	t.Run("bad pagination", func(t *testing.T) {
		handler := NewPostHandler(fixtureDb(t))
		req := &proto.ListPostsRequest{
			Page:        -2,
			PageSize:    10,
			RequesterId: requesterID,
			CreatorId:   creatorID,
			Tags:        []string{"tag2"},
		}

		response, err := handler.ListPosts(context.Background(), req)
		assert.Nil(t, response)
		assert.NotNil(t, err)

		req.Page, req.PageSize = 1, -20
		response, err = handler.ListPosts(context.Background(), req)
		assert.Nil(t, response)
		assert.NotNil(t, err)
	})
}
