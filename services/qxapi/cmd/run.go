package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	qxlog "github.com/qxproject/qx/pkg/log"
	"github.com/qxproject/qx/services/qxapi/internal/api"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/config"
	"github.com/qxproject/qx/services/qxapi/internal/database"
)

var connectDB = database.Connect

func bootstrap(cfg config.Config) (*gin.Engine, error) {
	db, err := connectDB(cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	tokens := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authSvc := auth.NewService(db, tokens)
	return api.NewRouter(db, authSvc, cfg.CORSOrigin, cfg.SSHMasterKey, api.DeploySettings{
		PublicAPIURL:    cfg.PublicAPIURL,
		AgentBinaryPath: cfg.AgentBinaryPath,
	}, api.MojangSettings{
		ClientID:     cfg.MojangClientID,
		ClientSecret: cfg.MojangClientSecret,
		RedirectURI:  cfg.ResolvedMojangRedirectURI(),
		WebBaseURL:   cfg.CORSOrigin,
		JWTSecret:    cfg.JWTSecret,
	}, api.ModsSettings{
		CurseForgeAPIKey:  cfg.CurseForgeAPIKey,
		ModrinthUserAgent: cfg.ModrinthUserAgent,
	}, api.CosmeticsSettings{
		DataDir:             cfg.CosmeticsDataDir,
		PublicAPIURL:        cfg.PublicAPIURL,
		SkinServerPublicURL: cfg.ResolvedSkinServerPublicURL(),
	}), nil
}

func run() error {
	cfg := config.Load()
	os.Setenv("GIN_MODE", cfg.GinMode)
	qxlog.Setup(qxlog.Options{Level: cfg.LogLevel, Format: cfg.LogFormat})

	router, err := bootstrap(cfg)
	if err != nil {
		slog.Error("QXApi failed to start", "error", err)
		return err
	}
	slog.Info("QXApi listening",
		"addr", cfg.Addr,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
		"agent_binary", config.AgentBinaryAbs(cfg.AgentBinaryPath),
		"public_api_url", cfg.PublicAPIURL,
	)
	return router.Run(cfg.Addr)
}
