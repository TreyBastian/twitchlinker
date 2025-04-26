package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type StreamStatusHandler interface {
	HandleStreamOnline(channelName string) error
	HandleStreamOffline(channelName string) error
}

type WebhookServer struct {
	port      string
	secretKey string
	handler   StreamStatusHandler
}

type EventSubNotification struct {
	Subscription struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"subscription"`
	Event map[string]interface{} `json:"event"`
}

func NewWebhookServer(port, secretKey string, handler StreamStatusHandler) *WebhookServer {
	return &WebhookServer{
		port:      port,
		secretKey: secretKey,
		handler:   handler,
	}
}

func (s *WebhookServer) Start() error {
	http.HandleFunc("/webhook", s.handleWebhook)

	log.Printf("Starting webhook server on port %s", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}

func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify the webhook is from Twitch
	if !s.verifyTwitchSignature(r) {
		log.Println("Invalid webhook signature")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Handle verification challenge
	messageType := r.Header.Get("Twitch-Eventsub-Message-Type")
	if messageType == "webhook_callback_verification" {
		var challenge struct {
			Challenge string `json:"challenge"`
		}

		if err := json.Unmarshal(body, &challenge); err != nil {
			log.Printf("Error unmarshaling verification challenge: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(challenge.Challenge))
		log.Println("Successfully responded to webhook verification challenge")
		return
	}

	// Parse the notification
	var notification EventSubNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		log.Printf("Error unmarshaling notification: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Process the event
	switch notification.Subscription.Type {
	case "stream.online":
		broadcasterUserLogin, ok := notification.Event["broadcaster_user_login"].(string)
		if !ok {
			log.Println("Error: couldn't get broadcaster_user_login from event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("Stream online event received for channel: %s", broadcasterUserLogin)
		if err := s.handler.HandleStreamOnline(broadcasterUserLogin); err != nil {
			log.Printf("Error handling stream online event: %v", err)
		}

	case "stream.offline":
		broadcasterUserLogin, ok := notification.Event["broadcaster_user_login"].(string)
		if !ok {
			log.Println("Error: couldn't get broadcaster_user_login from event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("Stream offline event received for channel: %s", broadcasterUserLogin)
		if err := s.handler.HandleStreamOffline(broadcasterUserLogin); err != nil {
			log.Printf("Error handling stream offline event: %v", err)
		}

	default:
		log.Printf("Received unhandled event type: %s", notification.Subscription.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *WebhookServer) verifyTwitchSignature(r *http.Request) bool {
	messageID := r.Header.Get("Twitch-Eventsub-Message-Id")
	timestamp := r.Header.Get("Twitch-Eventsub-Message-Timestamp")
	signature := r.Header.Get("Twitch-Eventsub-Message-Signature")

	if messageID == "" || timestamp == "" || signature == "" {
		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	// Replace the body so it can be read again
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Create the message used to compute the signature
	message := messageID + timestamp + string(body)

	// Compute the HMAC
	h := hmac.New(sha256.New, []byte(s.secretKey))
	h.Write([]byte(message))
	expectedSignature := "sha256=" + hex.EncodeToString(h.Sum(nil))

	return signature == expectedSignature
}
