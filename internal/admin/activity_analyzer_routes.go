package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// setupActivityAnalyzerRoutes sets up all activity analyzer routes
func (a *AdminPanel) setupActivityAnalyzerAPIRoutes() {
	// Activity analyzer API group - protected by auth
	analyzer := a.router.Group("/admin/api/user-activity-analyzer")
	analyzer.Use(a.ipFilterMiddleware(), a.authRequired())
	{
		// Dashboard & overview
		analyzer.GET("/dashboard", a.getActivityDashboard)
		analyzer.GET("/timeline", a.getOverallActivityTimeline)
		analyzer.GET("/action-types", a.getAllActionTypes)
		analyzer.GET("/top-action-types", a.getTopActionTypes)

		// Top suspicious users
		analyzer.GET("/users/top", a.getTopSuspiciousUsers)

		// Specific user analysis
		user := analyzer.Group("/users/:telegram_id")
		{
			user.GET("/detail", a.getUserActivityDetail)
			user.GET("/stats", a.getUserActivityStats)
			user.GET("/timeline", a.getUserActivityTimeline)
			user.GET("/actions", a.getUserRecentActions)
		}
	}
}

// API Handlers

func (a *AdminPanel) getActivityDashboard(c *gin.Context) {
	stats, err := a.service.GetActivityDashboardData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (a *AdminPanel) getTopSuspiciousUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	minActions, _ := strconv.Atoi(c.DefaultQuery("min_actions", "10"))

	timeFrom, timeTo := parseTimeRange(c)

	users, err := a.service.GetTopSuspiciousUsers(limit, timeFrom, timeTo, minActions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (a *AdminPanel) getUserActivityDetail(c *gin.Context) {
	telegramID, err := strconv.ParseInt(c.Param("telegram_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	timeFrom, timeTo := parseTimeRange(c)

	detail, err := a.service.GetUserActivityDetail(telegramID, timeFrom, timeTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (a *AdminPanel) getUserActivityStats(c *gin.Context) {
	telegramID, err := strconv.ParseInt(c.Param("telegram_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	timeFrom, timeTo := parseTimeRange(c)

	detail, err := a.service.GetUserActivityDetail(telegramID, timeFrom, timeTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail.Stats)
}

func (a *AdminPanel) getUserActivityTimeline(c *gin.Context) {
	telegramID, err := strconv.ParseInt(c.Param("telegram_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	interval := c.DefaultQuery("interval", "hour")
	actionType := c.Query("action_type")
	timeFrom, timeTo := parseTimeRange(c)

	timeline, err := a.service.GetUserActivityTimeline(telegramID, interval, timeFrom, timeTo, actionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, timeline)
}

func (a *AdminPanel) getUserRecentActions(c *gin.Context) {
	telegramID, err := strconv.ParseInt(c.Param("telegram_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	timeFrom, timeTo := parseTimeRange(c)

	detail, err := a.service.GetUserActivityDetail(telegramID, timeFrom, timeTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail.RecentActions)
}

func (a *AdminPanel) getAllActionTypes(c *gin.Context) {
	actionTypes, err := a.service.GetAllActivityActionTypes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, actionTypes)
}

func (a *AdminPanel) getTopActionTypes(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	actionTypes, err := a.service.GetTopActionTypes(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, actionTypes)
}

func (a *AdminPanel) getOverallActivityTimeline(c *gin.Context) {
	interval := c.DefaultQuery("interval", "hour")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "48"))
	timeFrom, timeTo := parseTimeRange(c)

	timeline, err := a.service.GetOverallActivityTimeline(interval, timeFrom, timeTo, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, timeline)
}

// Page Handlers

func (a *AdminPanel) activityAnalyzerDashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "activity_analyzer_dashboard", gin.H{
		"Title":     "Activity Analyzer - Dashboard",
		"activeTab": "activity_analyzer",
	})
}

func (a *AdminPanel) userActivityDetailPage(c *gin.Context) {
	telegramID := c.Param("telegram_id")
	c.HTML(http.StatusOK, "user_activity_detail", gin.H{
		"Title":      "User Activity Detail",
		"TelegramID": telegramID,
		"activeTab":  "activity_analyzer_user",
	})
}

// Helper function for parsing time range from query params
func parseTimeRange(c *gin.Context) (*time.Time, *time.Time) {
	var timeFrom, timeTo *time.Time

	if tf := c.Query("time_from"); tf != "" {
		if t, err := time.Parse(time.RFC3339, tf); err == nil {
			timeFrom = &t
		}
	}

	if tt := c.Query("time_to"); tt != "" {
		if t, err := time.Parse(time.RFC3339, tt); err == nil {
			timeTo = &t
		}
	}

	return timeFrom, timeTo
}
