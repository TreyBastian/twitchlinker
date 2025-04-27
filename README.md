# TwitchLinker

TwitchLinker is a Go application that automatically updates a Cloudflare DNS record to point to a Twitch channel when it goes live. This allows you to have a custom domain (like `stream.yourdomain.com`) that always redirects to your live Twitch stream.

## Features

- Monitors multiple Twitch channels and redirects to the first one that's live
- Falls back to a default URL when no channels are live
- Listens for Twitch EventSub notifications when channels go live or offline
- Automatically updates a Cloudflare DNS record with the appropriate redirect
- Falls back to polling the Twitch API if webhook setup fails
- Configurable via environment variables

## Requirements

- Go 1.18 or later
- A Twitch Developer account with API credentials
- A Cloudflare account with API token and a domain
- Public internet access for webhook notifications (or use a service like ngrok for development)

## Setup

1. Clone this repository:
   ```
   git clone https://github.com/treybastian/twitchlinker.git
   cd twitchlinker
   ```

2. Copy the example environment file:
   ```
   cp .env.example .env
   ```

3. Edit the `.env` file with your credentials:
   - **Twitch API Credentials**: Create an application at [Twitch Developer Console](https://dev.twitch.tv/console/apps)
   - **Cloudflare Credentials**: Get these from your Cloudflare dashboard
   - **Webhook Configuration**: Set up a publicly accessible URL for Twitch to send notifications to

4. Build the application:
   ```
   go build -o twitchlinker
   ```

5. Run the application:
   ```
   ./twitchlinker
   ```

## Webhook Testing

For local development, you can use [ngrok](https://ngrok.com/) to expose your local webhook server to the internet:

```
ngrok http 8080
```

Use the provided ngrok URL as your `WEBHOOK_URL` in the .env file.

## Twitch EventSub

This application uses Twitch's EventSub API to receive notifications when streams go live or offline. It subscribes to both the `stream.online` and `stream.offline` event types for all configured channels.

For more information, see the [Twitch EventSub documentation](https://dev.twitch.tv/docs/eventsub).

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| TWITCH_CLIENT_ID | Your Twitch application client ID | Yes |
| TWITCH_CLIENT_SECRET | Your Twitch application client secret | Yes |
| TWITCH_CHANNEL_NAMES | Comma-separated list of Twitch channels to monitor | Yes* |
| TWITCH_CHANNEL_NAME | Single Twitch channel to monitor (legacy, use TWITCH_CHANNEL_NAMES instead) | Yes* |
| DEFAULT_URL | URL to redirect to when no channels are live | No |
| CLOUDFLARE_API_TOKEN | Your Cloudflare API token | Yes |
| CLOUDFLARE_ZONE_ID | The Zone ID for your domain | Yes |
| CLOUDFLARE_DOMAIN | Your domain name (e.g., example.com) | Yes |
| CLOUDFLARE_RECORD | The subdomain to update (e.g., "stream" for stream.example.com) | Yes |
| WEBHOOK_PORT | The port for the webhook server | No (default: 8080) |
| WEBHOOK_SECRET | A secret for validating Twitch notifications | Yes |
| WEBHOOK_URL | The public URL for the webhook endpoint | Yes |
| POLL_INTERVAL_SECONDS | How often to poll Twitch if webhooks fail | No (default: 60) |

\* Either TWITCH_CHANNEL_NAMES or TWITCH_CHANNEL_NAME must be provided.

## License

MIT