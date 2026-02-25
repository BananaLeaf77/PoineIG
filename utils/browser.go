package utils

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func LaunchBrave() error {
	// Kill any existing Brave instances first
	exec.Command("pkill", "-f", "brave").Run()
	time.Sleep(2 * time.Second) // wait for it to fully die

	userDataDir := os.ExpandEnv("$HOME/.config/BraveSoftware/Brave-Browser")
	fmt.Println("🚀 Launching Brave...")

	brave := exec.Command("brave",
		"--remote-debugging-port=9222",
		"--user-data-dir="+userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
	)
	brave.Stdout = os.Stdout
	brave.Stderr = os.Stderr

	if err := brave.Start(); err != nil {
		return fmt.Errorf("failed to launch Brave: %v", err)
	}

	return WaitForBrave()
}

func WaitForBrave() error {
	fmt.Println("⏳ Waiting for Brave to be ready...")
	client := &http.Client{Timeout: time.Second}

	for i := 0; i < 30; i++ { // try for 30 seconds
		resp, err := client.Get("http://127.0.0.1:9222/json/version")
		if err == nil {
			resp.Body.Close()
			fmt.Println("✓ Brave is ready!")
			return nil
		}
		time.Sleep(time.Second)
		fmt.Printf("\r⏳ Waiting... %ds", i+1)
	}
	return fmt.Errorf("brave didn't start within 30 seconds")
}
