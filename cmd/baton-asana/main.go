package main

import (
	"context"
	"fmt"
	"os"

	cfg "github.com/conductorone/baton-asana/pkg/config"
	"github.com/conductorone/baton-asana/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"

	asanaClient "github.com/conductorone/baton-asana/pkg/asana"
)

var version = "dev"

func main() {
	ctx := context.Background()

	_, cmd, err := config.DefineConfiguration(
		ctx,
		"baton-asana",
		getConnector,
		cfg.Config,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version

	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, ac *cfg.Asana) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)

	if err := cfg.ValidateConfig(ac); err != nil {
		return nil, err
	}

	useServiceAccount := ac.UseServiceAccount
	defaultWorkspaceID := ac.DefaultWorkspaceId
	useScimApi := ac.UseScimApi
	apiUrl := ac.AsanaApiUrl

	// Set custom API URL if provided
	if apiUrl != "" {
		l.Info("using custom Asana API URL", zap.String("url", apiUrl))
		asanaClient.SetBaseUrl(apiUrl)
	}

	opts := []connector.Option{
		connector.WithServiceAccount(useServiceAccount),
		connector.WithScimApi(useScimApi),
	}

	// Add default workspace ID option if provided
	if defaultWorkspaceID != "" {
		opts = append(opts, connector.WithDefaultWorkspaceID(defaultWorkspaceID))
	}

	cb, err := connector.New(
		ctx,
		ac.Token,
		opts...,
	)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	c, err := connectorbuilder.NewConnector(ctx, cb)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
