# MyAssistant

A CLI tool that captures your **screen** and **microphone input**, then saves the data locally in `.data/` for later use (such as LLM processing).

---

## ✅ Features

* 📸 Captures screenshot of the primary display
* 🎤 Records mic input for a configurable duration
* 🧠 Loads optional `rules.json` for contextual metadata
* 💾 Saves base64-encoded audio & screenshot in `.data/`

---

## 📦 Requirements
* `portaudio` and `ffmpeg` must be installed system-wide.
* For macOS ARM (M1/M2), you may need:

```bash
brew install portaudio
brew install ffmpeg

export PKG_CONFIG_PATH=/opt/homebrew/lib/pkgconfig
```

### Golang

* Go 1.20 or later

---

## 🚀 Usage

### Run without building:

```bash
go run ./cmd/assistant
```

### Build a binary:

```bash
go build -o ./bin/myassistant ./cmd/assistant
./bin/myassistant
```

---

## ⚙️ Configuration (Optional)

Create a `rules.json` file at the project root:

```json
{
  "context": "You're helping me troubleshoot CLI issues.",
  "max_tokens": 300
}
```

---

## Troubleshoot

1. Accessibility API disabled

```
hook_run [1284]: Accessibility API is disabled!
Failed to enable access for assistive devices. (0X40)
```

Open System Settings → Privacy & Security → Accessibility

Click the + button and add your terminal (e.g., iTerm or Terminal)

Ensure the checkbox is checked

Restart the terminal after doing this