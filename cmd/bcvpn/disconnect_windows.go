//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"blockchain-vpn/internal/config"
)

func handleDisconnect(cfg *config.Config) {
	dir, err := config.AppConfigDir()
	if err != nil {
		log.Fatalf("Failed to get app config dir: %v", err)
	}
	pidPath := filepath.Join(dir, "client.pid")
	pid, err := readPIDFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No active VPN connection found.")
			fmt.Println()
			fmt.Println("To connect to a VPN provider, run: bcvpn scan")
			os.Exit(1)
		}
		log.Fatalf("Failed to read client PID file: %v", err)
	}
	if !isProcessRunning(pid) {
		os.Remove(pidPath)
		clearSessionInfo()
		fmt.Printf("Previous VPN connection (PID %d) is no longer running (stale PID file).\n", pid)
		fmt.Println("To connect to a VPN provider, run: bcvpn scan")
		os.Exit(1)
	}
	log.Printf("Disconnecting client (PID %d)...", pid)
	if err := stopProviderProcess(pid, true); err != nil {
		log.Fatalf("Failed to send disconnect signal: %v", err)
	}
	log.Println("Disconnect signal sent.")
	timeout := 5 * time.Second
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isProcessRunning(pid) {
			break
		}
		<-ticker.C
	}
	os.Remove(pidPath)
	if isProcessRunning(pid) {
		log.Println("Warning: client process did not exit within timeout; PID file cleared.")
	} else {
		log.Println("Client disconnected successfully.")
	}
	promptForRating(cfg)
}

func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Kill()
	return err == nil
}

func stopProviderProcess(pid int, graceful bool) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process PID %d: %w", pid, err)
	}
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("failed to kill process PID %d: %w", pid, err)
	}
	return nil
}
