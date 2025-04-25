package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"social-network/user-service/contracts"
	"social-network/user-service/handlers"
	"social-network/user-service/models"
	"social-network/user-service/repositories"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func fixture() (*gin.Engine, *repositories.UserRepository) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	err := db.AutoMigrate(&models.User{})
	if err != nil {
		panic(err)
	}

	userRepo := repositories.NewUserRepository(db)
	userHandler := handlers.NewUserHandler(userRepo, "test_secret_key", nil)

	router := gin.Default()
	router.POST("/api/auth/register", userHandler.Register)
	router.POST("/api/auth/login", userHandler.Login)

	auth := router.Group("/api/users")
	auth.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	auth.GET("/profile", userHandler.GetProfile)
	auth.PUT("/profile", userHandler.UpdateProfile)

	return router, userRepo
}

func TestRegister(t *testing.T) {
	router, userRepo := fixture()

	birthDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	registerRequest := contracts.RegisterRequest{
		Username:    "user",
		Email:       "email@email.com",
		Password:    "password",
		FirstName:   "f",
		LastName:    "s",
		BirthDate:   &birthDate,
		PhoneNumber: "0",
	}

	requestBody, _ := json.Marshal(registerRequest)
	w := httptest.NewRecorder()

	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response contracts.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	assert.Equal(t, registerRequest.Username, response.User.Username)
	assert.Equal(t, registerRequest.Email, response.User.Email)
	assert.Equal(t, registerRequest.FirstName, response.User.FirstName)
	assert.Equal(t, registerRequest.LastName, response.User.LastName)
	assert.Equal(t, registerRequest.PhoneNumber, response.User.PhoneNumber)
	assert.NotEmpty(t, response.JwtToken)

	user, _ := userRepo.FindByUsername("user")
	assert.NotNil(t, user)
	assert.Equal(t, registerRequest.Email, user.Email)
}

func TestLogin(t *testing.T) {
	router, userRepo := fixture()

	user := &models.User{
		Username:  "user",
		Email:     "email@email.com",
		Password:  "password",
		FirstName: "f",
		LastName:  "s",
	}
	{
		ttt := time.Now()
		user.CreatedAt = ttt
		user.UpdatedAt = ttt
	}
	err := userRepo.CreateUser(user)
	assert.Nil(t, err)

	loginRequest := contracts.LoginRequest{
		Username: "user",
		Password: "password",
	}

	requestBody, _ := json.Marshal(loginRequest)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response contracts.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	assert.Equal(t, user.Username, response.User.Username)
	assert.Equal(t, user.Email, response.User.Email)
	assert.NotEmpty(t, response.JwtToken)
}

func TestUpdateProfile(t *testing.T) {
	router, userRepo := fixture()

	// Создаем пользователя
	user := &models.User{
		Username:  "user",
		Email:     "email@email.com",
		Password:  "password",
		FirstName: "f",
		LastName:  "s",
	}
	{
		ttt := time.Now()
		user.CreatedAt = ttt
		user.UpdatedAt = ttt
		user.ID = 1
	}
	err := userRepo.CreateUser(user)
	assert.Nil(t, err)
	updateRequest := contracts.UpdateProfileRequest{
		FirstName:   "f1",
		LastName:    "s1",
		Email:       "email1@email.com",
		PhoneNumber: "01",
	}

	requestBody, _ := json.Marshal(updateRequest)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/profile", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedUser models.User
	err = json.Unmarshal(w.Body.Bytes(), &updatedUser)
	assert.Nil(t, err)

	assert.Equal(t, updateRequest.FirstName, updatedUser.FirstName)
	assert.Equal(t, updateRequest.LastName, updatedUser.LastName)
	assert.Equal(t, updateRequest.Email, updatedUser.Email)
	assert.Equal(t, updateRequest.PhoneNumber, updatedUser.PhoneNumber)

	userFromDB, _ := userRepo.FindByID(1)
	assert.Equal(t, updateRequest.FirstName, userFromDB.FirstName)
	assert.Equal(t, updateRequest.LastName, userFromDB.LastName)
	assert.Equal(t, updateRequest.Email, userFromDB.Email)
	assert.Equal(t, updateRequest.PhoneNumber, userFromDB.PhoneNumber)
}

