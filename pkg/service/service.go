package service

import (
	"log"
	"time"

	"github.com/treybastian/twitchlinker/pkg/cloudflare"
	"github.com/treybastian/twitchlinker/pkg/twitch"
	"github.com/treybastian/twitchlinker/pkg/webhook"
)

type Service struct {
	twitchClient     *twitch.Client
	cloudflareClient *cloudflare.Client
	webhookServer    *webhook.WebhookServer
	config           *Config
}

type Config struct {
	TwitchClientID     string
	TwitchClientSecret string
	TwitchChannelNames []string // Changed to a slice of channel names
	DefaultURL         string    // Added default URL fallback
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
		config.TwitchChannelNames,
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
	channels := s.twitchClient.GetChannelNames()
	channelList := "'"+channels[0]+"'"
	for i := 1; i < len(channels); i++ {
		channelList += ", '" + channels[i] + "'"
	}
	log.Printf("Subscribing to stream events for channels: %s", channelList)
	
	if err := s.twitchClient.SubscribeToStreamStatus(s.config.WebhookURL, s.config.WebhookSecret); err != nil {
		log.Printf("Warning: Failed to subscribe to stream events: %v", err)
		log.Println("Falling back to polling for stream status")
		go s.startPolling()
	}

	// Check current stream status
	if err := s.checkStreamStatus(); err != nil {
		log.Printf("Warning: Initial stream status check failed: %v", err)
	}

	// Start webhook server
	return s.webhookServer.Start()
}

func (s *Service) startPolling() {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		if err := s.checkStreamStatus(); err != nil {
			log.Printf("Error checking stream status during polling: %v", err)
		}
	}
}

func (s *Service) checkStreamStatus() error {
	isLive, streamURL, err := s.twitchClient.IsStreamLive()
	if err != nil {
		log.Printf("Error checking stream status: %v", err)
		return err
	}

	if isLive {
		log.Printf("Found a live channel, redirecting to: %s", streamURL)
		if err := s.cloudflareClient.UpdateRedirect(streamURL); err != nil {
			log.Printf("Error updating redirect: %v", err)
			return err
		}
	} else {
		log.Printf("No channels are currently live, redirecting to default URL: %s", s.config.DefaultURL)
		if s.config.DefaultURL != "" {
			if err := s.cloudflareClient.UpdateRedirect(s.config.DefaultURL); err != nil {
				log.Printf("Error updating redirect to default URL: %v", err)
				return err
			}
		} else {
			log.Printf("No default URL configured, keeping current redirect")
		}
	}
	
	return nil
}

// HandleStreamOnline implements webhook.StreamStatusHandler
func (s *Service) HandleStreamOnline(channelName string) error {
	log.Printf("Stream went online for channel: %s", channelName)

	// Verify this is one of our monitored channels
	channelNames := s.twitchClient.GetChannelNames()
	isMonitored := false
	for _, name := range channelNames {
		if channelName == name {
			isMonitored = true
			break
		}
	}

	if !isMonitored {
		log.Printf("Ignoring event for unmonitored channel: %s", channelName)
		return nil
	}

	// Recheck all streams to get the priority (in case multiple channels are live)
	return s.checkStreamStatus()
}

// HandleStreamOffline implements webhook.StreamStatusHandler
func (s *Service) HandleStreamOffline(channelName string) error {
	log.Printf("Stream went offline for channel: %s", channelName)

	// Verify this is one of our monitored channels
	channelNames := s.twitchClient.GetChannelNames()
	isMonitored := false
	for _, name := range channelNames {
		if channelName == name {
			isMonitored = true
			break
		}
	}

	if !isMonitored {
		log.Printf("Ignoring event for unmonitored channel: %s", channelName)
		return nil
	}

	// Recheck all streams to see if any other channel is live
	isLive, streamURL, err := s.twitchClient.IsStreamLive()
	if err != nil {
		log.Printf("Error checking stream status: %v", err)
		return err
	}

	if isLive {
		// Another channel is live, update to that one
		log.Printf("Another channel is live, updating redirect to: %s", streamURL)
		return s.cloudflareClient.UpdateRedirect(streamURL)
	} 
	
	// No channels are live, use default URL if configured
	if s.config.DefaultURL != "" {
		log.Printf("No channels are live, redirecting to default URL: %s", s.config.DefaultURL)
		return s.cloudflareClient.UpdateRedirect(s.config.DefaultURL)
	}
	
	log.Printf("No channels are live and no default URL configured, keeping current redirect")
	return nil
}
