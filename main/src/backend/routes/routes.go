package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

// Setup wires all dependencies and registers the API routes.
func Setup(db *gorm.DB) *gin.Engine {
	router := gin.Default()

	registrationService := registration.NewRegistrationService(
		repositories.NewUserRepository(db),
		password.Hasher{},
	)
	registerHandler := auth.NewRegisterHandler(registrationService)

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", registerHandler.Register)

	return router
}