func TestBadRegisterParameters(t *testing.T) {
	router, _ := fixture()

	makeRegisterRequest := func(request *contracts.RegisterRequest) *httptest.ResponseRecorder {
		requestBody, _ := json.Marshal(request)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}

	birthDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	registerRequest := contracts.RegisterRequest{
		Username:    "user",
		Email:       "e",
		Password:    "password",
		FirstName:   "f",
		LastName:    "s",
		BirthDate:   &birthDate,
		PhoneNumber: "0",
	}

	// wrong email
	w := makeRegisterRequest(&registerRequest)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// short password
	registerRequest.Email, registerRequest.Password = "email@email.com", "ab"
	w = makeRegisterRequest(&registerRequest)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// short username
	registerRequest.Password, registerRequest.Username = "aboba228", "ab"
	w = makeRegisterRequest(&registerRequest)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterExistingUser(t *testing.T) {
	router, _ := fixture()

	makeRegisterRequest := func(request *contracts.RegisterRequest) *httptest.ResponseRecorder {
		requestBody, _ := json.Marshal(request)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}

	birthDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	registerRequest := contracts.RegisterRequest{
		Username:    "user",
		Email:       "email@email.com",
		Password:    "password",
		FirstName:   "f",
		LastName:    "s",
		BirthDate:   &birthDate,
		PhoneNumber: "0",
	}

	w := makeRegisterRequest(&registerRequest)
	assert.Equal(t, http.StatusCreated, w.Code)

	// username already exists
	w = makeRegisterRequest(&registerRequest)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// email already exists
	registerRequest.Username = "test2"
	w = makeRegisterRequest(&registerRequest)
	t.Log(w.Body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBadLoginParameters(t *testing.T) {
	router, _ := fixture()

	makeRegisterRequest := func(request *contracts.RegisterRequest) *httptest.ResponseRecorder {
		requestBody, _ := json.Marshal(request)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}

	birthDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	registerRequest := contracts.RegisterRequest{
		Username:    "user",
		Email:       "email@email.com",
		Password:    "password",
		FirstName:   "f",
		LastName:    "s",
		BirthDate:   &birthDate,
		PhoneNumber: "0",
	}
	w := makeRegisterRequest(&registerRequest)
	assert.Equal(t, http.StatusCreated, w.Code)

	makeLoginRequest := func(request *contracts.LoginRequest) *httptest.ResponseRecorder {
		requestBody, _ := json.Marshal(request)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}

	loginRequest := contracts.LoginRequest{
		Username: "us",
		Password: "password",
	}
	w = makeLoginRequest(&loginRequest)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	loginRequest.Password, loginRequest.Username = "aboba228", "user"
	w = makeLoginRequest(&loginRequest)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetProfile(t *testing.T) {
	router, _ := fixture()

	// GET PROFILE
	makeGetProfileRequest := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/users/profile", bytes.NewBuffer(make([]byte, 0)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}

	w := makeGetProfileRequest()
	assert.Equal(t, http.StatusNotFound, w.Code)

	// REGISTER
	makeRegisterRequest := func(request *contracts.RegisterRequest) *httptest.ResponseRecorder {
		requestBody, _ := json.Marshal(request)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}

	birthDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	registerRequest := contracts.RegisterRequest{
		Username:    "user",
		Email:       "email@email.com",
		Password:    "password",
		FirstName:   "f",
		LastName:    "s",
		BirthDate:   &birthDate,
		PhoneNumber: "0",
	}
	w = makeRegisterRequest(&registerRequest)
	assert.Equal(t, http.StatusCreated, w.Code)

	w = makeGetProfileRequest()
	assert.Equal(t, http.StatusOK, w.Code)
}
