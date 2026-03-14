package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	statsHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747")).Bold(true)
	statsFieldStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	statsValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50"))
)

type StatsModel struct {
	Width        int
	Height       int
	Selected     int
	Running      bool
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
		Selected:     0,
		Running:      true,
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

func (m *StatsModel) MoveSelection(delta int) {
	if m.Running {
		return
	}
	m.Selected = (m.Selected + delta + 1) % 1
}

func (m *StatsModel) Activate() {
}

func (m *StatsModel) CancelEdit() {
}

func (m StatsModel) View() string {
	leftCol := borderStyle.Render("┌") + strings.Repeat("─", 24) + borderStyle.Render("┐") + "\n"
	leftCol += borderStyle.Render("│") + statsHeaderStyle.Render("  NETWORK STATS      ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("├") + strings.Repeat("─", 24) + borderStyle.Render("┤") + "\n"
	leftCol += borderStyle.Render("│") + statsFieldStyle.Render(" Local TUN IP:       ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + " " + m.LocalTunIP + strings.Repeat(" ", 19-len(m.LocalTunIP)) + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + statsFieldStyle.Render(" Remote Endpoint:    ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + " " + m.RemoteIP + strings.Repeat(" ", 19-len(m.RemoteIP)) + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + statsFieldStyle.Render(" Tunnel Status:     ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + " " + m.TunnelStatus + strings.Repeat(" ", 19-len(m.TunnelStatus)) + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + statsFieldStyle.Render(" Cert Valid Until:  ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + " " + m.CertValid + strings.Repeat(" ", 19-len(m.CertValid)) + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + statsFieldStyle.Render(" Connection Time:   ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + " " + m.ConnTime + strings.Repeat(" ", 19-len(m.ConnTime)) + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + statsFieldStyle.Render(" DNS Leak Test:     ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + " " + m.DNSLeak + strings.Repeat(" ", 19-len(m.DNSLeak)) + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + statsFieldStyle.Render(" Kill Switch:       ") + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("│") + " " + m.KillSwitch + strings.Repeat(" ", 19-len(m.KillSwitch)) + borderStyle.Render("│") + "\n"
	leftCol += borderStyle.Render("└") + strings.Repeat("─", 24) + borderStyle.Render("┘")

	rightCol := borderStyle.Render("┌") + strings.Repeat("─", 24) + borderStyle.Render("┐") + "\n"
	rightCol += borderStyle.Render("│") + statsHeaderStyle.Render(" REAL-TIME BANDWIDTH ") + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("├") + strings.Repeat("─", 24) + borderStyle.Render("┤") + "\n"
	rightCol += borderStyle.Render("│") + statsFieldStyle.Render(" Upload:  ") + m.UpSpeed + strings.Repeat(" ", 12-len(m.UpSpeed)) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + " ▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░" + strings.Repeat(" ", 5) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + statsFieldStyle.Render(" Download:") + m.DownSpeed + strings.Repeat(" ", 12-len(m.DownSpeed)) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + " ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓" + strings.Repeat(" ", 2) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("├") + strings.Repeat("─", 24) + borderStyle.Render("┤") + "\n"
	rightCol += borderStyle.Render("│") + statsHeaderStyle.Render(" SESSION PROGRESS    ") + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("├") + strings.Repeat("─", 24) + borderStyle.Render("┤") + "\n"
	rightCol += borderStyle.Render("│") + " ████████████████████░░░░░░░░░░░ │ 50%" + strings.Repeat(" ", 2) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + " Time: " + m.SessionTime + " | Data: " + m.SessionData + strings.Repeat(" ", 6) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + " Cost: " + m.SessionCost + strings.Repeat(" ", 14-len(m.SessionCost)) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("├") + strings.Repeat("─", 24) + borderStyle.Render("┤") + "\n"
	rightCol += borderStyle.Render("│") + statsHeaderStyle.Render(" COST HISTORY        ") + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("├") + strings.Repeat("─", 24) + borderStyle.Render("┤") + "\n"
	rightCol += borderStyle.Render("│") + "         25000 ┤              ╱" + strings.Repeat(" ", 6) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + "               │             ╱ " + strings.Repeat(" ", 7) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + "               │    ╱──────╱  " + strings.Repeat(" ", 7) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + "            5000┼───╱───────────" + strings.Repeat(" ", 5) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + "               └────┬────┬────┬" + strings.Repeat(" ", 7) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("│") + "                    10m  20m  30m  " + strings.Repeat(" ", 6) + borderStyle.Render("│") + "\n"
	rightCol += borderStyle.Render("└") + strings.Repeat("─", 24) + borderStyle.Render("┘")

	output := leftCol + strings.Repeat(" ", 4) + rightCol + "\n"
	output += " Session Total: " + m.DownTotal + " " + m.UpTotal + " | Est. Cost: " + m.SessionCost + "\n"

	return output
}
