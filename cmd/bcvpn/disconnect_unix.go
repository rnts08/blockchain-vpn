//go:build linux || darwin

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
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
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
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
	err := syscall.Kill(pid, 0)
	return err == nil
}

func stopProviderProcess(pid int, graceful bool) error {
	if graceful {
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM to PID %d: %w", pid, err)
		}
		timeout := 10 * time.Second
		for i := 0; i < int(timeout/time.Second); i++ {
			if !isProcessRunning(pid) {
				return nil
			}
			time.Sleep(1 * time.Second)
		}
		if isProcessRunning(pid) {
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				return fmt.Errorf("failed to send SIGKILL to PID %d: %w", pid, err)
			}
		}
		return nil
	}
	return syscall.Kill(pid, syscall.SIGKILL)
}
