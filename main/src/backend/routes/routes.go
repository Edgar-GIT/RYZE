package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminroles"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_login"
	"ryze/backend/services/admin_trainers"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/change_password"
	"ryze/backend/services/delete_account"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
	"ryze/backend/services/token"
)

// Setup wires all dependencies and registers the API routes.
func Setup(db *gorm.DB, jwtCfg config.JWTConfig, corsCfg config.CORSConfig, adminCfg config.AdminConfig) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CORS(corsCfg.AllowedOrigins))

	userRepository := repositories.NewUserRepository(db)

	registrationService := registration.NewRegistrationService(userRepository, password.Hasher{})
	registerHandler := auth.NewRegisterHandler(registrationService)
	tokenService := token.NewService([]byte(jwtCfg.Secret), jwtCfg.AccessTokenTTL)
	loginService := login.NewLoginService(userRepository, password.Verifier{})
	loginHandler := auth.NewLoginHandler(loginService, tokenService, jwtCfg.AccessTokenTTL, jwtCfg.CookieSecure)

	adminLoginService := admin_login.NewService(adminCredentials(adminCfg))
	adminLoginHandler := auth.NewAdminLoginHandler(adminLoginService, tokenService, jwtCfg.AccessTokenTTL, jwtCfg.CookieSecure)

	meHandler := auth.NewMeHandler(userRepository)
	logoutHandler := auth.NewLogoutHandler(jwtCfg.CookieSecure)

	changePasswordService := change_password.NewChangePasswordService(userRepository, password.Verifier{}, password.Hasher{})
	changePasswordHandler := auth.NewChangePasswordHandler(changePasswordService, jwtCfg.CookieSecure)

	deleteAccountService := delete_account.NewDeleteAccountService(userRepository, password.Verifier{})
	deleteAccountHandler := auth.NewDeleteAccountHandler(deleteAccountService, jwtCfg.CookieSecure)

	adminUsersService := admin_users.NewAdminUserService(userRepository, registrationService, password.Hasher{})
	adminUsersHandler := auth.NewAdminUserHandler(adminUsersService)

	trainerRepository := repositories.NewTrainerRepository(db)
	adminTrainersService := admin_trainers.NewAdminTrainerService(trainerRepository, userRepository, registrationService, adminUsersService)
	adminTrainersHandler := auth.NewAdminTrainerHandler(adminTrainersService)

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", registerHandler.Register)
	v1.POST("/auth/login", loginHandler.Login)
	v1.POST("/auth/logout", logoutHandler.Logout)
	v1.POST("/admin/auth/login", adminLoginHandler.Login)
	v1.POST("/admin/auth/verify", adminLoginHandler.Verify)
	v1.POST("/auth/change-password", middleware.Authenticate(tokenService, userRepository), changePasswordHandler.ChangePassword)
	v1.POST("/auth/delete-account", middleware.Authenticate(tokenService, userRepository), deleteAccountHandler.DeleteAccount)
	v1.GET("/me", middleware.Authenticate(tokenService, userRepository), meHandler.GetMe)

	admin := v1.Group("/admin")
	admin.Use(middleware.AdminAuthenticate(tokenService))

	adminRead := admin.Group("")
	adminRead.Use(middleware.RequireAdminPermission(adminroles.PermissionUsersRead))
	adminRead.GET("/users", adminUsersHandler.ListUsers)
	adminRead.GET("/users/:id", adminUsersHandler.GetUser)

	adminMutate := admin.Group("")
	adminMutate.Use(middleware.RequireAdminPermission(adminroles.PermissionUsersManage))
	adminMutate.GET("/users/deleted", adminUsersHandler.ListDeletedUsers)
	adminMutate.POST("/users", adminUsersHandler.CreateUser)
	adminMutate.PATCH("/users/:id", adminUsersHandler.UpdateUser)
	adminMutate.PATCH("/users/:id/disable", adminUsersHandler.SoftDeleteUser)
	adminMutate.POST("/users/:id/reactivate", adminUsersHandler.ReactivateUser)
	adminMutate.POST("/users/:id/password", adminUsersHandler.ResetUserPassword)

	adminTrainerRead := admin.Group("")
	adminTrainerRead.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainersRead))
	adminTrainerRead.GET("/trainers", adminTrainersHandler.ListTrainers)
	adminTrainerRead.GET("/trainers/:id", adminTrainersHandler.GetTrainer)

	adminTrainerMutate := admin.Group("")
	adminTrainerMutate.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainersManage))
	adminTrainerMutate.GET("/trainers/deleted", adminTrainersHandler.ListDeletedTrainers)
	adminTrainerMutate.POST("/trainers", adminTrainersHandler.CreateTrainer)
	adminTrainerMutate.PATCH("/trainers/:id", adminTrainersHandler.UpdateTrainer)
	adminTrainerMutate.PATCH("/trainers/:id/disable", adminTrainersHandler.SoftDeleteTrainer)
	adminTrainerMutate.POST("/trainers/:id/reactivate", adminTrainersHandler.ReactivateTrainer)

	return router
}

// adminCredentials converts the configured administrators into service
// credentials.
func adminCredentials(cfg config.AdminConfig) []admin_login.AdminCredential {
	credentials := make([]admin_login.AdminCredential, 0, len(cfg.Admins))
	for _, admin := range cfg.Admins {
		credentials = append(credentials, admin_login.AdminCredential{
			ID:         admin.ID,
			Username:   admin.Username,
			Password:   admin.Password,
			AccessCode: admin.AccessCode,
		})
	}
	return credentials
}
