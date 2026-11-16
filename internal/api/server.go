package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api/consumer"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api/handler"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api/middleware"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB            *gorm.DB
	BudgetRepo    repository.BudgetRepository
	PaymentRepo   repository.PaymentRepository
	Outbox        messaging.OutboxRepository
	Gateway       repository.PaymentGatewayService
	Notifications repository.NotificationService
}

type Server struct {
	router          *gin.Engine
	config          *configs.Config
	dependencies    Dependencies
	billingConsumer *consumer.BillingConsumer
	httpServer      *http.Server
}

func NewServer(cfg *configs.Config, dependencies Dependencies) *Server {
	router := gin.New()
	router.Use(
		gin.CustomRecovery(recoverWithLog),
		otelgin.Middleware(cfg.ServiceName),
		middleware.RequestID(),
		middleware.RequestLogger(),
	)

	server := &Server{
		router:       router,
		config:       cfg,
		dependencies: dependencies,
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%s", cfg.ServerPort),
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	server.registerRoutes()

	return server
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) Run() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func recoverWithLog(c *gin.Context, recovered any) {
	slog.ErrorContext(c.Request.Context(), "http.panic",
		slog.String("event", "http.panic"),
		slog.Any("panic", recovered),
	)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func (s *Server) RegisterMessageHandlers(registrar consumer.Registrar) {
	s.billingConsumer.RegisterHandlers(registrar)
}

func (s *Server) registerRoutes() {
	budgetRepo := s.dependencies.BudgetRepo
	paymentRepo := s.dependencies.PaymentRepo
	gateway := s.dependencies.Gateway
	notifications := s.dependencies.Notifications
	outbox := s.dependencies.Outbox

	getPaymentUC := payment.NewGetPaymentUseCase(paymentRepo)
	settlePaymentUC := payment.NewSettlePaymentUseCase(paymentRepo)

	budgetHandler := handler.NewBudgetHandler(
		budget.NewApproveBudgetUseCase(budgetRepo),
		budget.NewRejectBudgetUseCase(budgetRepo),
		budget.NewGetBudgetUseCase(budgetRepo),
		budget.NewListBudgetsUseCase(budgetRepo),
		getPaymentUC,
	)
	paymentHandler := handler.NewPaymentHandler(getPaymentUC, settlePaymentUC)

	s.billingConsumer = consumer.NewBillingConsumer(
		budget.NewGenerateBudgetUseCase(budgetRepo, notifications),
		budget.NewCancelBudgetUseCase(budgetRepo, outbox),
		payment.NewRequestPaymentUseCase(budgetRepo, paymentRepo, gateway, notifications),
		payment.NewRefundPaymentUseCase(paymentRepo, gateway, outbox),
	)

	s.router.GET("/health", s.health)
	s.router.GET("/ready", s.ready)

	v1 := s.router.Group("/api/v1")
	authenticated := v1.Group("", middleware.Auth(s.config.JWTSecret))
	operators := authenticated.Group("", middleware.RequireRole(middleware.RoleOperator))

	budgetHandler.RegisterRoutes(authenticated)
	paymentHandler.RegisterOperatorRoutes(operators)
	if s.config.PaymentSimulationEnabled() {
		paymentHandler.RegisterSimulationRoutes(authenticated)
	}
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) ready(c *gin.Context) {
	sqlDB, err := s.dependencies.DB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "database unavailable"})
		return
	}
	if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "database unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
