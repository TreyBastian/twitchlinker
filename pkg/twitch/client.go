package twitch

import (
	"errors"
	"log"

	"github.com/nicklaw5/helix/v2"
)

type Client struct {
	helixClient  *helix.Client
	channelNames []string
	channelIDs   map[string]string // Maps channel names to their IDs
	streamURLs   map[string]string // Maps channel names to their stream URLs
}

func NewClient(clientID, clientSecret string, channelNames []string) (*Client, error) {
	client, err := helix.NewClient(&helix.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})

	if err != nil {
		return nil, err
	}

	return &Client{
		helixClient:  client,
		channelNames: channelNames,
		channelIDs:   make(map[string]string),
		streamURLs:   make(map[string]string),
	}, nil
}

func (c *Client) Initialize() error {
	// Get app access token
	resp, err := c.helixClient.RequestAppAccessToken([]string{})
	if err != nil {
		return err
	}
	c.helixClient.SetAppAccessToken(resp.Data.AccessToken)
	
	// Get user IDs for all channels
	users, err := c.helixClient.GetUsers(&helix.UsersParams{
		Logins: c.channelNames,
	})
	if err != nil {
		return err
	}
	
	// Store user IDs and stream URLs
	for _, user := range users.Data.Users {
		c.channelIDs[user.Login] = user.ID
		c.streamURLs[user.Login] = "https://twitch.tv/" + user.Login
		log.Printf("Initialized channel %s with ID %s", user.Login, user.ID)
	}
	
	// Check if any channels weren't found
	if len(users.Data.Users) < len(c.channelNames) {
		// Log warning for channels not found
		foundChannels := make(map[string]bool)
		for _, user := range users.Data.Users {
			foundChannels[user.Login] = true
		}
		
		for _, channel := range c.channelNames {
			if !foundChannels[channel] {
				log.Printf("Warning: Channel not found: %s", channel)
			}
		}
	}
	
	return nil
}

func (c *Client) IsStreamLive() (bool, string, error) {
	if len(c.channelIDs) == 0 {
		return false, "", errors.New("no channels initialized")
	}
	
	// Get all user IDs
	var userIDs []string
	for _, id := range c.channelIDs {
		userIDs = append(userIDs, id)
	}
	
	// Check if any stream is live
	streams, err := c.helixClient.GetStreams(&helix.StreamsParams{
		UserIDs: userIDs,
		First:   100, // Maximum number of results
	})
	if err != nil {
		return false, "", err
	}
	
	// No streams are live
	if len(streams.Data.Streams) == 0 {
		return false, "", nil
	}
	
	// Get the first live stream (we can prioritize streams later if needed)
	liveStream := streams.Data.Streams[0]
	
	// Find the channel name from user ID
	var liveChannelName string
	for name, id := range c.channelIDs {
		if id == liveStream.UserID {
			liveChannelName = name
			break
		}
	}
	
	if liveChannelName == "" {
		return false, "", errors.New("couldn't map live stream to a channel name")
	}
	
	streamURL := c.streamURLs[liveChannelName]
	log.Printf("Channel %s is live", liveChannelName)
	
	return true, streamURL, nil
}

func (c *Client) SubscribeToStreamStatus(callbackURL, secret string) error {
	if len(c.channelIDs) == 0 {
		return errors.New("no channels initialized")
	}

	// Subscribe to all channels
	for channelName, userID := range c.channelIDs {
		// Create EventSub subscription for stream.online events
		onlineResp, err := c.helixClient.CreateEventSubSubscription(&helix.EventSubSubscription{
			Type:    "stream.online",
			Version: "1",
			Condition: helix.EventSubCondition{
				BroadcasterUserID: userID,
			},
			Transport: helix.EventSubTransport{
				Method:   "webhook",
				Callback: callbackURL,
				Secret:   secret,
			},
		})

		if err != nil {
			log.Printf("Error subscribing to stream.online events for channel %s: %v", channelName, err)
			continue
		}

		if onlineResp.StatusCode != 202 {
			log.Printf("EventSub subscription failed with status code: %d for channel %s", onlineResp.StatusCode, channelName)
			continue
		}

		log.Printf("Successfully subscribed to stream.online events for channel %s", channelName)
		
		// Create EventSub subscription for stream.offline events
		offlineResp, err := c.helixClient.CreateEventSubSubscription(&helix.EventSubSubscription{
			Type:    "stream.offline",
			Version: "1",
			Condition: helix.EventSubCondition{
				BroadcasterUserID: userID,
			},
			Transport: helix.EventSubTransport{
				Method:   "webhook",
				Callback: callbackURL,
				Secret:   secret,
			},
		})

		if err != nil {
			log.Printf("Error subscribing to stream.offline events for channel %s: %v", channelName, err)
			continue
		}

		if offlineResp.StatusCode != 202 {
			log.Printf("EventSub subscription failed with status code: %d for channel %s", offlineResp.StatusCode, channelName)
			continue
		}

		log.Printf("Successfully subscribed to stream.offline events for channel %s", channelName)
	}

	return nil
}

// GetStreamURL returns the URL for the first live channel or empty string if none are live
func (c *Client) GetStreamURL() string {
	isLive, url, err := c.IsStreamLive()
	if err != nil || !isLive {
		return ""
	}
	return url
}

// GetChannelNames returns all tracked channel names
func (c *Client) GetChannelNames() []string {
	return c.channelNames
}
