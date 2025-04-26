package twitch

import (
	"errors"
	"log"

	"github.com/nicklaw5/helix/v2"
)

type Client struct {
	helixClient *helix.Client
	channelName string
}

func NewClient(clientID, clientSecret, channelName string) (*Client, error) {
	client, err := helix.NewClient(&helix.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})

	if err != nil {
		return nil, err
	}

	return &Client{
		helixClient: client,
		channelName: channelName,
	}, nil
}

func (c *Client) Initialize() error {
	resp, err := c.helixClient.RequestAppAccessToken([]string{})
	if err != nil {
		return err
	}

	c.helixClient.SetAppAccessToken(resp.Data.AccessToken)
	return nil
}

func (c *Client) IsStreamLive() (bool, string, error) {
	// Get user ID from username first
	users, err := c.helixClient.GetUsers(&helix.UsersParams{
		Logins: []string{c.channelName},
	})
	if err != nil {
		return false, "", err
	}

	if len(users.Data.Users) == 0 {
		return false, "", errors.New("channel not found")
	}

	userID := users.Data.Users[0].ID

	// Check if stream is live
	streams, err := c.helixClient.GetStreams(&helix.StreamsParams{
		UserIDs: []string{userID},
	})
	if err != nil {
		return false, "", err
	}

	if len(streams.Data.Streams) == 0 {
		return false, "", nil
	}

	// Return true and the stream URL
	return true, "https://twitch.tv/" + c.channelName, nil
}

func (c *Client) SubscribeToStreamStatus(callbackURL, secret string) error {
	// Get user ID from username first
	users, err := c.helixClient.GetUsers(&helix.UsersParams{
		Logins: []string{c.channelName},
	})
	if err != nil {
		return err
	}

	if len(users.Data.Users) == 0 {
		return errors.New("channel not found")
	}

	userID := users.Data.Users[0].ID

	// Create EventSub subscription for stream.online events
	resp, err := c.helixClient.CreateEventSubSubscription(&helix.EventSubSubscription{
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
		return err
	}

	if resp.StatusCode != 202 {
		log.Printf("EventSub subscription failed with status code: %d", resp.StatusCode)
		return errors.New("failed to create EventSub subscription")
	}

	log.Printf("Successfully subscribed to stream.online events for channel %s", c.channelName)
	return nil
}

func (c *Client) GetStreamURL() string {
	return "https://twitch.tv/" + c.channelName
}

