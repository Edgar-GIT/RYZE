package routes

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	paypal "github.com/plutov/paypal/v4"
	stripe "github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/api/webhooks"
	"ryze/backend/config"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminroles"
	"ryze/backend/middleware/trainerroles"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_login"
	"ryze/backend/services/admin_program_pricing"
	"ryze/backend/services/admin_trainers"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/change_password"
	"ryze/backend/services/client_programs"
	"ryze/backend/services/commission_rules"
	"ryze/backend/services/delete_account"
	"ryze/backend/services/entitlements"
	"ryze/backend/services/exercises"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/payments"
	"ryze/backend/services/program_assignments"
	"ryze/backend/services/program_structure"
	"ryze/backend/services/programs"
	"ryze/backend/services/public_programs"
	"ryze/backend/services/purchases"
	"ryze/backend/services/registration"
	"ryze/backend/services/statistics"
	"ryze/backend/services/token"
	"ryze/backend/services/trainer_applications"
	"ryze/backend/services/trainer_clients"
	"ryze/backend/services/trainer_profile"
	"ryze/backend/services/workout_exercises"
	"ryze/backend/services/workout_history"
)

// Setup wires all dependencies and registers the API routes.
func Setup(db *gorm.DB, jwtCfg config.JWTConfig, corsCfg config.CORSConfig, adminCfg config.AdminConfig, pricingCfg config.PricingConfig, commissionCfg config.CommissionConfig, stripeCfg config.StripeConfig, paypalCfg config.PayPalConfig, webhookCfg config.WebhookConfig) *gin.Engine {
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

	trainerApplicationRepository := repositories.NewTrainerApplicationRepository(db)
	trainerApplicationService := trainer_applications.NewService(trainerApplicationRepository, userRepository, trainerRepository)
	adminTrainerApplicationHandler := auth.NewAdminTrainerApplicationHandler(trainerApplicationService)
	trainerApplicationHandler := auth.NewTrainerApplicationHandler(trainerApplicationService)

	trainerProfileService := trainer_profile.NewService(trainerRepository)
	trainerProfileHandler := auth.NewTrainerProfileHandler(trainerProfileService)

	trainerClientRepository := repositories.NewTrainerClientRepository(db)
	trainerClientService := trainer_clients.NewService(trainerClientRepository, userRepository)
	trainerClientHandler := auth.NewTrainerClientsHandler(trainerClientService)

	programAssignmentRepository := repositories.NewProgramAssignmentRepository(db)
	programAssignmentService := program_assignments.NewService(programAssignmentRepository)
	programAssignmentHandler := auth.NewTrainerClientProgramHandler(programAssignmentService)

	clientProgramService := client_programs.NewService(programAssignmentRepository)
	clientProgramHandler := auth.NewClientProgramHandler(clientProgramService)

	entitlementRepository := repositories.NewEntitlementRepository(db)
	entitlementService := entitlements.NewService(entitlementRepository)
	entitlementHandler := auth.NewEntitlementsHandler(entitlementService)

	trainerProgramRepository := repositories.NewProgramRepository(db)
	trainerProgramService := programs.NewService(trainerProgramRepository, pricingCfg)
	trainerProgramHandler := auth.NewTrainerProgramsHandler(trainerProgramService)

	publicProgramService := public_programs.NewService(trainerProgramRepository)
	publicProgramHandler := auth.NewPublicProgramsHandler(publicProgramService)

	adminProgramPricingService := admin_program_pricing.NewService(trainerProgramService)
	adminProgramPricingHandler := auth.NewAdminProgramPricingHandler(adminProgramPricingService)

	commissionRuleRepository := repositories.NewCommissionRuleRepository(db)
	commissionRulesService := commission_rules.NewService(commissionRuleRepository, trainerRepository, commissionCfg)
	adminCommissionHandler := auth.NewAdminCommissionHandler(commissionRulesService)

	purchaseRepository := repositories.NewPurchaseRepository(db)
	stripeProvider, paypalProvider := resolvePaymentProviders(stripeCfg, paypalCfg)
	methodMap := payments.NewMethodProviderMap(stripeProvider, paypalProvider)
	purchaseService := purchases.NewService(trainerProgramRepository, purchaseRepository, entitlementRepository, &commissionAdapter{svc: commissionRulesService}, nil, methodMap.Resolve)
	purchaseHandler := auth.NewPurchaseHandler(purchaseService)

	programWeekRepository := repositories.NewProgramWeekRepository(db)
	programWorkoutRepository := repositories.NewProgramWorkoutRepository(db)
	programStructureService := program_structure.NewService(programWeekRepository, programWorkoutRepository)
	programStructureHandler := auth.NewTrainerProgramStructureHandler(programStructureService)

	workoutExerciseRepository := repositories.NewWorkoutExerciseRepository(db)
	workoutExerciseService := workout_exercises.NewService(workoutExerciseRepository)
	workoutExerciseHandler := auth.NewTrainerWorkoutExerciseHandler(workoutExerciseService)

	exerciseRepository := repositories.NewExerciseRepository(db)
	exerciseService := exercises.NewService(exerciseRepository)
	exercisesHandler := auth.NewExercisesHandler(exerciseService)

	workoutHistoryRepository := repositories.NewWorkoutHistoryRepository(db)
	workoutHistoryService := workout_history.NewService(workoutHistoryRepository)
	workoutHistoryHandler := auth.NewWorkoutHistoryHandler(workoutHistoryService)

	statisticsRepository := repositories.NewStatisticsRepository(db)
	statisticsService := statistics.NewService(statisticsRepository)
	statisticsHandler := auth.NewStatisticsHandler(statisticsService)

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", registerHandler.Register)
	v1.POST("/auth/login", loginHandler.Login)
	v1.POST("/auth/logout", logoutHandler.Logout)
	v1.POST("/admin/auth/login", adminLoginHandler.Login)
	v1.POST("/admin/auth/verify", adminLoginHandler.Verify)
	v1.GET("/exercises", exercisesHandler.ListExercises)
	v1.GET("/exercises/search", exercisesHandler.SearchExercises)
	v1.GET("/exercises/:exerciseID", exercisesHandler.GetExercise)
	v1.GET("/programs", publicProgramHandler.ListPublishedPrograms)
	v1.GET("/programs/:programID", publicProgramHandler.GetPublishedProgram)
	v1.POST("/auth/change-password", middleware.Authenticate(tokenService, userRepository), changePasswordHandler.ChangePassword)
	v1.POST("/auth/delete-account", middleware.Authenticate(tokenService, userRepository), deleteAccountHandler.DeleteAccount)
	v1.GET("/me", middleware.Authenticate(tokenService, userRepository), meHandler.GetMe)
	v1.GET("/me/program", middleware.Authenticate(tokenService, userRepository), clientProgramHandler.GetProgram)
	v1.GET("/me/entitlements", middleware.Authenticate(tokenService, userRepository), entitlementHandler.ListEntitlements)
	v1.POST("/me/programs/:programID/purchase", middleware.Authenticate(tokenService, userRepository), purchaseHandler.CreatePurchase)
	v1.POST("/me/purchases/:purchaseID/payment", middleware.Authenticate(tokenService, userRepository), purchaseHandler.InitiatePayment)
	v1.POST("/me/workouts/:workoutID/complete", middleware.Authenticate(tokenService, userRepository), workoutHistoryHandler.CompleteWorkout)
	v1.GET("/me/workouts/history", middleware.Authenticate(tokenService, userRepository), workoutHistoryHandler.ListHistory)
	v1.GET("/me/statistics", middleware.Authenticate(tokenService, userRepository), statisticsHandler.GetStatistics)
	v1.POST("/trainer/apply", middleware.Authenticate(tokenService, userRepository), trainerApplicationHandler.Apply)

	trainer := v1.Group("/trainer")
	trainer.Use(middleware.Authenticate(tokenService, userRepository))
	trainer.Use(middleware.TrainerAuthenticate(trainerRepository))
	trainer.GET("/profile", middleware.RequireTrainerPermission(trainerroles.PermissionProfile), trainerProfileHandler.GetProfile)
	trainer.GET("/clients", middleware.RequireTrainerPermission(trainerroles.PermissionClients), trainerClientHandler.ListClients)
	trainer.GET("/clients/:userID", middleware.RequireTrainerPermission(trainerroles.PermissionClients), trainerClientHandler.GetClient)
	trainer.POST("/clients", middleware.RequireTrainerPermission(trainerroles.PermissionClients), trainerClientHandler.AddClient)
	trainer.DELETE("/clients/:userID", middleware.RequireTrainerPermission(trainerroles.PermissionClients), trainerClientHandler.RemoveClient)
	trainer.POST("/clients/:userID/reactivate", middleware.RequireTrainerPermission(trainerroles.PermissionClients), trainerClientHandler.ReactivateClient)
	trainer.POST("/clients/:userID/programs", middleware.RequireTrainerPermission(trainerroles.PermissionClients), programAssignmentHandler.AssignProgram)
	trainer.GET("/clients/:userID/programs", middleware.RequireTrainerPermission(trainerroles.PermissionClients), programAssignmentHandler.ListClientPrograms)
	trainer.DELETE("/clients/:userID/programs/:assignmentID", middleware.RequireTrainerPermission(trainerroles.PermissionClients), programAssignmentHandler.RemoveAssignment)
	trainer.GET("/programs", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), trainerProgramHandler.ListPrograms)
	trainer.POST("/programs", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), trainerProgramHandler.CreateProgram)
	trainer.GET("/programs/:programID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), trainerProgramHandler.GetProgram)
	trainer.PATCH("/programs/:programID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), trainerProgramHandler.UpdateProgram)
	trainer.POST("/programs/:programID/publish", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), trainerProgramHandler.PublishProgram)
	trainer.DELETE("/programs/:programID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), trainerProgramHandler.DeleteProgram)
	trainer.POST("/programs/:programID/weeks", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.CreateWeek)
	trainer.GET("/programs/:programID/weeks", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.ListWeeks)
	trainer.PATCH("/programs/:programID/weeks/order", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.ReorderWeeks)
	trainer.GET("/programs/:programID/weeks/:weekID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.GetWeek)
	trainer.DELETE("/programs/:programID/weeks/:weekID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.DeleteWeek)
	trainer.POST("/programs/:programID/weeks/:weekID/workouts", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.CreateWorkout)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.ListWorkouts)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/order", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.ReorderWorkouts)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.GetWorkout)
	trainer.DELETE("/programs/:programID/weeks/:weekID/workouts/:workoutID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), programStructureHandler.DeleteWorkout)
	trainer.POST("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), workoutExerciseHandler.AddExercise)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), workoutExerciseHandler.ListExercises)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/order", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), workoutExerciseHandler.ReorderExercises)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), workoutExerciseHandler.GetExercise)
	trainer.DELETE("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", middleware.RequireTrainerPermission(trainerroles.PermissionPrograms), workoutExerciseHandler.DeleteExercise)

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

	adminApplicationRead := admin.Group("")
	adminApplicationRead.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainerApplicationsRead))
	adminApplicationRead.GET("/trainer-applications", adminTrainerApplicationHandler.ListApplications)
	adminApplicationRead.GET("/trainer-applications/:id", adminTrainerApplicationHandler.GetApplication)

	adminApplicationMutate := admin.Group("")
	adminApplicationMutate.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainerApplicationsManage))
	adminApplicationMutate.POST("/trainer-applications/:id/approve", adminTrainerApplicationHandler.ApproveApplication)
	adminApplicationMutate.POST("/trainer-applications/:id/reject", adminTrainerApplicationHandler.RejectApplication)

	adminPricing := admin.Group("")
	adminPricing.Use(middleware.RequireAdminPermission(adminroles.PermissionPlans))
	adminPricing.GET("/programs/:programID", adminProgramPricingHandler.GetProgram)
	adminPricing.PATCH("/programs/:programID/pricing", adminProgramPricingHandler.UpdatePricing)

	adminCommission := admin.Group("")
	adminCommission.Use(middleware.RequireAdminPermission(adminroles.PermissionPlansCommissionManage))
	adminCommission.GET("/trainers/:id/commission", adminCommissionHandler.GetCommissionRule)
	adminCommission.PATCH("/trainers/:id/commission", adminCommissionHandler.UpsertCommissionRule)
	adminCommission.DELETE("/trainers/:id/commission", adminCommissionHandler.DeleteCommissionRule)
	adminCommission.GET("/trainers/:id/commission/resolve", adminCommissionHandler.GetCommissionResolution)

	if webhookCfg.StripeWebhookSecret != "" {
		stripeWebhookHandler := webhooks.NewStripeWebhookHandler(webhookCfg.StripeWebhookSecret, purchaseService)
		v1.POST("/webhooks/stripe", stripeWebhookHandler.Handle)
	}
	if webhookCfg.PayPalWebhookID != "" && paypalCfg.ClientID != "" {
		paypalVerifier, err := createPayPalWebhookVerifier(paypalCfg)
		if err == nil {
			paypalWebhookHandler := webhooks.NewPayPalWebhookHandler(paypalVerifier, webhookCfg.PayPalWebhookID, purchaseService)
			v1.POST("/webhooks/paypal", paypalWebhookHandler.Handle)
		}
	}

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

