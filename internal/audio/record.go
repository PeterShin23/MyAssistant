package audio

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

const FOLDER_PERMS_READALL_WRITEOWN = 0755

var (
	recording bool
	mu        sync.Mutex
	stream    *portaudio.Stream
	buffer    []int16
)

func StartRecording() error {
	// Prevents changes by simulatneous processes
	mu.Lock()
	defer mu.Unlock()

	// If already recording, don't do anything
	if recording {
		return nil
	}

	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("Failed to initialize portaudio: %w", err)
	}

	// Creating a slice with length 0 and capacity 44100*25 -> 44100Hz for a maximum of 25 sec
	buffer = make([]int16, 0, 44100*25)
	// Creating a slice with length 512 and capacity 512 to act as buffer per loop
	tmpBuf := make([]int16, 512)

	var err error
	// Open the stream
	stream, err = portaudio.OpenDefaultStream(1, 0, 44100, len(tmpBuf), &tmpBuf)
	if err != nil {
		return fmt.Errorf("Failed to open stream: %w", err)
	}

	recording = true

	// This is an anonymous function, kind of like immediately invoked function
	// We invoke this in a goroutine which executes concurrently with the current call
	go func() {
		stream.Start()

		for recording {
			if err := stream.Read(); err == nil {
				buffer = append(buffer, tmpBuf...)
			}
		}

		stream.Stop()
		portaudio.Terminate()
	}()

	fmt.Println("Recording...")

	return nil
}


func StopRecording() error {
	mu.Lock()
	defer mu.Unlock()

	// Do nothing, if it's already stopped
	if !recording {
		return nil
	}

	recording = false

	now := time.Now().Format("2006-01-02T15:04:05.000")
	outputPath := fmt.Sprintf(".data/%s.wav", now)

	// Ensure .data directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), FOLDER_PERMS_READALL_WRITEOWN); err != nil {
		return fmt.Errorf("failed to create .data directory: %w", err)
	}
	
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Failed to create wav file: %w", err)
	}
	defer file.Close()

	writeWavHeader(file, len(buffer))

	if err := binary.Write(file, binary.LittleEndian, buffer); err != nil {
		return fmt.Errorf("Failed to write wav data: %w", err)
	}

	if err := convertWavToMp3(outputPath); err != nil {
		return fmt.Errorf("mp3 conversion failed: %w", err)
	}

	return nil
}

func writeWavHeader(f *os.File, sampleCount int) {
	dataSize := sampleCount * 2
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte("WAVEfmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(f, binary.LittleEndian, uint16(1)) // Mono
	binary.Write(f, binary.LittleEndian, uint32(44100))
	binary.Write(f, binary.LittleEndian, uint32(44100*2))
	binary.Write(f, binary.LittleEndian, uint16(2))  // BlockAlign
	binary.Write(f, binary.LittleEndian, uint16(16)) // Bits
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))
}

func convertWavToMp3(wavPath string) error {
	mp3Path := strings.Replace(wavPath, ".wav", ".mp3", 1)

	cmd := exec.Command("ffmpeg", "-y", "-i", wavPath, "-codec:a", "libmp3lame", "-qscale:a", "2", mp3Path)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}