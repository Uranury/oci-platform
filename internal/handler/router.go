package handler

import (
	"io/fs"
	"mime"
	"net/http"
	"strings"

	"github.com/Uranury/oci-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

// Go's table has no entry for the PWA manifest; without one it would be
// sniffed and served as text/plain.
func init() {
	if err := mime.AddExtensionType(".webmanifest", "application/manifest+json"); err != nil {
		panic(err)
	}
}

// NewRouter wires the API routes and serves the frontend (static files from
// webFS) for every GET path no API route claims.
func NewRouter(authH *AuthHandler, exerciseH *ExerciseHandler, workoutH *WorkoutHandler, templateH *TemplateHandler, bodyweightH *BodyweightHandler, jwtIssuer *auth.JWTIssuer, webFS fs.FS) http.Handler {
	gin.SetMode(gin.ReleaseMode) // no [GIN-debug] route dump on stdout; logging goes through slog
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(logRequests, securityHeaders, gin.CustomRecovery(recoverPanic), handleErrors)

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "Hello, you found me!\n") })

	r.POST("/auth/register", authH.Register)
	r.POST("/auth/login", authH.Login)
	r.POST("/auth/refresh", authH.Refresh)
	r.POST("/auth/logout", authH.Logout)
	r.GET("/exercise-options", exerciseH.Options)

	api := r.Group("/", requireAuth(jwtIssuer))
	api.POST("/auth/logout-all", authH.LogoutAll)
	api.POST("/auth/delete-account", authH.DeleteAccount)

	api.GET("/exercises", exerciseH.List)
	api.POST("/exercises", exerciseH.Create)
	api.GET("/exercises/similar", exerciseH.Similar)
	api.GET("/exercises/last", workoutH.LastSets)
	api.GET("/exercises/:id/history", workoutH.ExerciseHistory)
	api.PUT("/exercises/:id", exerciseH.Update)
	api.DELETE("/exercises/:id", exerciseH.Delete)
	api.POST("/exercises/:id/duplicate", exerciseH.Duplicate)

	api.GET("/workouts", workoutH.ListWorkouts)
	api.GET("/workouts/export", workoutH.ExportWorkouts)
	api.POST("/workouts", workoutH.CreateWorkout)
	api.GET("/workouts/:id", workoutH.GetWorkout)
	api.PUT("/workouts/:id", workoutH.UpdateWorkout)
	api.DELETE("/workouts/:id", workoutH.DeleteWorkout)
	api.POST("/workouts/:id/finish", workoutH.FinishWorkout)
	api.GET("/workouts/:id/previous", workoutH.PreviousSets)
	api.POST("/workouts/:id/exercises", workoutH.AddExercise)
	api.PUT("/workouts/:id/exercises", workoutH.ReorderExercises)
	api.DELETE("/workouts/:id/exercises/:exerciseID", workoutH.RemoveExercise)
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

	api.GET("/bodyweight", bodyweightH.List)
	api.PUT("/bodyweight/:day", bodyweightH.Put)
	api.DELETE("/bodyweight/:day", bodyweightH.Delete)

	files := http.FileServerFS(webFS)
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Vite puts a content hash in every file name under /assets/, so
		// those never change; everything else (index.html, favicon) is
		// revalidated so a deploy shows up on the next load.
		if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		files.ServeHTTP(c.Writer, c.Request)
	})

	return r
}
