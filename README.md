# MyAssistant

An LLM-Powered CLI tool written in Go that takes a screenshot of your computer display and optionally records your mic. Your personal assistant without having to do any more than what you already do.

---
## Usage

### To Run
Hold down backtick (`) to record. Keep held down if you're also recording audio.

```bash
go run ./cmd/assistant listen
```

### To Run without MIC (Only screen capture)
```bash
go run ./cmd/assistant listen --no-audio
```

### To clear data saved in .data
```bash
go run ./cmd/assistant clear
```


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