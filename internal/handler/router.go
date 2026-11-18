package handler

import (
	"io/fs"
	"net/http"

	"github.com/Uranury/oci-platform/internal/auth"
)

// NewRouter wires the API routes and serves the frontend (static files from
// webFS) at "/" for every GET path no API route claims.
func NewRouter(authH *AuthHandler, workoutH *WorkoutHandler, jwtIssuer *auth.JWTIssuer, webFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", authH.Register)
	mux.HandleFunc("POST /auth/login", authH.Login)
	mux.HandleFunc("POST /auth/refresh", authH.Refresh)
	mux.HandleFunc("POST /auth/logout", authH.Logout)
	mux.HandleFunc("POST /auth/logout-all", requireAuth(jwtIssuer, authH.LogoutAll))

	mux.HandleFunc("GET /exercises", requireAuth(jwtIssuer, workoutH.ListExercises))
	mux.HandleFunc("POST /exercises", requireAuth(jwtIssuer, workoutH.CreateExercise))
	mux.HandleFunc("DELETE /exercises/{id}", requireAuth(jwtIssuer, workoutH.DeleteExercise))

	mux.HandleFunc("GET /workouts", requireAuth(jwtIssuer, workoutH.ListWorkouts))
	mux.HandleFunc("POST /workouts", requireAuth(jwtIssuer, workoutH.CreateWorkout))
	mux.HandleFunc("GET /workouts/{id}", requireAuth(jwtIssuer, workoutH.GetWorkout))
	mux.HandleFunc("PUT /workouts/{id}", requireAuth(jwtIssuer, workoutH.UpdateWorkout))
	mux.HandleFunc("DELETE /workouts/{id}", requireAuth(jwtIssuer, workoutH.DeleteWorkout))
	mux.HandleFunc("POST /workouts/{id}/sets", requireAuth(jwtIssuer, workoutH.AddSet))
	mux.HandleFunc("DELETE /workouts/{id}/sets/{setID}", requireAuth(jwtIssuer, workoutH.DeleteSet))

	mux.Handle("GET /", http.FileServerFS(webFS))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello, you found me!\n"))
	})

	return mux
}
