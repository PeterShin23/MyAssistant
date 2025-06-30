# MyAssistant

A CLI tool that captures your **screen** and **microphone input**, then uses that data to predict what you need assistance with.


```bash
go run ./cmd/assistant listen
```

---

## Features

* Captures screenshot of the primary display
* Records mic input for a configurable duration
* Loads optional `rules.json` for contextual metadata
* Saves base64-encoded audio & screenshot in `.data/`

---

## Requirements
* `portaudio` and `ffmpeg` must be installed system-wide.
* For macOS ARM (M1/M2), you may need:

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
## Usage

### To Run
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