// notConfiguredPaymentProvider is a placeholder used when no real payment
// provider is configured. It returns an error for every payment initiation
// request. A real provider must be injected before enabling payment flows.
type notConfiguredPaymentProvider struct{}

func (p *notConfiguredPaymentProvider) InitiatePayment(_ context.Context, _ payments.PaymentRequest) (payments.PaymentResult, error) {
	return payments.PaymentResult{}, fmt.Errorf("no payment provider configured")
}

// resolvePaymentProviders returns the configured payment providers. When a
// valid secret key / client ID is configured the corresponding provider is
// created; otherwise nil is returned for that provider. The Stripe global key
// is set here so the provider can make API calls.
func resolvePaymentProviders(stripeCfg config.StripeConfig, paypalCfg config.PayPalConfig) (payments.Provider, payments.Provider) {
	var stripeProvider payments.Provider
	if stripeCfg.SecretKey != "" {
		stripe.Key = stripeCfg.SecretKey
		stripeProvider = payments.NewStripeProvider(stripeCfg.SuccessURL, stripeCfg.CancelURL)
	}

	var pp payments.Provider
	if paypalCfg.ClientID != "" {
		provider, err := payments.NewPayPalProvider(paypalCfg.ClientID, paypalCfg.Secret, paypalCfg.Mode)
		if err == nil {
			pp = provider
		}
	}

	return stripeProvider, pp
}

