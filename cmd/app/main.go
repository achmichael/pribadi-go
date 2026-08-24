package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
    "database/sql"
    _ "github.com/lib/pq"

	"github.com/achmichael/pribadi-go/internal/classifier"
	"github.com/achmichael/pribadi-go/internal/config"
	"github.com/achmichael/pribadi-go/internal/conversation"
	contextpkg "github.com/achmichael/pribadi-go/internal/context"
	"github.com/achmichael/pribadi-go/internal/delivery/scheduler"
	"github.com/achmichael/pribadi-go/internal/delivery/rest"
	"github.com/achmichael/pribadi-go/internal/tools"
	"github.com/achmichael/pribadi-go/internal/delivery/webhook"
	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/identity"
	"github.com/achmichael/pribadi-go/internal/logger"
	"github.com/achmichael/pribadi-go/internal/prompt"
	"github.com/achmichael/pribadi-go/internal/reasoning"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/response"
	"github.com/achmichael/pribadi-go/internal/usecase"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/internal/usecase/reminder"
    authrepo "github.com/achmichael/pribadi-go/internal/auth/repository"
    authinfra "github.com/achmichael/pribadi-go/internal/auth/infrastructure"
    authuc "github.com/achmichael/pribadi-go/internal/auth/usecase"
    authhttp "github.com/achmichael/pribadi-go/internal/auth/transport/http"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/achmichael/pribadi-go/pkg/utils"
)

type messageRouterAdapter struct {
	orchestrator usecase.Orchestrator
}

func (a *messageRouterAdapter) TextMessage(ctx context.Context, msg whatsapp.IncomingMessage) error {
	return a.orchestrator.Handle(ctx, msg)
}

func (a *messageRouterAdapter) VoiceMessage(ctx context.Context, msg whatsapp.IncomingMessage) error {
	return a.orchestrator.Handle(ctx, msg)
}

