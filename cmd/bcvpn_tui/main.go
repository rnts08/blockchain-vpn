package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Tab int

const (
	TabStatus Tab = iota
	TabProvider
	TabConnect
	TabStats
)

var (
	primaryStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747"))
	backgroundStyle = lipgloss.NewStyle().Background(lipgloss.Color("#212121"))
	titleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747")).Bold(true)
	successStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#f44336"))
	warningStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9800"))
	boxStyle        = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#dca747")).
			Foreground(lipgloss.Color("#e0e0e0"))
	focusedTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#dca747")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)
	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#212121")).
			Padding(0, 1)
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#e0e0e0"))
	boldStyle = lipgloss.NewStyle().Bold(true)
)

type MainModel struct {
	CurrentTab    Tab
	StatusModel   StatusModel
	ProviderModel ProviderModel
	ConnectModel  ConnectModel
	StatsModel    StatsModel
	ConfigOpen    bool
	HelpOpen      bool
}

func NewMainModel() MainModel {
	return MainModel{
		CurrentTab:    TabStatus,
		StatusModel:   NewStatusModel(),
		ProviderModel: NewProviderModel(),
		ConnectModel:  NewConnectModel(),
		StatsModel:    NewStatsModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "right", "left":
			m.CurrentTab = m.switchTab(msg.String())
		case "q", "Q":
			return m, tea.Quit
		case "c", "C":
			m.ConfigOpen = !m.ConfigOpen
		case "?":
			m.HelpOpen = !m.HelpOpen
		case "1":
			m.CurrentTab = TabStatus
		case "2":
			m.CurrentTab = TabProvider
		case "3":
			m.CurrentTab = TabConnect
		case "4":
			m.CurrentTab = TabStats
		}
	case tea.WindowSizeMsg:
		m.StatusModel.Width = msg.Width
		m.StatusModel.Height = msg.Height
		m.ProviderModel.Width = msg.Width
		m.ProviderModel.Height = msg.Height
		m.ConnectModel.Width = msg.Width
		m.ConnectModel.Height = msg.Height
		m.StatsModel.Width = msg.Width
		m.StatsModel.Height = msg.Height
	}

	var cmd tea.Cmd
	switch m.CurrentTab {
	case TabStatus:
		m.StatusModel, cmd = m.StatusModel.Update(msg)
	case TabProvider:
		m.ProviderModel, cmd = m.ProviderModel.Update(msg)
	case TabConnect:
		m.ConnectModel, cmd = m.ConnectModel.Update(msg)
	case TabStats:
		m.StatsModel, cmd = m.StatsModel.Update(msg)
	}

	return m, cmd
}

func (m MainModel) switchTab(key string) Tab {
	switch key {
	case "tab", "right":
		return (m.CurrentTab + 1) % 4
	case "shift+tab", "left":
		return (m.CurrentTab + 3) % 4
	}
	return m.CurrentTab
}

func (m MainModel) View() string {
	if m.ConfigOpen {
		return m.renderConfigView()
	}
	if m.HelpOpen {
		return m.renderHelpView()
	}

	header := m.renderTabs()
	content := m.renderContent()
	footer := m.renderStatusBar()

	return header + "\n" + content + "\n" + footer
}

func (m MainModel) renderTabs() string {
	tabs := []string{
		"STATUS",
		"PROVIDER",
		"CONNECT",
		"STATS",
	}

	var result string
	for i, tab := range tabs {
		if Tab(i) == m.CurrentTab {
			result += focusedTabStyle.Render(" "+tab+" ") + " "
		} else {
			result += tabStyle.Render(" "+tab+" ") + " "
		}
	}

	helpText := tabStyle.Render(" [?]Help")
	return primaryStyle.Render("┌─") + result + primaryStyle.Render("─┬") + helpText + primaryStyle.Render("─┐")
}