// createPayPalWebhookVerifier creates a PayPal client suitable for webhook
// signature verification. The client is separate from the payment-initiation
// provider to maintain clean separation of concerns.
func createPayPalWebhookVerifier(cfg config.PayPalConfig) (*paypal.Client, error) {
	mode := strings.TrimSpace(strings.ToLower(cfg.Mode))
	var apiBase string
	switch mode {
	case "sandbox":
		apiBase = paypal.APIBaseSandBox
	case "live":
		apiBase = paypal.APIBaseLive
	default:
		apiBase = paypal.APIBaseSandBox
	}

	client, err := paypal.NewClient(cfg.ClientID, cfg.Secret, apiBase)
	if err != nil {
		return nil, fmt.Errorf("failed to create PayPal webhook verifier: %w", err)
	}
	return client, nil
}

// commissionAdapter adapts the commission_rules.Service to the
// purchases.CommissionResolver interface, bridging the different types used
// by each package without coupling them.
type commissionAdapter struct {
	svc commission_rules.Service
}

func (a *commissionAdapter) ResolveCommission(ctx context.Context, trainerID string) (purchases.CommissionResolution, error) {
	res, err := a.svc.ResolveCommission(ctx, trainerID)
	if err != nil {
		return purchases.CommissionResolution{}, err
	}
	return purchases.CommissionResolution{
		CommissionBPS: res.CommissionBPS,
		IsOverride:    res.IsOverride,
	}, nil
}

func (a *commissionAdapter) CalculateCommissionSplit(priceMinorUnits int64, resolution purchases.CommissionResolution) purchases.CommissionCalculation {
	res := commission_rules.CommissionResolution{
		CommissionBPS: resolution.CommissionBPS,
		IsOverride:    resolution.IsOverride,
	}
	calc := a.svc.CalculateCommissionSplit(priceMinorUnits, res)
	return purchases.CommissionCalculation{
		PlatformAmount: calc.PlatformAmount,
		TrainerAmount:  calc.TrainerAmount,
	}
}
