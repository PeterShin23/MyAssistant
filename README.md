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

### Golang

* Go 1.20 or later

### Dependencies

#### macOS:

```bash
brew install portaudio ffmpeg
```

#### Ubuntu/Debian:

```bash
sudo apt update
sudo apt install portaudio19-dev ffmpeg
```

---

## 🚀 Usage

### Run without building:

```bash
go run ./cmd/desktopassistant --duration 5
```

### Build a binary:

```bash
go build -o ./bin/desktopassistant ./cmd/desktopassistant
./bin/desktopassistant --duration 5
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

## 📁 Output

* `.data/payload_<timestamp>.json` – JSON with base64-encoded screenshot & audio
* `capture.png` – PNG screenshot
* `audio.wav` – Raw microphone input

---

## 🧱 Project Structure

```
DesktopAssistant/
├── cmd/desktopassistant/main.go        # CLI entrypoint
├── internal/audio/mic.go               # Mic recording logic
├── internal/screen/capture.go          # Screenshot logic
├── internal/config/rules.go            # Config loader
├── internal/storage/store.go           # Payload builder/saver
├── rules.json                          # Optional config
├── .data/                              # Captured sessions
```

---

## 🛠 Dev Commands

```bash
# Format code
go fmt ./...

# Clean artifacts
rm -rf bin/ audio.wav capture.png .data/
```

---

## 📌 Notes

* `portaudio` and `ffmpeg` must be installed system-wide.
* For macOS ARM (M1/M2), you may need:

```bash
export PKG_CONFIG_PATH=/opt/homebrew/lib/pkgconfig
```

* Compatible with macOS and Linux (x86/ARM).
