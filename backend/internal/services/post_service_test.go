package services

import (
	"testing"

	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type MockPostRepository struct {
	mock.Mock
}

func (m *MockPostRepository) Create(post *models.Post) error {
	return m.Called(post).Error(0)
}

func (m *MockPostRepository) GetByID(id uint) (*models.Post, error) {
	args := m.Called(id)
	post, _ := args.Get(0).(*models.Post)
	return post, args.Error(1)
}

func (m *MockPostRepository) GetBySlug(slug string) (*models.Post, error) {
	args := m.Called(slug)
	post, _ := args.Get(0).(*models.Post)
	return post, args.Error(1)
}

func (m *MockPostRepository) Update(post *models.Post) error {
	return m.Called(post).Error(0)
}

func (m *MockPostRepository) Delete(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockPostRepository) List(page, perPage int, filters map[string]interface{}) ([]models.Post, int64, error) {
	args := m.Called(page, perPage, filters)
	posts, _ := args.Get(0).([]models.Post)
	return posts, args.Get(1).(int64), args.Error(2)
}

func (m *MockPostRepository) Search(req *models.PostSearchRequest) ([]models.Post, int64, error) {
	args := m.Called(req)
	posts, _ := args.Get(0).([]models.Post)
	return posts, args.Get(1).(int64), args.Error(2)
}

func (m *MockPostRepository) GetByAuthor(authorID uint, page, perPage int) ([]models.Post, int64, error) {
	args := m.Called(authorID, page, perPage)
	posts, _ := args.Get(0).([]models.Post)
	return posts, args.Get(1).(int64), args.Error(2)
}

func (m *MockPostRepository) GetByCategory(categoryID uint, page, perPage int) ([]models.Post, int64, error) {
	args := m.Called(categoryID, page, perPage)
	posts, _ := args.Get(0).([]models.Post)
	return posts, args.Get(1).(int64), args.Error(2)
}

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(category *models.Category) error {
	return m.Called(category).Error(0)
}

func (m *MockCategoryRepository) GetByID(id uint) (*models.Category, error) {
	args := m.Called(id)
	category, _ := args.Get(0).(*models.Category)
	return category, args.Error(1)
}

func (m *MockCategoryRepository) GetBySlug(slug string) (*models.Category, error) {
	args := m.Called(slug)
	category, _ := args.Get(0).(*models.Category)
	return category, args.Error(1)
}

func (m *MockCategoryRepository) Update(category *models.Category) error {
	return m.Called(category).Error(0)
}

func (m *MockCategoryRepository) Delete(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockCategoryRepository) List(page, perPage int) ([]models.Category, int64, error) {
	args := m.Called(page, perPage)
	categories, _ := args.Get(0).([]models.Category)
	return categories, args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryRepository) Search(req *models.CategorySearchRequest) ([]models.Category, int64, error) {
	args := m.Called(req)
	categories, _ := args.Get(0).([]models.Category)
	return categories, args.Get(1).(int64), args.Error(2)
}

var (
	_ repositories.PostRepository     = (*MockPostRepository)(nil)
	_ repositories.CategoryRepository = (*MockCategoryRepository)(nil)
)

func TestPostServiceCreate(t *testing.T) {
	t.Run("creates a draft post with a generated slug", func(t *testing.T) {
		postRepo := new(MockPostRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewPostService(postRepo, new(MockUserRepository), categoryRepo)
		category := &models.Category{ID: 3, Name: "DevOps"}
		var createdPost *models.Post

		categoryRepo.On("GetByID", uint(3)).Return(category, nil).Once()
		postRepo.On("Create", mock.MatchedBy(func(post *models.Post) bool {
			return post.Title == "Reliable Deployments" &&
				post.Slug == "reliable-deployments" &&
				post.AuthorID == 7 &&
				post.CategoryID == 3 &&
				post.Status == "draft"
		})).Run(func(args mock.Arguments) {
			createdPost = args.Get(0).(*models.Post)
			createdPost.ID = 11
		}).Return(nil).Once()
		postRepo.On("GetByID", uint(11)).Return(&models.Post{ID: 11, Title: "Reliable Deployments"}, nil).Once()

		post, err := service.Create(&models.CreatePostRequest{
			Title:      "Reliable Deployments",
			Content:    "A post about reliable deployments.",
			CategoryID: 3,
		}, 7)

		require.NoError(t, err)
		require.Equal(t, uint(11), post.ID)
		postRepo.AssertExpectations(t)
		categoryRepo.AssertExpectations(t)
	})

	t.Run("rejects a post with a missing category", func(t *testing.T) {
		postRepo := new(MockPostRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewPostService(postRepo, new(MockUserRepository), categoryRepo)
		categoryRepo.On("GetByID", uint(99)).Return(nil, gorm.ErrRecordNotFound).Once()

		post, err := service.Create(&models.CreatePostRequest{
			Title:      "Uncategorized",
			CategoryID: 99,
		}, 7)

		require.Nil(t, post)
		require.EqualError(t, err, "category not found")
		categoryRepo.AssertExpectations(t)
		postRepo.AssertExpectations(t)
	})
}

func TestPostServiceUpdate(t *testing.T) {
	t.Run("denies an author access to another author's post", func(t *testing.T) {
		postRepo := new(MockPostRepository)
		service := NewPostService(postRepo, new(MockUserRepository), new(MockCategoryRepository))
		postRepo.On("GetByID", uint(11)).Return(&models.Post{ID: 11, AuthorID: 7}, nil).Once()

		title := "Changed title"
		post, err := service.Update(11, &models.UpdatePostRequest{Title: &title}, 8, "author")

		require.Nil(t, post)
		require.EqualError(t, err, "you don't have permission to update this post")
		postRepo.AssertExpectations(t)
	})

	t.Run("allows an admin to update another author's post", func(t *testing.T) {
		postRepo := new(MockPostRepository)
		service := NewPostService(postRepo, new(MockUserRepository), new(MockCategoryRepository))
		existing := &models.Post{ID: 11, Title: "Old title", Slug: "old-title", AuthorID: 7}
		updated := &models.Post{ID: 11, Title: "New title", Slug: "new-title", AuthorID: 7}
		title := "New title"

		postRepo.On("GetByID", uint(11)).Return(existing, nil).Once()
		postRepo.On("Update", mock.MatchedBy(func(post *models.Post) bool {
			return post.ID == 11 && post.Title == "New title" && post.Slug == "new-title"
		})).Return(nil).Once()
		postRepo.On("GetByID", uint(11)).Return(updated, nil).Once()

		post, err := service.Update(11, &models.UpdatePostRequest{Title: &title}, 8, "admin")

		require.NoError(t, err)
		require.Equal(t, "New title", post.Title)
		postRepo.AssertExpectations(t)
	})
}

func TestPostServiceDelete(t *testing.T) {
	t.Run("deletes the author's own post", func(t *testing.T) {
		postRepo := new(MockPostRepository)
		service := NewPostService(postRepo, new(MockUserRepository), new(MockCategoryRepository))
		postRepo.On("GetByID", uint(11)).Return(&models.Post{ID: 11, AuthorID: 7}, nil).Once()
		postRepo.On("Delete", uint(11)).Return(nil).Once()

		require.NoError(t, service.Delete(11, 7, "author"))
		postRepo.AssertExpectations(t)
	})
}
