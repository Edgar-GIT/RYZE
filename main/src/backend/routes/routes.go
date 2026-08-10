package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/middleware"
	"ryze/backend/repositories"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
	"ryze/backend/services/token"
)

// Setup wires all dependencies and registers the API routes.
func Setup(db *gorm.DB, jwtCfg config.JWTConfig) *gin.Engine {
	router := gin.Default()

	userRepository := repositories.NewUserRepository(db)

	registrationService := registration.NewRegistrationService(userRepository, password.Hasher{})
	registerHandler := auth.NewRegisterHandler(registrationService)

	tokenService := token.NewService([]byte(jwtCfg.Secret), jwtCfg.AccessTokenTTL)
	loginService := login.NewLoginService(userRepository, password.Verifier{})
	loginHandler := auth.NewLoginHandler(loginService, tokenService, jwtCfg.AccessTokenTTL, jwtCfg.CookieSecure)

	meHandler := auth.NewMeHandler(userRepository)
	logoutHandler := auth.NewLogoutHandler(jwtCfg.CookieSecure)

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", registerHandler.Register)
	v1.POST("/auth/login", loginHandler.Login)
	v1.POST("/auth/logout", logoutHandler.Logout)
	v1.GET("/me", middleware.Authenticate(tokenService), meHandler.GetMe)

	return router
}
