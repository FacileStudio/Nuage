package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/version"
	"github.com/FacileStudio/Nuage/apps/api/modules/mcp"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	"github.com/FacileStudio/porte/oidc"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// newMCP builds the MCP server when MCP_RESOURCE is set and SSO is on, and
// nothing otherwise. The verifier runs Registre discovery here, so a wrong
// issuer fails at boot like the OIDC kit does. A person's first MCP call
// creates their account through the OIDC kit, as a first web login would.
func newMCP(
	ctx context.Context, sqlDB *sql.DB, appEnv env.Config, deps mcp.Deps, identities porte.IdentityStore, kit *oidc.Kit, appLogger *slog.Logger,
) (*mcp.Service, error) {
	if appEnv.OIDC == nil || appEnv.MCP.Resource == "" {
		return nil, nil
	}
	verifier, err := mcpauth.Verifier(ctx, mcpauth.Config{
		Issuer:      appEnv.OIDC.Issuer,
		Resource:    appEnv.MCP.Resource,
		ScopePrefix: "nuage",
		Identities:  identities,
		Provision:   true,
		Accounts:    kit,
		Logger:      appLogger,
	})
	if err != nil {
		return nil, err
	}
	appLogger.Info("mcp server on", slog.String("resource", appEnv.MCP.Resource))
	return mcp.NewService(deps, mcp.Config{
		Resource: appEnv.MCP.Resource,
		Issuer:   appEnv.OIDC.Issuer,
		Version:  version.String(),
		Origins:  appEnv.CORSAllowedOrigins,
		StateKey: appEnv.MCP.StateKey,
		Verifier: verifier,
		Pending:  mcpkit.PostgresPending{DB: sqlDB},
		Logger:   appLogger,
	})
}
