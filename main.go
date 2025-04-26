package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/treybastian/twitchlinker/pkg/service"
)

func main() {
	log.Println("Starting TwitchLinker service...")

	// Load configuration from environment variables
	config := &service.Config{
		TwitchClientID:     getEnv("TWITCH_CLIENT_ID", ""),
		TwitchClientSecret: getEnv("TWITCH_CLIENT_SECRET", ""),
		TwitchChannelName:  getEnv("TWITCH_CHANNEL_NAME", ""),
		CloudflareAPIToken: getEnv("CLOUDFLARE_API_TOKEN", ""),
		CloudflareZoneID:   getEnv("CLOUDFLARE_ZONE_ID", ""),
		CloudflareDomain:   getEnv("CLOUDFLARE_DOMAIN", ""),
		CloudflareRecord:   getEnv("CLOUDFLARE_RECORD", ""),
		WebhookPort:        getEnv("WEBHOOK_PORT", "8080"),
		WebhookSecret:      getEnv("WEBHOOK_SECRET", ""),
		WebhookURL:         getEnv("WEBHOOK_URL", ""),
		PollInterval:       time.Duration(getEnvInt("POLL_INTERVAL_SECONDS", 60)) * time.Second,
	}

	// Validate required configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create and start the service
	svc, err := service.NewService(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Start service in a goroutine
	go func() {
		if err := svc.Start(); err != nil {
			log.Fatalf("Service error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down TwitchLinker service...")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("Warning: Could not parse %s as int: %v. Using default: %d", key, err, defaultValue)
		return defaultValue
	}

	return int(intValue.Seconds())
}

func validateConfig(config *service.Config) error {
	if config.TwitchClientID == "" {
		return ErrMissingEnv("TWITCH_CLIENT_ID")
	}
	if config.TwitchClientSecret == "" {
		return ErrMissingEnv("TWITCH_CLIENT_SECRET")
	}
	if config.TwitchChannelName == "" {
		return ErrMissingEnv("TWITCH_CHANNEL_NAME")
	}
	if config.CloudflareAPIToken == "" {
		return ErrMissingEnv("CLOUDFLARE_API_TOKEN")
	}
	if config.CloudflareZoneID == "" {
		return ErrMissingEnv("CLOUDFLARE_ZONE_ID")
	}
	if config.CloudflareDomain == "" {
		return ErrMissingEnv("CLOUDFLARE_DOMAIN")
	}
	if config.CloudflareRecord == "" {
		return ErrMissingEnv("CLOUDFLARE_RECORD")
	}
	if config.WebhookSecret == "" {
		return ErrMissingEnv("WEBHOOK_SECRET")
	}
	if config.WebhookURL == "" {
		return ErrMissingEnv("WEBHOOK_URL")
	}
	return nil
}

type MissingEnvError struct {
	EnvVar string
}

func (e MissingEnvError) Error() string {
	return "required environment variable not set: " + e.EnvVar
}

func ErrMissingEnv(envVar string) error {
	return MissingEnvError{EnvVar: envVar}
}