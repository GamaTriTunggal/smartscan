package main

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/handlers"
	"github.com/gamatritunggal/smartscan/backend/internal/metrics"
	"github.com/gamatritunggal/smartscan/backend/internal/middleware"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"gorm.io/gorm"
)

func setupRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Set maximum multipart memory (10MB) for file uploads
	r.MaxMultipartMemory = 10 << 20 // 10 MB

	// Sentry/GlitchTip error tracking middleware (captures panics and errors)
	r.Use(sentry.GinMiddleware())

	// Prometheus metrics middleware (track all requests)
	r.Use(metrics.HTTPMetrics())

	// Request size limiting middleware (prevents DoS via large payloads)
	r.Use(middleware.RequestSizeLimiter())

	// Security headers middleware (OWASP recommended)
	r.Use(middleware.SecurityHeaders())

	// CORS configuration with validated origins
	// Production: only FRONTEND_URL allowed (no localhost)
	// Development: localhost + FRONTEND_URL allowed
	allowedOrigins := cfg.GetAllowedOrigins()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           86400, // Cache preflight for 24 hours
	}))

	// CSRF Protection (validates Origin/Referer for state-changing requests)
	r.Use(middleware.CSRFProtection(allowedOrigins))

	// Audit logging for security-sensitive operations
	r.Use(middleware.AuditLogger())

	// Initialize services
	authHandler := handlers.NewAuthHandler(db, cfg)
	setupHandler := handlers.NewSetupHandler(db, cfg)
	tenantHandler := handlers.NewTenantHandler(db, cfg)
	notificationsHandler := handlers.NewNotificationsHandler(db, cfg)
	webhookSettingsHandler := handlers.NewWebhookSettingsHandler(db, cfg)
	companyContactHandler := handlers.NewCompanyContactHandler(db, cfg)
	productHandler := handlers.NewProductHandler(db, cfg)
	qrBatchHandler := handlers.NewQRBatchHandler(db, cfg)
	validationHandler := handlers.NewValidationHandler(db, cfg)
	warrantyHandler := handlers.NewWarrantyHandler(db, cfg)
	staffHandler := handlers.NewStaffHandler(db, cfg)
	dashboardHandler := handlers.NewDashboardHandler(db, cfg)
	locationHandler := handlers.NewLocationHandler(db, cfg)
	tenantLocationHandler := handlers.NewTenantLocationHandler(db, cfg)
	scanningHandler := handlers.NewScanningHandler(db, cfg)
	templateHandler := handlers.NewTemplateHandler(db, cfg)
	certificationHandler := handlers.NewCertificationHandler(db, cfg)
	socialMediaHandler := handlers.NewSocialMediaHandler(db, cfg)
	locationMasterHandler := handlers.NewLocationMasterHandler(db, cfg)
	warrantyAdminHandler := handlers.NewWarrantyAdminHandler(db, cfg)
	counterfeitHandler := handlers.NewCounterfeitHandler(db, cfg)
	geofenceHandler := handlers.NewGeofenceHandler(db, cfg)
	appSettingsHandler := handlers.NewAppSettingsHandler(db, cfg)
	uploadHandler := handlers.NewUploadHandler(db, cfg)
	themePresetHandler := handlers.NewThemePresetHandler(db, cfg)
	tenantSocialAccountHandler := handlers.NewTenantSocialAccountHandler(db, cfg)
	productSocialAccountHandler := handlers.NewProductSocialAccountHandler(db, cfg)
	productImageHandler := handlers.NewProductImageHandler(db, cfg)
	auditLogHandler := handlers.NewAuditLogHandler(db, cfg)

	// Health check
	r.GET("/health", handlers.HealthCheck)

	// Prometheus metrics endpoint — only exposed when METRICS_USER/METRICS_PASS
	// are BOTH set. No default credentials: an unconfigured endpoint stays off.
	metricsUser := os.Getenv("METRICS_USER")
	metricsPass := os.Getenv("METRICS_PASS")
	if metricsUser != "" && metricsPass != "" {
		metricsAuth := gin.BasicAuth(gin.Accounts{
			metricsUser: metricsPass,
		})
		r.GET("/metrics", metricsAuth, gin.WrapH(promhttp.Handler()))
	}

	// QR Scan redirect route (outside /api group - direct URL for QR codes)
	// GET /s/:code - records interaction and redirects to /v/:code?sig=X&ts=Y
	// Signature prevents bypass of geolocation requirement via URL manipulation
	// This implements the redirect pattern to prevent scan count increment on page refresh
	r.GET("/s/:code", middleware.RateLimiter(middleware.ScanRateLimit), validationHandler.ScanRedirect)

	// Static file serving for uploaded files
	// Uses OptionalAuth to check tenant isolation when user is logged in
	uploads := r.Group("/uploads")
	uploads.Use(middleware.OptionalAuth(cfg))
	{
		// Background images: /uploads/backgrounds/*filepath
		uploads.GET("/backgrounds/*filepath", uploadHandler.ServeBackgroundFile)
		// Product gallery images: /uploads/products/*filepath
		uploads.GET("/products/*filepath", uploadHandler.ServeProductFile)
		// Template logos: /uploads/templates/*filepath
		uploads.GET("/templates/*filepath", uploadHandler.ServeTemplateFile)
		// Counterfeit report photos: /uploads/counterfeit-reports/*filepath
		uploads.GET("/counterfeit-reports/*filepath", uploadHandler.ServeCounterfeitReportFile)
		// Payment proof images
	}

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public) - with stricter rate limiting
		// First-run setup wizard (single-shot; refuses once a company exists)
		setup := v1.Group("/setup")
		setup.Use(middleware.RateLimiter(middleware.AuthRateLimit))
		{
			setup.GET("/status", setupHandler.Status)
			setup.POST("", setupHandler.Run)
		}

		auth := v1.Group("/auth")
		auth.Use(middleware.RateLimiter(middleware.AuthRateLimit))
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}


		// Public validation routes (no auth required) - with moderate rate limiting
		public := v1.Group("/public")
		public.Use(middleware.RateLimiter(middleware.PublicRateLimit))
		{
			// Note: POST /validate was removed - scan recording now only at GET /s/:code
			public.GET("/validate-info/:code", validationHandler.GetValidationInfo)
			public.GET("/product/:code", validationHandler.GetProductInfo)
			public.POST("/scan-location", validationHandler.UpdateScanLocation)
			public.GET("/verify-scan-session", validationHandler.VerifyScanSession)
			public.GET("/template/:uuid", templateHandler.GetPublicTemplate)
			// Counterfeit report (end-user submission)
			public.POST("/counterfeit-report", counterfeitHandler.SubmitCounterfeitReport)
			// Warranty routes
			public.GET("/warranty/:code", warrantyHandler.GetWarrantyStatus)
			public.POST("/warranty/:code", warrantyHandler.RegisterWarranty)
			// Branding (app settings)
			public.GET("/branding", appSettingsHandler.GetBrandingPublic)
			public.GET("/company-contact", companyContactHandler.GetPublic)

			// WhatsApp contact (for "Chat with us" button)

			// Location data (for warranty/registration forms)
			public.GET("/locations/countries", locationMasterHandler.ListCountries)
			public.GET("/locations/provinces", locationMasterHandler.ListProvinces)
			public.GET("/locations/cities", locationMasterHandler.ListCities)

			// Theme presets (for rendering customized backgrounds)
			public.GET("/theme-presets", themePresetHandler.ListPublicThemePresets)
			public.GET("/theme-presets/:id", themePresetHandler.GetPublicThemePreset)

		}

		// Location routes (public) - with rate limiting to prevent abuse
		locations := v1.Group("/locations")
		locations.Use(middleware.RateLimiter(middleware.PublicRateLimit))
		{
			locations.GET("/countries", locationHandler.GetCountries)
			locations.GET("/provinces/:country_code", locationHandler.GetProvinces)
			locations.GET("/cities/:province_id", locationHandler.GetCities)
		}

		// Protected routes - with general rate limiting
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		protected.Use(middleware.RateLimiter(middleware.GeneralRateLimit))
		{
			// User profile
			protected.GET("/me", authHandler.GetMe)
			protected.PUT("/me", authHandler.UpdateMe)
			protected.POST("/auth/change-password", authHandler.ChangePassword)



			// Tenant routes (Tenant staff only)
			tenant := protected.Group("/tenant")
			tenant.Use(middleware.TenantOnly())
			tenant.Use(middleware.SetTenantStaffID(db)) // Sets staff_id for audit tracking
			{
				// Dashboard
				tenant.GET("/dashboard", dashboardHandler.GetTenantDashboard)
				tenant.GET("/analytics", dashboardHandler.GetAnalytics)
				tenant.GET("/heatmap", dashboardHandler.GetScanHeatmap)

				// In-app notification center
				tenant.GET("/notifications", notificationsHandler.List)
				tenant.POST("/notifications/:id/read", notificationsHandler.MarkRead)
				tenant.POST("/notifications/read-all", notificationsHandler.MarkAllRead)

				// Outbound webhook integration (admin)
				tenant.GET("/integrations/webhook", middleware.TenantAdminOnly(), webhookSettingsHandler.Get)
				tenant.PUT("/integrations/webhook", middleware.TenantAdminOnly(), webhookSettingsHandler.Update)

				// Public company contact (shown on consumer pages)
				tenant.GET("/company-contact", companyContactHandler.Get)
				tenant.PUT("/company-contact", middleware.TenantAdminOnly(), companyContactHandler.Update)

				// Tenant info
				tenant.GET("/info", tenantHandler.GetMyTenant)
				tenant.PUT("/info", middleware.TenantAdminOnly(), tenantHandler.UpdateMyTenant)

				// Products (admin-only: qc/warehouse staff have their own job pages)
				products := tenant.Group("/products")
				products.Use(middleware.TenantAdminOnly())
				{
					products.GET("", productHandler.ListProducts)
					products.GET("/:id", productHandler.GetProduct)
					products.POST("", productHandler.CreateProduct)
					products.PUT("/:id", productHandler.UpdateProduct)
					products.DELETE("/:id", productHandler.DeleteProduct)
					// NOTE: Landing appearance config moved to templates (background_config in page_templates)

					// Product Images (Gallery)
					products.GET("/:id/images", productImageHandler.List)
					products.POST("/:id/images", productImageHandler.Upload)
					products.PUT("/:id/images/:img_id", productImageHandler.Update)
					products.DELETE("/:id/images/:img_id", productImageHandler.Delete)
					products.PUT("/:id/images/:img_id/main", productImageHandler.SetMain)
					products.PUT("/:id/images/reorder", productImageHandler.Reorder)

					// Product Social Account Links (N:M)
					products.GET("/:id/social-accounts", productSocialAccountHandler.List)
					products.POST("/:id/social-accounts", productSocialAccountHandler.Link)
					products.DELETE("/:id/social-accounts/:link_id", productSocialAccountHandler.Unlink)
					products.PUT("/:id/social-accounts/reorder", productSocialAccountHandler.Reorder)
				}

				// QR Batches (admin-only)
				batches := tenant.Group("/qr-batches")
				batches.Use(middleware.TenantAdminOnly())
				{
					// Async generation endpoints (MUST come before /:id to avoid path conflicts in Gin)
					batches.GET("/active-generations", qrBatchHandler.GetActiveGenerations)

					batches.GET("", qrBatchHandler.ListQRBatches)
					batches.GET("/:id", qrBatchHandler.GetQRBatch)
					batches.POST("", qrBatchHandler.CreateQRBatch)
					batches.PUT("/:id", qrBatchHandler.UpdateQRBatch)
					batches.DELETE("/:id", qrBatchHandler.DeleteQRBatch)
					batches.PUT("/:id/restore", qrBatchHandler.RestoreQRBatch)
					batches.GET("/:id/codes", qrBatchHandler.ListQRCodes)
					batches.GET("/:id/codes/:codeId", qrBatchHandler.GetQRCodeDetail)
					batches.GET("/:id/export/csv", qrBatchHandler.ExportQRCodesCSV)
					batches.GET("/:id/export/excel", qrBatchHandler.ExportQRCodesExcel)
					batches.GET("/:id/export/pdf", qrBatchHandler.ExportQRCodesPDF)
					batches.GET("/:id/heatmap", qrBatchHandler.GetBatchHeatmap)
					batches.GET("/:id/analytics", qrBatchHandler.GetBatchAnalytics)
					// Async generation: per-batch status + retry
					batches.GET("/:id/generation-status", qrBatchHandler.GetGenerationStatus)
					batches.POST("/:id/retry-generation", qrBatchHandler.RetryFailedGeneration)
					// Geofence violations for specific batch
					batches.GET("/:id/geofence-violations", geofenceHandler.GetBatchGeofenceViolations)
					batches.GET("/:id/geofence-analytics", geofenceHandler.GetBatchGeofenceAnalytics)
				}

				// Uploads (Admin only, Intermediate+ tier)
				tenantUploads := tenant.Group("/uploads")
				tenantUploads.Use(middleware.TenantAdminOnly())
				{
					tenantUploads.GET("/backgrounds", uploadHandler.ListTenantBackgrounds)
					tenantUploads.POST("/background", uploadHandler.UploadBackground)
					tenantUploads.DELETE("/background/:filename", uploadHandler.DeleteBackground)
				}

				// Theme Presets (Tenant - read only for selection)
				tenant.GET("/theme-presets", themePresetHandler.ListTenantThemePresets)

				// Theme preset management (admin)
				themePresets := tenant.Group("/theme-presets")
				themePresets.Use(middleware.TenantAdminOnly())
				{
					themePresets.GET("/manage", themePresetHandler.ListThemePresets)
					themePresets.GET("/manage/:id", themePresetHandler.GetThemePreset)
					themePresets.POST("", themePresetHandler.CreateThemePreset)
					themePresets.POST("/upload", uploadHandler.UploadPresetBackground)
					themePresets.PUT("/reorder", themePresetHandler.ReorderThemePresets)
					themePresets.PUT("/:id", themePresetHandler.UpdateThemePreset)
					themePresets.DELETE("/:id", themePresetHandler.DeleteThemePreset)
					themePresets.POST("/:id/restore", themePresetHandler.RestoreThemePreset)
				}

				// Application settings / branding (admin)
				adminSettings := tenant.Group("/app-settings")
				adminSettings.Use(middleware.TenantAdminOnly())
				{
					adminSettings.GET("/branding", appSettingsHandler.GetBranding)
					adminSettings.PUT("/branding", appSettingsHandler.UpdateBranding)
				}

				// Audit logs (admin)
				tenant.GET("/audit-logs", middleware.TenantAdminOnly(), auditLogHandler.ListAuditLogs)
				tenant.GET("/audit-logs/stats", middleware.TenantAdminOnly(), auditLogHandler.GetAuditLogStats)

				// Location master data (admin)
				locationMaster := tenant.Group("/location-master")
				locationMaster.Use(middleware.TenantAdminOnly())
				{
					locationMaster.GET("/countries", locationMasterHandler.ListCountries)
					locationMaster.GET("/countries/:code", locationMasterHandler.GetCountry)
					locationMaster.POST("/countries", locationMasterHandler.CreateCountry)
					locationMaster.PUT("/countries/:code", locationMasterHandler.UpdateCountry)
					locationMaster.DELETE("/countries/:code", locationMasterHandler.DeleteCountry)
					locationMaster.POST("/countries/:code/restore", locationMasterHandler.RestoreCountry)
					locationMaster.GET("/provinces", locationMasterHandler.ListProvinces)
					locationMaster.GET("/provinces/:id", locationMasterHandler.GetProvince)
					locationMaster.POST("/provinces", locationMasterHandler.CreateProvince)
					locationMaster.PUT("/provinces/:id", locationMasterHandler.UpdateProvince)
					locationMaster.DELETE("/provinces/:id", locationMasterHandler.DeleteProvince)
					locationMaster.POST("/provinces/:id/restore", locationMasterHandler.RestoreProvince)
					locationMaster.GET("/cities", locationMasterHandler.ListCities)
					locationMaster.GET("/cities/:id", locationMasterHandler.GetCity)
					locationMaster.POST("/cities", locationMasterHandler.CreateCity)
					locationMaster.PUT("/cities/:id", locationMasterHandler.UpdateCity)
					locationMaster.DELETE("/cities/:id", locationMasterHandler.DeleteCity)
					locationMaster.POST("/cities/:id/restore", locationMasterHandler.RestoreCity)
				}

				// Staff management (Admin only)
				staff := tenant.Group("/staff")
				staff.Use(middleware.TenantAdminOnly())
				{
					staff.GET("", staffHandler.ListTenantStaff)
					staff.POST("", staffHandler.CreateTenantStaff)
					staff.PUT("/:id", staffHandler.UpdateTenantStaff)
					staff.DELETE("/:id", staffHandler.DeleteTenantStaff)
					staff.POST("/:staff_id/reset-password", tenantHandler.ResetTenantStaffPassword)
				}

				// Tenant Social Accounts (Admin only)
				// These are tenant-owned social media accounts that can be linked to products
				socialAccounts := tenant.Group("/social-accounts")
				socialAccounts.Use(middleware.TenantAdminOnly())
				{
					socialAccounts.GET("", tenantSocialAccountHandler.List)
					socialAccounts.GET("/:id", tenantSocialAccountHandler.Get)
					socialAccounts.POST("", tenantSocialAccountHandler.Create)
					socialAccounts.PUT("/:id", tenantSocialAccountHandler.Update)
					socialAccounts.DELETE("/:id", tenantSocialAccountHandler.Delete)
				}

				// Locations management (Admin only, Intermediate+ tier)
				tenantLocations := tenant.Group("/locations")
				tenantLocations.Use(middleware.TenantAdminOnly())
				{
					tenantLocations.GET("", tenantLocationHandler.GetLocations)
					tenantLocations.GET("/:id", tenantLocationHandler.GetLocation)
					tenantLocations.POST("", tenantLocationHandler.CreateLocation)
					tenantLocations.PUT("/:id", tenantLocationHandler.UpdateLocation)
					tenantLocations.DELETE("/:id", tenantLocationHandler.DeleteLocation)
					tenantLocations.POST("/:id/restore", tenantLocationHandler.RestoreLocation)
					tenantLocations.GET("/type/:type", tenantLocationHandler.GetLocationsByType)
				}

				// QC Scanning routes (QC Staff and Admin)
				qcRoutes := tenant.Group("/qc")
				qcRoutes.Use(middleware.QCStaffOrAdmin())
				{
					qcRoutes.POST("/scan", scanningHandler.QCScan)
					qcRoutes.GET("/history", scanningHandler.GetQCHistory)
					qcRoutes.GET("/locations", scanningHandler.GetQCLocations)
				}

				// Warehouse Scanning routes (Warehouse Staff and Admin)
				warehouseRoutes := tenant.Group("/warehouse")
				warehouseRoutes.Use(middleware.WarehouseStaffOrAdmin())
				{
					warehouseRoutes.POST("/scan", scanningHandler.WarehouseScan)
					warehouseRoutes.GET("/history", scanningHandler.GetWarehouseHistory)
					warehouseRoutes.GET("/stock", scanningHandler.GetInventoryStock)
					warehouseRoutes.GET("/locations", scanningHandler.GetWarehouseLocations)
				}

				// Page Templates (Admin only)
				templates := tenant.Group("/templates")
				templates.Use(middleware.TenantAdminOnly())
				{
					templates.GET("", templateHandler.ListTemplates)
					templates.GET("/defaults", templateHandler.GetTenantDefaults)
					templates.GET("/:id", templateHandler.GetTemplate)
					templates.POST("", templateHandler.CreateTemplate)
					templates.PUT("/:id", templateHandler.UpdateTemplate)
					templates.DELETE("/:id", templateHandler.DeleteTemplate)
					templates.POST("/:id/set-default", templateHandler.SetAsDefault)
				templates.POST("/:id/logo", templateHandler.UploadTemplateLogo)
				templates.DELETE("/:id/logo", templateHandler.DeleteTemplateLogo)
				}

				// Product Certifications (Admin only)
				certRoutes := tenant.Group("/certifications")
				certRoutes.Use(middleware.TenantAdminOnly())
				{
					// Get available certification types for dropdown
					certRoutes.GET("/types", certificationHandler.GetAvailableCertificationTypes)

					// Certification type master data (admin-managed)
					certRoutes.GET("/types/all", certificationHandler.ListCertificationTypes)
					certRoutes.GET("/types/:id", certificationHandler.GetCertificationType)
					certRoutes.POST("/types", certificationHandler.CreateCertificationType)
					certRoutes.PUT("/types/:id", certificationHandler.UpdateCertificationType)
					certRoutes.DELETE("/types/:id", certificationHandler.DeleteCertificationType)
					certRoutes.POST("/types/:id/restore", certificationHandler.RestoreCertificationType)

					// Product certifications management
					certRoutes.GET("/products/:product_id", certificationHandler.GetProductCertifications)
					certRoutes.POST("/products/:product_id", certificationHandler.AddProductCertification)
					certRoutes.PUT("/products/:product_id/:cert_id", certificationHandler.UpdateProductCertification)
					certRoutes.DELETE("/products/:product_id/:cert_id", certificationHandler.RemoveProductCertification)
					certRoutes.PUT("/products/:product_id/reorder", certificationHandler.ReorderProductCertifications)

					// Request new certification type
				}

				// Product Social Media Links (Admin only)
				socialRoutes := tenant.Group("/social-media")
				socialRoutes.Use(middleware.TenantAdminOnly())
				{
					// Get available platforms for dropdown
					socialRoutes.GET("/platforms", socialMediaHandler.GetAvailableSocialMediaPlatforms)

					// Product social links management
					socialRoutes.GET("/products/:product_id", socialMediaHandler.GetProductSocialLinks)
					socialRoutes.POST("/products/:product_id", socialMediaHandler.AddProductSocialLink)
					socialRoutes.PUT("/products/:product_id/:link_id", socialMediaHandler.UpdateProductSocialLink)
					socialRoutes.DELETE("/products/:product_id/:link_id", socialMediaHandler.RemoveProductSocialLink)

					// Social platform master data (admin-managed)
					socialRoutes.GET("/platforms/all", socialMediaHandler.ListSocialMediaPlatforms)
					socialRoutes.POST("/platforms", socialMediaHandler.CreateSocialMediaPlatform)
					socialRoutes.PUT("/platforms/:id", socialMediaHandler.UpdateSocialMediaPlatform)
					socialRoutes.DELETE("/platforms/:id", socialMediaHandler.DeleteSocialMediaPlatform)
					socialRoutes.POST("/platforms/:id/restore", socialMediaHandler.RestoreSocialMediaPlatform)
				}

	



				// Warranty Admin routes (After Sales Staff or Admin, Intermediate+ tier)
				warranties := tenant.Group("/warranties")
				warranties.Use(middleware.TenantAdminOnly())
				{
					// List all warranty activations
					warranties.GET("", warrantyAdminHandler.ListWarranties)
					// Get warranty statistics
					warranties.GET("/stats", warrantyAdminHandler.GetWarrantyStats)
					// Export warranties to CSV (rate limited)
					warranties.GET("/export", middleware.RateLimiterByUserID(middleware.ExportRateLimit), warrantyAdminHandler.ExportWarrantiesToCSV)
					// Get single warranty detail
					warranties.GET("/:id", warrantyAdminHandler.GetWarrantyDetail)
				}

				// Counterfeit Detection routes (Admin only, Intermediate+ tier)
				counterfeit := tenant.Group("/counterfeit")
				counterfeit.Use(middleware.TenantAdminOnly())
				{
					// List counterfeit detections
					counterfeit.GET("", counterfeitHandler.ListCounterfeitDetections)
					// Get counterfeit stats for dashboard
					counterfeit.GET("/stats", counterfeitHandler.GetCounterfeitStats)
					// Get single detection with interactions and velocity data
					counterfeit.GET("/:id", counterfeitHandler.GetCounterfeitDetection)
					// Get geolocations for map visualization
					counterfeit.GET("/:id/geolocations", counterfeitHandler.GetCounterfeitGeolocations)
					// Resolve detection
	
					// Mark as false positive
					counterfeit.POST("/:id/false-positive", counterfeitHandler.MarkAsFalsePositive)
					// Override threshold (false positive with new threshold)
					counterfeit.POST("/:id/override-threshold", counterfeitHandler.OverrideThreshold)
					// Get counterfeit threshold settings
					counterfeit.GET("/settings", counterfeitHandler.GetCounterfeitSettings)
					// Update counterfeit threshold settings
					counterfeit.PUT("/settings", counterfeitHandler.UpdateCounterfeitSettings)
					// Counterfeit reports from end-users
					counterfeit.GET("/reports", counterfeitHandler.ListCounterfeitReports)
					counterfeit.GET("/reports/stats", counterfeitHandler.GetCounterfeitReportStats)
					counterfeit.GET("/reports/:id", counterfeitHandler.GetCounterfeitReport)
				}


				// Geofence Distribution Zone routes (Admin only, Intermediate+ tier)
				geofence := tenant.Group("/geofence")
				geofence.Use(middleware.TenantAdminOnly())
				{
					geofence.GET("/areas", geofenceHandler.GetGeofenceAreas)
					geofence.GET("/violations", geofenceHandler.ListGeofenceViolations)
					geofence.GET("/stats", geofenceHandler.GetGeofenceStats)
					geofence.GET("/violations/export", geofenceHandler.ExportGeofenceViolations)
					geofence.GET("/map-data", geofenceHandler.GetGeofenceMapData)

					// Analytics (Pro tier only)
					geofence.GET("/analytics", geofenceHandler.GetGeofenceAnalytics)

					// Zone templates (Pro tier only)
					zoneTemplates := geofence.Group("/zone-templates")
						{
						zoneTemplates.GET("", geofenceHandler.ListZoneTemplates)
						zoneTemplates.POST("", geofenceHandler.CreateZoneTemplate)
						zoneTemplates.PUT("/:id", geofenceHandler.UpdateZoneTemplate)
						zoneTemplates.DELETE("/:id", geofenceHandler.DeleteZoneTemplate)
					zoneTemplates.POST("/:id/restore", geofenceHandler.RestoreZoneTemplate)
					}
				}







			}

		}
	}

	// ========================

	// ========================

	return r
}
