package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/Endea4/studExE4-location-service/docs"
	"github.com/Endea4/studExE4-location-service/internal/handler"
	"github.com/Endea4/studExE4-location-service/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title StudEx Location Service API
// @version 1.0
// @description Real-time GPS tracking service with Redis geo indexing.
// @description Generic entity tracking — accepts any ref_id (driver, rider, courier, etc).
// @host localhost:8083
// @BasePath /
// @schemes http https
func main() {
	port := getEnv("PORT", "8083")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPass := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPass,
		DB:           redisDB,
		DialTimeout:  0,
		ReadTimeout:  0,
		WriteTimeout: 0,
		PoolSize:     100,
		MinIdleConns: 10,
	})

	repo := repository.NewLocationRepository(rdb)
	locHandler := handler.NewLocationHandler(repo)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","service":"location-service"}`))
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Get("/dashboard", handler.MapDashboard)

	r.Route("/location", func(r chi.Router) {
		r.Post("/", locHandler.UpdateLocation)
		r.Get("/", locHandler.GetAllLocations)
		r.Get("/nearby", locHandler.FindNearby)
		r.Get("/{ref_id}", locHandler.GetLocation)
		r.Delete("/{ref_id}", locHandler.RemoveEntity)
	})

	addr := ":" + port
	fmt.Printf("Location Service running on %s\n", addr)
	fmt.Printf("Swagger UI: http://localhost%s/swagger/index.html\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