func (m MainModel) renderContent() string {
	var content string
	switch m.CurrentTab {
	case TabStatus:
		content = m.StatusModel.View()
	case TabProvider:
		content = m.ProviderModel.View()
	case TabConnect:
		content = m.ConnectModel.View()
	case TabStats:
		content = m.StatsModel.View()
	}

	width := 80
	if m.StatusModel.Width > 0 {
		width = m.StatusModel.Width
	}

	lines := ""
	for i, line := range splitLines(content) {
		padding := width - stringWidth(line)
		if padding < 0 {
			padding = 0
		}
		if i == 0 {
			lines += primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		} else {
			lines += "\n" + primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		}
	}

	emptyLines := 20 - len(splitLines(content))
	for i := 0; i < emptyLines; i++ {
		lines += "\n" + primaryStyle.Render("│ ") + spaces(width-2) + primaryStyle.Render(" │")
	}

	return lines
}

func (m MainModel) renderStatusBar() string {
	status := successStyle.Render("●") + " CONNECTED"
	download := " ↓ 5.2 MB/s"
	upload := " ↑ 1.1 MB/s"
	rpc := " RPC: " + successStyle.Render("✓")
	balance := " BAL: 50K sats"
	tun := " TUN: " + successStyle.Render("✓")

	bar := statusBarStyle.Render(status + download + upload + rpc + balance + tun)
	return primaryStyle.Render("└") + bar + primaryStyle.Render("─┘")
}

func (m MainModel) renderConfigView() string {
	content := boxStyle.Render(`CONFIGURATION EDITOR

[RPC SETTINGS]
  Host:           [localhost:25173                                  ]
  User:           [rpcuser                                         ]
  Password:       [••••••••••••                                    ]
  Network:       [MAINNET ▼]  Token Symbol: [ORDEX]

[SECURITY]
  Key Storage:   [FILE    ▼]
  TLS Min Ver:   [1.3    ▼]  Profile: [MODERN ▼]
  Metrics Token: [••••••••••••]

[PROVIDER]
  Interface:    [bcvpn0  ]  Listen Port: [51820]
  TUN IP:       [10.0.0.1]  Subnet: [/24 ▼]

[CLIENT]
  Interface:    [bcvpn1  ]
  Max Tunnels:  [1   ]  Kill Switch: [●]

                        [ Cancel ]    [ Save & Apply ]`)

	width := 80
	lines := ""
	for i, line := range splitLines(content) {
		padding := width - stringWidth(line)
		if padding < 0 {
			padding = 0
		}
		if i == 0 {
			lines += primaryStyle.Render("┌─") + line + spaces(padding) + primaryStyle.Render("─┐")
		} else if i == len(splitLines(content))-1 {
			lines += "\n" + primaryStyle.Render("└─") + line + spaces(padding) + primaryStyle.Render("─┘")
		} else {
			lines += "\n" + primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		}
	}
	return lines
}

func (m MainModel) renderHelpView() string {
	content := boxStyle.Render(`KEYBOARD SHORTCUTS

Navigation
─────────────────────────────────────────────────────────────────────
Tab/Shift+Tab    Next / Previous tab
←/→              Navigate tabs
j/k              Down / Up (in lists)
Enter            Select / Confirm
Esc              Back / Cancel

Actions
─────────────────────────────────────────────────────────────────────
C                   Open configuration editor
R                   Refresh / Rescan
S                   Save / Start
T                   Test speed
Q                   Quit
?                   Show this help

Provider Mode
─────────────────────────────────────────────────────────────────────
A                   Announce to blockchain
U                   Update price

Consumer Mode
─────────────────────────────────────────────────────────────────────
F                   Filter providers
/                   Search
C                   Connect to selected
D                   Disconnect`)

	width := 80
	lines := ""
	for i, line := range splitLines(content) {
		padding := width - stringWidth(line)
		if padding < 0 {
			padding = 0
		}
		if i == 0 {
			lines += primaryStyle.Render("┌─") + line + spaces(padding) + primaryStyle.Render("─┐")
		} else if i == len(splitLines(content))-1 {
			lines += "\n" + primaryStyle.Render("└─") + line + spaces(padding) + primaryStyle.Render("─┘")
		} else {
			lines += "\n" + primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		}
	}
	return lines
}

func splitLines(s string) []string {
	var lines []string
	line := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func spaces(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += " "
	}
	return result
}

func stringWidth(s string) int {
	width := 0
	for _, r := range s {
		if r == '\t' {
			width += 8
		} else {
			width++
		}
	}
	return width
}

func main() {
	p := tea.NewProgram(
		NewMainModel(),
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running TUI:", err)
	}
}
