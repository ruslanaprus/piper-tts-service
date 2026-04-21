package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type VoiceConfig struct {
	Name   string `json:"name"`
	Model  string `json:"model"`
	Config string `json:"config"`
}

type Voices map[string]VoiceConfig

var voices Voices

func loadVoices() error {
	data, err := os.ReadFile("/app/voices/voices.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &voices)
}

func ttsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
		Lang string `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text field is required", http.StatusBadRequest)
		return
	}

	if req.Lang == "" {
		req.Lang = "en"
	}

	voice, ok := voices[req.Lang]
	if !ok {
		voice = voices["en"]
	}

	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("piper-%d.wav", time.Now().UnixNano()))

	cmd := exec.Command("piper", "--model", voice.Model, "--config", voice.Config, "--output_file", outputFile)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		http.Error(w, "Failed to start Piper", http.StatusInternalServerError)
		return
	}

	_, err = io.WriteString(stdin, req.Text)
	if err != nil {
		http.Error(w, "Failed to write to Piper", http.StatusInternalServerError)
		return
	}
	stdin.Close()

	if err := cmd.Run(); err != nil {
		log.Printf("Piper execution failed: %v", err)
		http.Error(w, "Piper execution failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/wav")
	http.ServeFile(w, r, outputFile)

	go func() {
		time.Sleep(2 * time.Second)
		os.Remove(outputFile)
	}()
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	if err := loadVoices(); err != nil {
		log.Fatalf("Failed to load voices: %v", err)
	}

	http.HandleFunc("/tts", ttsHandler)
	http.HandleFunc("/health", healthHandler)

	log.Println("Piper TTS service running on :5000")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
