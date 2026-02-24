package admin

import (
	"net/http"
	"net/http/httptest"
	"roulette/internal/models"
	"roulette/internal/repository"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- MOCKS ---

type mockRender struct{}

func (m mockRender) Instance(name string, data interface{}) render.Render {
	return mockRenderInstance{}
}

type mockRenderInstance struct{}

func (m mockRenderInstance) Render(w http.ResponseWriter) error {
	return nil
}
func (m mockRenderInstance) WriteContentType(w http.ResponseWriter) {}

type MockRepository struct {
	repository.Repository
	mock.Mock
}

func (m *MockRepository) GetAdminUserByUsername(username string) (*models.AdminUserWithAccess, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AdminUserWithAccess), args.Error(1)
}

func (m *MockRepository) LogAccess(userID uint, module, perm, ip, ua string, allowed bool) error {
	args := m.Called(userID, module, perm, ip, ua, allowed)
	return args.Error(0)
}

func setupTestRouter(repo *MockRepository) (*gin.Engine, *AdminPanel) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.HTMLRender = mockRender{}
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("test_session", store))

	panel := &AdminPanel{
		repo:   repo,
		router: r,
	}
	return r, panel
}

// --- AUTH TESTS ---

func TestRbacAuthRequired(t *testing.T) {
	t.Run("No session - Redirect to login", func(t *testing.T) {
		repo := new(MockRepository)
		r, panel := setupTestRouter(repo)

		r.GET("/protected", panel.rbacAuthRequired(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login", w.Header().Get("Location"))
	})

	t.Run("User Inactive - Redirect with error", func(t *testing.T) {
		repo := new(MockRepository)
		r, panel := setupTestRouter(repo)

		repo.On("GetAdminUserByUsername", "bad_user").Return(&models.AdminUserWithAccess{
			AdminUser: models.AdminUser{Username: "bad_user", IsActive: false},
		}, nil)

		r.GET("/login_sim", func(c *gin.Context) {
			s := sessions.Default(c)
			s.Set("rbac_user", "bad_user")
			s.Save()
		})
		r.GET("/protected", panel.rbacAuthRequired(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/login_sim", nil)
		r.ServeHTTP(w1, req1)

		cookie := w1.Header().Get("Set-Cookie")

		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/protected", nil)
		req2.Header.Set("Cookie", cookie)
		r.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusFound, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "account_disabled")
	})
}

// --- PERMISSION TESTS ---

func TestRequireRead_ComplexLogic(t *testing.T) {
	t.Run("No permissions - Deny with 403", func(t *testing.T) {
		repo := new(MockRepository)
		r, panel := setupTestRouter(repo)

		r.GET("/test-read", func(c *gin.Context) {
			user := &models.AdminUserWithAccess{
				AdminUser:   models.AdminUser{Username: "tester"},
				Permissions: map[string]map[string]bool{"other": {"can_read": true}},
			}
			c.Set("admin_user", user)
			c.Next()
		}, panel.requireRead("users"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test-read", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestSuperAdminAccess(t *testing.T) {
	t.Run("SuperAdmin has access even without explicit permissions", func(t *testing.T) {
		repo := new(MockRepository)
		r, panel := setupTestRouter(repo)

		r.POST("/test-super", func(c *gin.Context) {
			user := &models.AdminUserWithAccess{
				AdminUser:    models.AdminUser{ID: 1, Username: "god_mode"},
				IsSuperAdmin: true,
				Permissions:  make(map[string]map[string]bool),
			}
			c.Set("admin_user", user)
			c.Next()
		}, panel.requireDelete("any_module"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test-super", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAllPermissionsMethods(t *testing.T) {
	repo := new(MockRepository)
	r, panel := setupTestRouter(repo)

	user := &models.AdminUserWithAccess{
		AdminUser: models.AdminUser{Username: "limited_user"},
		Permissions: map[string]map[string]bool{
			"reports": {"can_read": true},
			"users":   {"can_edit": true},
			"logs":    {"can_delete": true},
		},
	}

	tests := []struct {
		name     string
		path     string
		handler  gin.HandlerFunc
		expected int
	}{
		{"RequireEdit - Success", "/edit-ok", panel.requireEdit("users"), http.StatusOK},
		{"RequireEdit - Fail", "/edit-fail", panel.requireEdit("reports"), http.StatusForbidden},
		{"RequireDelete - Success", "/del-ok", panel.requireDelete("logs"), http.StatusOK},
		{"RequireWrite - Fail", "/write-fail", panel.requireWrite("reports"), http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.POST(tt.path, func(c *gin.Context) {
				c.Set("admin_user", user)
				c.Next()
			}, tt.handler, func(c *gin.Context) { c.Status(http.StatusOK) })

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", tt.path, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, w.Code)
		})
	}
}

// --- AJAX TESTS ---

func TestRbacAuthRequired_AJAX_Failure(t *testing.T) {
	repo := new(MockRepository)
	r, panel := setupTestRouter(repo)

	r.GET("/api/data", panel.rbacAuthRequired(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Session expired")
}

// --- LOGGING TESTS ---

func TestLogAccess_Triggered(t *testing.T) {
	repo := new(MockRepository)
	r, panel := setupTestRouter(repo)

	repo.On("LogAccess", uint(1), "users", "can_edit", mock.Anything, mock.Anything, true).Return(nil)

	r.POST("/users/edit/:id", func(c *gin.Context) {
		user := &models.AdminUserWithAccess{
			AdminUser:   models.AdminUser{ID: 1, Username: "admin"},
			Permissions: map[string]map[string]bool{"users": {"can_edit": true}},
		}
		c.Set("admin_user", user)
		c.Next()
	}, panel.requirePermission("users", "can_edit"), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users/edit/1", nil)
	req.Header.Set("User-Agent", "Test-Agent")
	req.RemoteAddr = "127.0.0.1:1234"

	r.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, http.StatusOK, w.Code)
	repo.AssertExpectations(t)
}