func (a *messageRouterAdapter) MediaMessage(ctx context.Context, msg whatsapp.IncomingMessage) error {
	return a.orchestrator.Handle(ctx, msg)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	sqlite, err := repository.NewSQLiteRepository(cfg.SQLitePath, "db/schema.sql")
	if err != nil {
		log.Fatal().Err(err).Msg("Repo init failed")
	}

	vecRepo, err := repository.NewVectorRepository(cfg.QdrantAddr, cfg.OllamaBaseURL, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Vector repo failed")
	}

	fileRepo, _ := repository.NewFileRepository(cfg.LocalDiskPath)
	toolRegistry := tools.NewRegistry()
	ollamaClient := ollama.NewClient(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.OllamaNumCtx, cfg.OllamaNumPredict, log.Logger, toolRegistry)

	memory, err := factmemory.NewMemoryManager(sqlite, ollamaClient, cfg.QdrantAddr, cfg.OllamaBaseURL, log.Logger)

	// Pre-warm both models so first user request doesn't pay cold-start penalty.
	go func() {
		log.Info().Msg("Warming up embedding model...")
		embedder := utils.NewOllamaEmbedder(cfg.OllamaBaseURL, "nomic-embed-text")
		if err := embedder.Warmup(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Embedding warmup failed")
		} else {
			log.Info().Msg("Embedding model warmed up")
		}

		log.Info().Msg("Warming up chat model...")
		if err := ollamaClient.Warmup(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Chat model warmup failed")
		} else {
			log.Info().Msg("Chat model warmed up")
		}
	}()

	waClient, err := whatsapp.NewClient(sqlite.GetDB(), log.Logger) // simplified
	if err != nil {
		log.Fatal().Err(err).Msg("WA client failed")
	}

	transcription := usecase.NewTranscriptionService(fileRepo, cfg.WhisperBinPath, cfg.WhisperModelPath, log.Logger)
	extraction := usecase.NewExtractionService(log.Logger, cfg.FlorenceURL)
	ragIngest := rag.NewIngestionService(vecRepo, log.Logger)
	ragRetrieve := rag.NewRetrievalService(vecRepo, log.Logger)

	notification := usecase.NewNotificationService(waClient, sqlite, cfg.NotifyJIDs, log.Logger)
	_ = usecase.NewProjectService(sqlite)
	_ = reminder.NewReminderService(sqlite)

	// Dashboard API
	dashboardRepo := repository.NewDashboardRepository(sqlite.GetDB())
	webChatRepo := repository.NewWebChatRepository(sqlite.GetDB())
	dashboardService := usecase.NewDashboardService(dashboardRepo, cfg.DashboardJWT, log.Logger)
	
	resolver := identity.NewResolver(sqlite, log.Logger)
	refResolver := usecase.NewReferenceResolver(ollamaClient, sqlite, log.Logger)
	embedder := utils.NewOllamaEmbedder(cfg.OllamaBaseURL, "nomic-embed-text")
	
	// ── Pipeline Components ────────────────────────────────────────
	stateManager := conversation.NewStateManager(sqlite, log.Logger)
	prefManager := conversation.NewPreferenceManager(sqlite, log.Logger)
	
	intentClassifier := classifier.NewIntentClassifier(ollamaClient, log.Logger)
	
	contextBuilder := contextpkg.NewBuilder(stateManager, prefManager, memory, ragRetrieve, sqlite, dashboardRepo, ollamaClient, log.Logger)
	promptBuilder := prompt.NewBuilder()
	promptComposer := prompt.NewComposer(promptBuilder, log.Logger)
	
	verifier := reasoning.NewVerifier(ollamaClient, sqlite, memory, log.Logger)
	reflector := reasoning.NewReflector(ollamaClient, log.Logger)
	
	responseProcessor := response.NewProcessor(stateManager, memory, sqlite, log.Logger)
	interactionLogger := response.NewInteractionLogger(sqlite, log.Logger)
	
	orchestrator := usecase.NewOrchestrator(
		waClient,
		transcription,
		extraction,
		ragIngest,
		ragRetrieve,
		ollamaClient,
		sqlite,
		dashboardRepo,
		memory,
		resolver,
		refResolver,
		embedder,
		stateManager,
		prefManager,
		intentClassifier,
		contextBuilder,
		promptBuilder,
		promptComposer,
		verifier,
		reflector,
		responseProcessor,
		interactionLogger,
		log.Logger,
	)

	// WhatsApp Event Listener
	routerAdapter := &messageRouterAdapter{orchestrator: orchestrator}
	eventListener := whatsapp.NewEventListener(waClient.GetClient(), routerAdapter, log.Logger)
	eventListener.Start()

	// Webhook
	whHandler := webhook.NewHandler(notification, cfg.WebhookSecret, log.Logger)
	whServer := webhook.NewServer(whHandler, log.Logger, cfg.AppPort)

	// WebChat Service
	webChatService := usecase.NewWebChatService(
		webChatRepo,
		ollamaClient,
		ragIngest,
		ragRetrieve,
		extraction,
		cfg.EncryptionKey,
		log.Logger,
	)

	// Create default user if not exists
	_ = dashboardService.CreateDefaultUser(context.Background(), "admin", "admin123")
	
	dashboardServer := rest.NewServer(dashboardService, webChatService, cfg.DashboardJWT, log.Logger, cfg.DashboardPort)

    // Auth Module setup
    pgDB, err := sql.Open("postgres", cfg.PostgresDSN)
    if err != nil {
        log.Fatal().Err(err).Msg("Postgres connection failed")
    }
    defer pgDB.Close()
    
    authRepo := authrepo.NewPostgresAuthRepository(pgDB)
    otpSender := &authinfra.MockOTPSender{} // Or whatsapp sender
    tokenSvc := &authinfra.TokenService{Secret: cfg.DashboardJWT} // reuse or new secret
    
    authUsecase := authuc.NewAuthUsecase(authRepo, otpSender, tokenSvc)
    authHandler := authhttp.NewAuthHandler(authUsecase)
    
    // Mount to dashboard router or create new one
    authhttp.RegisterAuthRoutes(dashboardServer.Router(), authHandler, tokenSvc)

	// Scheduler
	sched, _ := scheduler.NewReminderScheduler(waClient, sqlite, log.Logger)
	dashboardSched := scheduler.NewDashboardCronScheduler(dashboardRepo, waClient, log.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	go whServer.Start(ctx)
	go dashboardServer.Start(ctx)
	go sched.Start(ctx)
	go dashboardSched.Start(ctx)

	log.Info().Msg("Application started")

	go func() {
		if err := waClient.Connect(ctx); err != nil {
			log.Error().Err(err).Msg("WhatsApp connection failed")
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	cancel()
	sqlite.Close()
}
