package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/controllers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/database"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/routes"
)

func main() {
	// Initialize configuration and database
	env := config.LoadENV()
	db := database.Connection(env)
	validator := models.InitValidator()

	repo := repositories.InitializeRepo(db)

	handlerFunc := controllers.NewHandler(env, repo, validator)

	// Create a new Gin router
	r := gin.Default()
	models.InitValidator()
	routes.SetupRoutes(r, handlerFunc)

	fmt.Printf("Starting server on port %s\n", env.APP_PORT)

	// Start the Gin server
	if err := r.Run(":" + env.APP_PORT); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
