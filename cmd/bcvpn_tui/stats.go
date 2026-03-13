package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type StatsModel struct {
	Width        int
	Height       int
	LocalTunIP   string
	RemoteIP     string
	TunnelStatus string
	CertValid    string
	ConnTime     string
	DNSLeak      string
	KillSwitch   string
	SessionTime  string
	SessionData  string
	SessionCost  string
	DownTotal    string
	UpTotal      string
	DownSpeed    string
	UpSpeed      string
}

func NewStatsModel() StatsModel {
	return StatsModel{
		Width:        80,
		Height:       24,
		LocalTunIP:   "10.0.0.2/24",
		RemoteIP:     "1.2.3.4:51820",
		TunnelStatus: "ENCRYPTED",
		CertValid:    "2026-03-15 14:22",
		ConnTime:     "00:45:23",
		DNSLeak:      "PROTECTED",
		KillSwitch:   "ENABLED",
		SessionTime:  "00:45:23",
		SessionData:  "1.24 GB",
		SessionCost:  "234 sats",
		DownTotal:    "1.24 GB",
		UpTotal:      "456.78 MB",
		DownSpeed:    "12.4 MB/s",
		UpSpeed:      "5.2 MB/s",
	}
}

func (m StatsModel) Init() tea.Cmd {
	return nil
}

func (m StatsModel) Update(msg tea.Msg) (StatsModel, tea.Cmd) {
	return m, nil
}

func (m StatsModel) View() string {
	leftCol := " NETWORK STATS\n"
	leftCol += " ==================\n\n"
	leftCol += " Local TUN IP:\n"
	leftCol += " " + m.LocalTunIP + "\n\n"
	leftCol += " Remote Endpoint:\n"
	leftCol += " " + m.RemoteIP + "\n\n"
	leftCol += " Tunnel Status:\n"
	leftCol += " " + m.TunnelStatus + "\n\n"
	leftCol += " Cert Valid Until:\n"
	leftCol += " " + m.CertValid + "\n\n"
	leftCol += " Connection Time:\n"
	leftCol += " " + m.ConnTime + "\n\n"
	leftCol += " DNS Leak Test:\n"
	leftCol += " " + m.DNSLeak + "\n\n"
	leftCol += " Kill Switch:\n"
	leftCol += " " + m.KillSwitch

	rightCol := " REAL-TIME BANDWIDTH\n"
	rightCol += " ======================\n\n"
	rightCol += " Upload: " + m.UpSpeed + "\n"
	rightCol += " ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░\n\n"
	rightCol += " Download: " + m.DownSpeed + "\n"
	rightCol += " ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓\n\n"
	rightCol += " ======================\n"
	rightCol += " SESSION PROGRESS\n"
	rightCol += " ======================\n"
	rightCol += " +-------------------------------+\n"
	rightCol += " | ████████████████████░░░░░░░░░░ | 50%\n"
	rightCol += " | Time: " + m.SessionTime + " | Data: " + m.SessionData + " |\n"
	rightCol += " | Cost: " + m.SessionCost + " |\n"
	rightCol += " +-------------------------------+\n\n"
	rightCol += " ======================\n"
	rightCol += " COST HISTORY (Provider)\n"
	rightCol += " ======================\n"
	rightCol += " sats\n"
	rightCol += " 25000 |                  /\n"
	rightCol += "       |                 /\n"
	rightCol += "       |                /\n"
	rightCol += "       |       /-------/\n"
	rightCol += "       |      /\n"
	rightCol += "       |     /\n"
	rightCol += "   5000 +----/--------------------\n"
	rightCol += "         10m  20m  30m  40m  50m  60m\n\n"

	output := " ========================================================================\n"
	output += leftCol + spaces(40) + rightCol
	output += " ========================================================================\n\n"
	output += " Session Total: " + m.DownTotal + " " + m.UpTotal + " | Est. Cost: " + m.SessionCost + "\n"

	return output
}
