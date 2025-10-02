# MyAssistant

An LLM-Powered CLI tool written in Go that takes a screenshot of your computer display and optionally records your mic. Your personal assistant without having to do any more than what you already do.

---
## Usage

### To Run
Hold down backtick (`) to record. Keep held down if you're also recording audio.

```bash
go run ./backend/cmd/assistant listen
```

### To Run without MIC (Only screen capture)
```bash
go run ./backend/cmd/assistant listen --no-audio
```

### To clear data saved in .data
```bash
go run ./backend/cmd/assistant clear
```

### WebSocket Streaming

The assistant can optionally stream output over WebSocket to a phone app in addition to printing to stdout.

```bash
go run ./backend/cmd/assistant listen --ws-url="ws://localhost:4000/stream?role=producer" --ws-token=secret
```

You can also set environment variables as fallbacks:
- `MYASSISTANT_WS_URL`
- `MYASSISTANT_WS_TOKEN`

### WebSocket Relay Server

A minimal WebSocket relay server is provided for local testing. It accepts producer connections with `?role=producer` and viewer connections with `?role=viewer`, broadcasting messages from producers to all viewers.

To run the relay server:

```bash
cd tools/ws-relay
go run main.go
```

To run with authentication:

```bash
cd tools/ws-relay
go run main.go --ws-token=secret
```

### React Native Client

A minimal React Native (Expo) client is provided in `mobile/stream-viewer/` that connects to the WebSocket relay server and renders streamed content.

To run the Expo app:

```bash
cd mobile/stream-viewer
npm install --legacy-peer-deps
npx expo start
```

If you encounter dependency conflicts, you can also use yarn instead of npm:

```bash
cd mobile/stream-viewer
yarn install
npx expo start
```

Scan the QR code with your phone to run the app.

Before connecting, make sure to update the URL in the app to match your computer's IP address. The default URL in the app (`http://localhost:4000`) needs to be changed to your Mac's actual IP address.

For remote device testing, you can use Cloudflare Tunnel or ngrok:

1. Start the relay server locally
2. Expose it with ngrok: `ngrok http 4000`
3. Update the URL in the Expo app to use the ngrok URL
4. Run the CLI with the ngrok URL: `go run ./backend/cmd/assistant listen --ws-url="<ngrok-url>/stream?role=producer"`

---

## Requirements
Sorry, but this is MacOS only :(

* `portaudio` and `ffmpeg` must be installed system-wide.

```bash
brew install portaudio
brew install ffmpeg

export PKG_CONFIG_PATH=/opt/homebrew/lib/pkgconfig
```

### .env

* Must have OPENAI_API_KEY - Follow `.env.template`

### Golang

* Go 1.20 or later

---

## Configuration

Create a `rules.json` file at the project root:

```json
{
  "whatDoYouNeedHelpWith": "I'm a software engineer, preparing for interviews.",
  "display": 1
}
```

---

## Troubleshoot

### Accessibility API disabled

```
hook_run [1284]: Accessibility API is disabled!
Failed to enable access for assistive devices. (0X40)
```

Open System Settings → Privacy & Security → Accessibility

Click the + button and add your terminal (e.g., iTerm or Terminal)

Ensure the checkbox is checked

Restart the terminal after doing this
