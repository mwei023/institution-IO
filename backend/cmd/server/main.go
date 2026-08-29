package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"

	"institution-os/internal/academics"
	"institution-os/internal/audit"
	"institution-os/internal/auth"
	"institution-os/internal/cbc"
	"institution-os/internal/communication"
	digitalocker "institution-os/internal/digital_locker"
	"institution-os/internal/finance"
	finmpesa "institution-os/internal/finance/mpesa"
	"institution-os/internal/institution"
	"institution-os/internal/notification"
	"institution-os/internal/student"
	"institution-os/pkg/config"
	"institution-os/pkg/db"

	pkglogger "institution-os/pkg/logger"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	pkglogger.Setup(cfg.LogLevel, cfg.Environment)

	log.Info().Str("environment", cfg.Environment).Str("port", cfg.Port).
		Msg("Institution OS starting")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbPool, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("database connect failed")
	}
	defer dbPool.Close()
	log.Info().Msg("database connected")

	// ── Services ──────────────────────────────────────────────────
	auditSvc := audit.NewService(dbPool)
	notificationSvc := notification.NewService()

	// Event publisher — dispatches synchronously to the notification service.
	// Phase 1: single process, no queue. Swap for Redis Streams in Phase 2
	// once volume or reliability requirements justify the added complexity.
	publisher := notification.NewEventPublisher(notificationSvc)

	authRepo := auth.NewRepository(dbPool)
	authSvc, err := auth.NewService(authRepo, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("auth service init failed")
	}

	studentRepo := student.NewRepository(dbPool)
	studentSvc := student.NewService(studentRepo, publisher)

	financeRepo := finance.NewRepository(dbPool)
	mpesaClient := finmpesa.NewClient(cfg)
	financeSvc := finance.NewService(financeRepo, mpesaClient, publisher, cfg.MpesaCallbackURL)

	academicsRepo := academics.NewRepository(dbPool)
	academicsSvc := academics.NewService(academicsRepo, publisher)

	// CBC v1.0 academic layer — deliberately separate service/handler
	// from academics, but depends on academicsRepo directly for
	// attendance/class data, which is curriculum-agnostic and already
	// correct there (see cbc/service.go doc comment for the rationale).
	cbcRepo := cbc.NewRepository(dbPool)
	cbcSvc := cbc.NewService(cbcRepo, academicsRepo, publisher)

	institutionRepo := institution.NewRepository(dbPool)
	institutionSvc := institution.NewService(institutionRepo)

	commRepo := communication.NewRepository(dbPool)
	commSvc := communication.NewService(commRepo, publisher)

	lockerSvc, err := digitalocker.NewService(dbPool, digitalocker.StorageConfig{
		Endpoint:  cfg.StorageEndpoint,
		AccessKey: cfg.StorageAccessKey,
		SecretKey: cfg.StorageSecretKey,
		Bucket:    cfg.StorageBucket,
		UseSSL:    cfg.StorageUseSSL,
	})
	if err != nil {
		log.Warn().Err(err).Msg("digital locker init failed — file upload disabled")
		lockerSvc = nil
	}

	// ── Handlers ──────────────────────────────────────────────────
	authHandler      := auth.NewHandler(authSvc)
	studentHandler   := student.NewHandler(studentSvc)
	financeHandler   := finance.NewHandler(financeSvc)
	academicsHandler := academics.NewHandler(academicsSvc)
	cbcHandler       := cbc.NewHandler(cbcSvc)

	institutionHandler := institution.NewHandler(institutionSvc)
	
	commHandler      := communication.NewHandler(commSvc)
	var lockerHandler *digitalocker.Handler
	if lockerSvc != nil {
		lockerHandler = digitalocker.NewHandler(lockerSvc)
	}

	// ── Router ────────────────────────────────────────────────────
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(audit.RequestIDMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Institution-Slug", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health
	r.Get("/health", healthHandler(dbPool))
	r.Get("/health/ready", readinessHandler(dbPool))

	// M-Pesa callback (no auth — Safaricom server)
	r.Post("/api/v1/finance/mpesa/callback", financeHandler.MpesaCallback)
	//r.Post("/api/v1/finance/mpesa/callback/{institution_slug}", financeHandler.MpesaCallback)

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/mfa/verify", authHandler.VerifyMFA)
		r.Post("/auth/refresh", authHandler.RefreshToken)
		r.Post("/auth/logout", authHandler.Logout)

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(auth.JWTMiddleware(authSvc))
			r.Use(audit.HTTPMiddleware(auditSvc))

			// Auth
			r.Get("/auth/me", authHandler.Me)
			r.Post("/auth/change-password", authHandler.ChangePassword)
			r.Get("/institution/context", institutionHandler.GetContext)

			// Admissions & Students
			r.Route("/admissions", func(r chi.Router) {
				r.With(auth.RequirePermission("students:write")).Post("/", studentHandler.CreateAdmission)
				r.With(auth.RequirePermission("students:read")).Get("/", studentHandler.ListAdmissions)
				r.With(auth.RequirePermission("students:read")).Get("/{id}", studentHandler.GetAdmission)
				r.With(auth.RequirePermission("students:write")).Post("/{id}/review", studentHandler.ReviewAdmission)
				r.With(auth.RequirePermission("students:write")).Post("/{id}/enroll", studentHandler.EnrollStudent)
			})

			r.Route("/students", func(r chi.Router) {
				r.With(auth.RequirePermission("students:read")).Get("/", studentHandler.ListStudents)
				r.With(auth.RequirePermission("students:read")).Get("/{id}", studentHandler.GetStudent)
				r.With(auth.RequirePermission("students:read")).Get("/{id}/timeline", studentHandler.GetStudentTimeline)
				r.With(auth.RequirePermission("students:read")).Get("/{id}/parents", studentHandler.GetStudentParents)
			})

			// Finance
			r.Route("/finance", func(r chi.Router) {
				r.With(auth.RequirePermission("fees:read")).Get("/students/{id}/account", financeHandler.GetFeeAccount)
				r.With(auth.RequirePermission("fees:read")).Get("/students/{id}/payments", financeHandler.GetPaymentHistory)
				r.With(auth.RequirePermission("fees:read")).Get("/summary", financeHandler.GetFeeCollectionSummary)
				r.With(auth.RequirePermission("fees:read")).Get("/outstanding", financeHandler.GetOutstandingBalances)
				r.With(auth.RequirePermission("fees:receipt")).Post("/pay/stk-push", financeHandler.InitiateSTKPush)
				r.With(auth.RequirePermission("fees:receipt")).Post("/pay/manual", financeHandler.RecordManualPayment)
				r.With(auth.RequirePermission("fees:manage")).Post("/fee-structures", financeHandler.CreateFeeStructure)
				r.With(auth.RequirePermission("fees:read")).Get("/fee-structures", financeHandler.ListFeeStructures)
				r.With(auth.RequirePermission("fees:manage")).Post("/fee-structures/{id}/components", financeHandler.AddFeeComponent)
				r.With(auth.RequirePermission("fees:manage")).Post("/invoices/generate-class", financeHandler.GenerateInvoicesForClass)
			})

			// Academics (8-4-4)
			r.Route("/academics", func(r chi.Router) {
				r.With(auth.RequirePermission("attendance:read")).Get("/classes", academicsHandler.ListClasses)
				r.With(auth.RequirePermission("attendance:read")).Get("/classes/{id}", academicsHandler.GetClass)
				r.With(auth.RequirePermission("attendance:read")).Get("/classes/{id}/timetable", academicsHandler.GetClassTimetable)
				r.With(auth.RequirePermission("attendance:read")).Get("/subjects", academicsHandler.ListSubjects)
				r.With(auth.RequirePermission("attendance:read")).Get("/terms", academicsHandler.ListTerms)
				r.With(auth.RequirePermission("attendance:read")).Get("/my-timetable", academicsHandler.GetMyTimetable)
				r.With(auth.RequirePermission("attendance:read")).Get("/attendance/sheet", academicsHandler.GetAttendanceSheet)
				r.With(auth.RequirePermission("attendance:write")).Post("/attendance", academicsHandler.SubmitAttendance)
				r.With(auth.RequirePermission("attendance:read")).Get("/attendance/summary", academicsHandler.GetAttendanceSummary)
				r.With(auth.RequirePermission("attendance:read")).Get("/attendance/today", academicsHandler.GetInstitutionAttendanceToday)
				r.With(auth.RequirePermission("grades:read")).Get("/grades/components", academicsHandler.GetGradeComponents)
				r.With(auth.RequirePermission("grades:read")).Get("/grades/entries", academicsHandler.GetGradebook)
				r.With(auth.RequirePermission("grades:write")).Post("/grades/entries", academicsHandler.SaveGrades)
				r.With(auth.RequirePermission("grades:write")).Post("/grades/components", academicsHandler.CreateGradeComponent)
				r.With(auth.RequirePermission("grades:write")).Post("/report-cards/generate", academicsHandler.GenerateReportCard)
				r.With(auth.RequirePermission("grades:write")).Post("/report-cards/generate-class", academicsHandler.GenerateReportCardsForClass)
				r.With(auth.RequirePermission("grades:publish")).Post("/report-cards/publish", academicsHandler.PublishReportCard)
				r.With(auth.RequirePermission("grades:publish")).Post("/report-cards/unpublish", academicsHandler.UnpublishReportCard)
				r.With(auth.RequirePermission("classes:write")).Post("/classes", academicsHandler.CreateClass)
				r.With(auth.RequirePermission("classes:write")).Post("/classes/{id}/assign-teacher", academicsHandler.AssignClassTeacher)
				r.With(auth.RequirePermission("classes:write")).Get("/teachers", academicsHandler.ListTeachers)
				r.With(auth.RequirePermission("classes:write")).Get("/academic-years", academicsHandler.ListAcademicYears)
			})

			// CBC v1.0 academic layer — separate route group, separate
			// package. Reuses the same grades:* permission keys as
			// academics rather than inventing new ones — no new
			// permissions/role_permissions seeding required for v1.0.
			r.Route("/cbc", func(r chi.Router) {
				r.With(auth.RequirePermission("grades:read")).Get("/learning-areas", cbcHandler.ListLearningAreas)
				r.With(auth.RequirePermission("grades:write")).Post("/learning-areas", cbcHandler.CreateLearningArea)
				r.With(auth.RequirePermission("grades:read")).Get("/components", cbcHandler.GetComponents)
				r.With(auth.RequirePermission("grades:write")).Post("/components", cbcHandler.CreateComponent)
				r.With(auth.RequirePermission("grades:read")).Get("/entries", cbcHandler.GetEntries)
				r.With(auth.RequirePermission("grades:write")).Post("/entries", cbcHandler.SaveEntries)
				r.With(auth.RequirePermission("grades:write")).Post("/teacher-assignments", cbcHandler.CreateTeacherAssignment)
				r.With(auth.RequirePermission("grades:write")).Post("/report-cards/generate-class", cbcHandler.GenerateReportCardsForClass)
				r.With(auth.RequirePermission("grades:publish")).Post("/report-cards/publish", cbcHandler.PublishReportCard)
				r.With(auth.RequirePermission("grades:publish")).Post("/report-cards/unpublish", cbcHandler.UnpublishReportCard)
				// NOT YET BUILT (v1.0 follow-up, not v1.1 Curriculum
				// Engine scope): rating-band CRUD for institution
				// overrides — v1.0 ships with seeded defaults only,
				// editable directly in the DB until this exists.
			})

			// Communications
			r.Route("/communications", func(r chi.Router) {
				r.With(auth.RequirePermission("announcements:read")).Get("/announcements", commHandler.ListAnnouncements)
				r.With(auth.RequirePermission("announcements:write")).Post("/announcements", commHandler.CreateAnnouncement)
				r.Post("/announcements/{id}/read", commHandler.MarkAnnouncementRead)
				r.Get("/messages", commHandler.GetThreads)
				r.Get("/messages/{user_id}", commHandler.GetMessages)
				r.Post("/messages", commHandler.SendMessage)
			})

			// Parent portal — role-gated, not permission-gated
			r.Route("/parent", func(r chi.Router) {
				r.Use(auth.RequireRole("parent"))
				r.Get("/children", studentHandler.GetMyChildren)
				r.Get("/children/{id}/report-card", academicsHandler.GetChildReportCard)
				r.Get("/children/{id}/cbc-report-card", cbcHandler.GetChildReportCard)
				r.Get("/children/{id}/subject-scores", academicsHandler.GetChildSubjectScores)
				r.Get("/children/{id}/account", financeHandler.GetFeeAccount)
				r.Get("/children/{id}/attendance", academicsHandler.GetAttendanceSummary) // uses class_id query param
			})

			// Digital Locker
			r.Route("/locker", func(r chi.Router) {
				r.With(auth.RequirePermission("documents:read")).Get("/{entity_type}/{entity_id}", func(w http.ResponseWriter, r *http.Request) {
					if lockerHandler != nil {
						lockerHandler.ListDocuments(w, r)
					} else {
						stubHandler("digital_locker")(w, r)
					}
				})
				r.With(auth.RequirePermission("documents:write")).Post("/{entity_type}/{entity_id}", func(w http.ResponseWriter, r *http.Request) {
					if lockerHandler != nil {
						lockerHandler.UploadDocument(w, r)
					} else {
						stubHandler("digital_locker")(w, r)
					}
				})
				r.With(auth.RequirePermission("documents:read")).Get("/documents/{id}/download", func(w http.ResponseWriter, r *http.Request) {
					if lockerHandler != nil {
						lockerHandler.DownloadDocument(w, r)
					} else {
						stubHandler("digital_locker")(w, r)
					}
				})
				r.With(auth.RequirePermission("documents:delete")).Delete("/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
					if lockerHandler != nil {
						lockerHandler.DeleteDocument(w, r)
					} else {
						stubHandler("digital_locker")(w, r)
					}
				})
			})

			// Phase 2 stubs
			r.Get("/staff", stubHandler("hr:staff"))
			r.Get("/boarding", stubHandler("boarding"))
			r.Get("/clinic", stubHandler("clinic"))
			r.Get("/library", stubHandler("library"))
			r.Get("/transport", stubHandler("transport"))
		})
	})

	// ── Server start ──────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Info().Msg("shutting down...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutCancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Error().Err(err).Msg("shutdown error")
		}
		close(done)
	}()

	log.Info().Str("addr", srv.Addr).Msg("server listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}
	<-done
	log.Info().Msg("server stopped")
}

func healthHandler(dbPool *db.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"institution-os"}`))
	}
}

func readinessHandler(dbPool *db.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := dbPool.HealthCheck(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unhealthy","database":"unreachable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","database":"ok"}`))
	}
}

func stubHandler(module string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"error":{"code":"NOT_IMPLEMENTED","message":"The %s module is not yet implemented in this phase."}}`,
			module,
		)))
	}
}