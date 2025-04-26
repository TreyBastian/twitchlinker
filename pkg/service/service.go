package service

import (
	"log"
	"time"

	"github.com/treybastian/twitchlinker/pkg/cloudflare"
	"github.com/treybastian/twitchlinker/pkg/twitch"
	"github.com/treybastian/twitchlinker/pkg/webhook"
)

type Service struct {
	twitchClient    *twitch.Client
	cloudflareClient *cloudflare.Client
	webhookServer   *webhook.WebhookServer
	config          *Config
}

type Config struct {
	TwitchClientID     string
	TwitchClientSecret string
	TwitchChannelName  string
	CloudflareAPIToken string
	CloudflareZoneID   string
	CloudflareDomain   string
	CloudflareRecord   string
	WebhookPort        string
	WebhookSecret      string
	WebhookURL         string
	PollInterval       time.Duration
}

func NewService(config *Config) (*Service, error) {
	// Initialize Twitch client
	twitchClient, err := twitch.NewClient(
		config.TwitchClientID,
		config.TwitchClientSecret,
		config.TwitchChannelName,
	)
	if err != nil {
		return nil, err
	}

	// Initialize Cloudflare client
	cloudflareClient, err := cloudflare.NewClient(
		config.CloudflareAPIToken,
		config.CloudflareZoneID,
		config.CloudflareDomain,
		config.CloudflareRecord,
	)
	if err != nil {
		return nil, err
	}

	service := &Service{
		twitchClient:     twitchClient,
		cloudflareClient: cloudflareClient,
		config:           config,
	}

	// Initialize webhook server
	webhookServer := webhook.NewWebhookServer(
		config.WebhookPort,
		config.WebhookSecret,
		service,
	)

	service.webhookServer = webhookServer

	return service, nil
}

func (s *Service) Start() error {
	// Initialize API clients
	log.Println("Initializing Twitch API client...")
	if err := s.twitchClient.Initialize(); err != nil {
		return err
	}

	log.Println("Initializing Cloudflare API client...")
	if err := s.cloudflareClient.Initialize(); err != nil {
		return err
	}

	// Subscribe to Twitch stream events
	log.Printf("Subscribing to stream events for channel: %s", s.config.TwitchChannelName)
	if err := s.twitchClient.SubscribeToStreamStatus(s.config.WebhookURL, s.config.WebhookSecret); err != nil {
		log.Printf("Warning: Failed to subscribe to stream events: %v", err)
		log.Println("Falling back to polling for stream status")
		go s.startPolling()
	}

	// Check current stream status
	s.checkStreamStatus()

	// Start webhook server
	return s.webhookServer.Start()
}

func (s *Service) startPolling() {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		s.checkStreamStatus()
	}
}

func (s *Service) checkStreamStatus() {
	isLive, streamURL, err := s.twitchClient.IsStreamLive()
	if err != nil {
		log.Printf("Error checking stream status: %v", err)
		return
	}

	if isLive {
		log.Printf("Channel %s is live", s.config.TwitchChannelName)
		if err := s.cloudflareClient.UpdateRedirect(streamURL); err != nil {
			log.Printf("Error updating redirect: %v", err)
		}
	} else {
		log.Printf("Channel %s is offline", s.config.TwitchChannelName)
		// You might want to handle the offline state differently
		// For example, redirect to a different URL or do nothing
	}
}

// HandleStreamOnline implements webhook.StreamStatusHandler
func (s *Service) HandleStreamOnline(channelName string) error {
	log.Printf("Stream went online for channel: %s", channelName)
	
	// Only process events for the configured channel
	if channelName != s.config.TwitchChannelName {
		log.Printf("Ignoring event for different channel: %s", channelName)
		return nil
	}
	
	streamURL := s.twitchClient.GetStreamURL()
	return s.cloudflareClient.UpdateRedirect(streamURL)
}

// HandleStreamOffline implements webhook.StreamStatusHandler
func (s *Service) HandleStreamOffline(channelName string) error {
	log.Printf("Stream went offline for channel: %s", channelName)
	
	// Only process events for the configured channel
	if channelName != s.config.TwitchChannelName {
		log.Printf("Ignoring event for different channel: %s", channelName)
		return nil
	}
	
	// You can customize this to redirect to a different URL when offline
	// For now, we'll just log it
	log.Printf("Channel %s is now offline", channelName)
	return nil
}