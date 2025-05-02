package main

import (
	"context"
	"fmt"
	"os"

	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/spf13/viper"

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
		field.Configuration{
			Fields: ConfigurationFields,
		},
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

func getConnector(ctx context.Context, v *viper.Viper) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)

	if err := ValidateConfig(v); err != nil {
		return nil, err
	}

	useServiceAccount := v.GetBool(UseServiceAccountField.FieldName)
	defaultWorkspaceID := v.GetString(DefaultWorkspaceIDField.FieldName)
	useScimApi := v.GetBool(UseScimApiField.FieldName)
	apiUrl := v.GetString(AsanaApiUrlField.FieldName)

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
		v.GetString(TokenField.FieldName),
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
