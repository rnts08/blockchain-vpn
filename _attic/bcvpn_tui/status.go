package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type StatusModel struct {
	Width        int
	Height       int
	Mode         string
	Status       string
	ExternalIP   string
	Country      string
	PublicKey    string
	ActiveConn   int
	MaxConn      int
	Uptime       string
	DownTotal    string
	UpTotal      string
	DownSpeed    string
	UpSpeed      string
	PaymentModel string
	Price        string
	Balance      string
	Earned       string
}

func NewStatusModel() StatusModel {
	return StatusModel{
		Mode:         "PROVIDER",
		Status:       "RUNNING",
		ExternalIP:   "45.33.22.11",
		Country:      "US",
		PublicKey:    "02a1...b4c3",
		ActiveConn:   2,
		MaxConn:      10,
		Uptime:       "3d 14h 22m",
		DownTotal:    "1.24 GB",
		UpTotal:      "456.78 MB",
		DownSpeed:    "2.4 MB/s",
		UpSpeed:      "512 KB/s",
		PaymentModel: "TIME (per minute)",
		Price:        "10 sats/minute",
		Balance:      "50,000 sats",
		Earned:       "12,450 sats",
	}
}

func (m StatusModel) Init() tea.Cmd {
	return nil
}

func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
	return m, nil
}

func (m StatusModel) View() string {
	return ` ╔═══════════════════════════════════════════════════════════════════════╗ 
 ║                        SYSTEM STATUS                                  ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║  Mode:              PROVIDER                                          ║ 
 ║  Status:            RUNNING                                           ║ 
 ║  External IP:       45.33.22.11 (Rotated)                            ║ 
 ║  Country:           US (United States)                               ║ 
 ║  Public Key:        02a1...b4c3                                       ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║                        CONNECTIONS                                     ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║  Active:             2 / 10                                           ║ 
 ║  Total Sessions:    147                                                ║ 
 ║  Uptime:            3d 14h 22m                                        ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║                        BANDWIDTH (Session)                           ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║  Downloaded:       1.24 GB    │  Uploaded:      456.78 MB           ║ 
 ║  Current:           ↓ 2.4 MB/s │  ↑ 512 KB/s                         ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║                        PAYMENT                                         ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║  Model:             TIME (per minute)                                 ║ 
 ║  Price:             10 sats/minute                                    ║ 
 ║  Earned:            12,450 sats                                        ║ 
 ║  Last Payment:      2 minutes ago                                      ║ 
 ╚═══════════════════════════════════════════════════════════════════════╝`
}
