package handler

import (
	"io/fs"
	"net/http"

	"github.com/Uranury/oci-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

// NewRouter wires the API routes and serves the frontend (static files from
// webFS) for every GET path no API route claims.
func NewRouter(authH *AuthHandler, exerciseH *ExerciseHandler, workoutH *WorkoutHandler, templateH *TemplateHandler, jwtIssuer *auth.JWTIssuer, webFS fs.FS) http.Handler {
	gin.SetMode(gin.ReleaseMode) // no [GIN-debug] route dump on stdout; logging goes through slog
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(logRequests, gin.CustomRecovery(recoverPanic), handleErrors)

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "Hello, you found me!\n") })

	r.POST("/auth/register", authH.Register)
	r.POST("/auth/login", authH.Login)
	r.POST("/auth/refresh", authH.Refresh)
	r.POST("/auth/logout", authH.Logout)
	r.GET("/exercise-options", exerciseH.Options)

	api := r.Group("/", requireAuth(jwtIssuer))
	api.POST("/auth/logout-all", authH.LogoutAll)

	api.GET("/exercises", exerciseH.List)
	api.POST("/exercises", exerciseH.Create)
	api.GET("/exercises/similar", exerciseH.Similar)
	api.PUT("/exercises/:id", exerciseH.Update)
	api.DELETE("/exercises/:id", exerciseH.Delete)
	api.POST("/exercises/:id/duplicate", exerciseH.Duplicate)

	api.GET("/workouts", workoutH.ListWorkouts)
	api.POST("/workouts", workoutH.CreateWorkout)
	api.GET("/workouts/:id", workoutH.GetWorkout)
	api.PUT("/workouts/:id", workoutH.UpdateWorkout)
	api.DELETE("/workouts/:id", workoutH.DeleteWorkout)
	api.POST("/workouts/:id/finish", workoutH.FinishWorkout)
	api.GET("/workouts/:id/previous", workoutH.PreviousSets)
	api.POST("/workouts/:id/sets", workoutH.AddSet)
	api.PUT("/workouts/:id/sets/:setID", workoutH.UpdateSet)
	api.DELETE("/workouts/:id/sets/:setID", workoutH.DeleteSet)
	api.POST("/workouts/:id/save-as-template", templateH.FromWorkout)

	api.GET("/templates", templateH.List)
	api.POST("/templates", templateH.Create)
	api.GET("/templates/:id", templateH.Get)
	api.PUT("/templates/:id", templateH.Update)
	api.DELETE("/templates/:id", templateH.Delete)
	api.POST("/templates/:id/start", templateH.Start)

	files := http.FileServerFS(webFS)
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		files.ServeHTTP(c.Writer, c.Request)
	})

	return r
}
