package handler

import (
	"io/fs"
	"net/http"

	"github.com/Uranury/oci-platform/internal/auth"
)

// NewRouter wires the API routes and serves the frontend (static files from
// webFS) at "/" for every GET path no API route claims.
func NewRouter(authH *AuthHandler, exerciseH *ExerciseHandler, workoutH *WorkoutHandler, templateH *TemplateHandler, jwtIssuer *auth.JWTIssuer, webFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", authH.Register)
	mux.HandleFunc("POST /auth/login", authH.Login)
	mux.HandleFunc("POST /auth/refresh", authH.Refresh)
	mux.HandleFunc("POST /auth/logout", authH.Logout)
	mux.HandleFunc("POST /auth/logout-all", requireAuth(jwtIssuer, authH.LogoutAll))

	mux.HandleFunc("GET /exercise-options", exerciseH.Options)
	mux.HandleFunc("GET /exercises", requireAuth(jwtIssuer, exerciseH.List))
	mux.HandleFunc("POST /exercises", requireAuth(jwtIssuer, exerciseH.Create))
	mux.HandleFunc("GET /exercises/similar", requireAuth(jwtIssuer, exerciseH.Similar))
	mux.HandleFunc("PUT /exercises/{id}", requireAuth(jwtIssuer, exerciseH.Update))
	mux.HandleFunc("DELETE /exercises/{id}", requireAuth(jwtIssuer, exerciseH.Delete))
	mux.HandleFunc("POST /exercises/{id}/duplicate", requireAuth(jwtIssuer, exerciseH.Duplicate))

	mux.HandleFunc("GET /workouts", requireAuth(jwtIssuer, workoutH.ListWorkouts))
	mux.HandleFunc("POST /workouts", requireAuth(jwtIssuer, workoutH.CreateWorkout))
	mux.HandleFunc("GET /workouts/{id}", requireAuth(jwtIssuer, workoutH.GetWorkout))
	mux.HandleFunc("PUT /workouts/{id}", requireAuth(jwtIssuer, workoutH.UpdateWorkout))
	mux.HandleFunc("DELETE /workouts/{id}", requireAuth(jwtIssuer, workoutH.DeleteWorkout))
	mux.HandleFunc("POST /workouts/{id}/sets", requireAuth(jwtIssuer, workoutH.AddSet))
	mux.HandleFunc("DELETE /workouts/{id}/sets/{setID}", requireAuth(jwtIssuer, workoutH.DeleteSet))
	mux.HandleFunc("POST /workouts/{id}/save-as-template", requireAuth(jwtIssuer, templateH.FromWorkout))

	mux.HandleFunc("GET /templates", requireAuth(jwtIssuer, templateH.List))
	mux.HandleFunc("POST /templates", requireAuth(jwtIssuer, templateH.Create))
	mux.HandleFunc("GET /templates/{id}", requireAuth(jwtIssuer, templateH.Get))
	mux.HandleFunc("PUT /templates/{id}", requireAuth(jwtIssuer, templateH.Update))
	mux.HandleFunc("DELETE /templates/{id}", requireAuth(jwtIssuer, templateH.Delete))
	mux.HandleFunc("POST /templates/{id}/start", requireAuth(jwtIssuer, templateH.Start))

	mux.Handle("GET /", http.FileServerFS(webFS))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello, you found me!\n"))
	})

	return mux
}
