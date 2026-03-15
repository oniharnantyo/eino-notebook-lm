// Package config handles application configuration.
//
// Configuration is loaded from environment variables with defaults.
// Uses viper for flexible configuration management.
//
// Example:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	dbPool, _ := pgxpool.New(ctx, cfg.Database.GetDSN())
//
// Environment Variables:
//
//	SERVER_HOST=localhost
//	SERVER_PORT=8080
//	DATABASE_HOST=localhost
//	GEMINI_API_KEY=your_key_here
//
// Guidelines:
//   - Support environment variables
//   - Provide sensible defaults
//   - Validate at startup
//   - Fail fast on invalid config
//   - Use struct tags for env mapping
package config
