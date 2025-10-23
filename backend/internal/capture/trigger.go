package capture

import (
	"fmt"
	"sync"

	"github.com/PeterShin23/MyAssistant/backend/internal/openai"
	"github.com/PeterShin23/MyAssistant/backend/internal/screen"
)

// Manager handles capture session triggers from multiple sources
type Manager struct {
	session *openai.Session
	mu      sync.Mutex
	running bool
}

// NewManager creates a new capture manager
func NewManager(session *openai.Session) *Manager {
	return &Manager{
		session: session,
	}
}

// TriggerScreenshot triggers a screenshot-only capture session
// Returns an error if a session is already running or if capture fails
func (m *Manager) TriggerScreenshot() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("capture session already in progress")
	}
	m.running = true
	m.mu.Unlock()

	// Ensure we mark as not running when done
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	fmt.Println("📸 Remote screenshot triggered...")

	// Capture screenshot
	screenshotPath, err := screen.CaptureScreenshot()
	if err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}

	fmt.Println("✅ Screenshot captured, processing...")

	// Process with OpenAI (no audio)
	if err := m.session.Process(screenshotPath, "", false); err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	fmt.Println("✅ Processing complete")
	return nil
}

// IsRunning returns true if a capture session is currently running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}
