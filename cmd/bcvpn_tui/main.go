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
			Foreground(lipgloss.Color("#212121")).
			Background(lipgloss.Color("#dca747")).
			Padding(0, 1).
			Bold(true)
	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#212121")).
			Padding(0, 1)
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#e0e0e0"))
	fieldStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0e0e0")).
			Background(lipgloss.Color("#333333"))
	activeFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#dca747")).
				Background(lipgloss.Color("#444444")).
				Bold(true)
	boldStyle = lipgloss.NewStyle().Bold(true)
)

type MainModel struct {
	Width         int
	Height        int
	CurrentTab    Tab
	StatusModel   StatusModel
	ProviderModel ProviderModel
	ConnectModel  ConnectModel
	StatsModel    StatsModel
	ConfigOpen    bool
	HelpOpen      bool
	Editing       bool
}

func NewMainModel() MainModel {
	return MainModel{
		Width:         80,
		Height:        24,
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
		case "tab", "right":
			m.CurrentTab = (m.CurrentTab + 1) % 4
			m.Editing = false
		case "shift+tab", "left":
			m.CurrentTab = (m.CurrentTab + 3) % 4
			m.Editing = false
		case "q", "Q":
			return m, tea.Quit
		case "c", "C":
			m.ConfigOpen = !m.ConfigOpen
			m.Editing = false
		case "?":
			m.HelpOpen = !m.HelpOpen
		case "1":
			m.CurrentTab = TabStatus
			m.Editing = false
		case "2":
			m.CurrentTab = TabProvider
			m.Editing = false
		case "3":
			m.CurrentTab = TabConnect
			m.Editing = false
		case "4":
			m.CurrentTab = TabStats
			m.Editing = false
		case "enter":
			if !m.Editing {
				m.Editing = true
				m.ProviderModel.ActiveField = 0
			}
		case "up", "k":
			if m.Editing && m.CurrentTab == TabProvider {
				m.ProviderModel.ActiveField = (m.ProviderModel.ActiveField - 1 + len(providerFields)) % len(providerFields)
			}
		case "down", "j":
			if m.Editing && m.CurrentTab == TabProvider {
				m.ProviderModel.ActiveField = (m.ProviderModel.ActiveField + 1) % len(providerFields)
			}
		case "esc":
			if m.Editing {
				m.Editing = false
			} else if m.ConfigOpen {
				m.ConfigOpen = false
			} else if m.HelpOpen {
				m.HelpOpen = false
			}
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
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

	lines := header + "\n" + content + "\n" + footer
	return lines
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
			if m.Editing && Tab(i) == TabProvider {
				result += focusedTabStyle.Render(" "+tab+"*") + " "
			} else {
				result += focusedTabStyle.Render(" "+tab+" ") + " "
			}
		} else {
			result += tabStyle.Render(" "+tab+" ") + " "
		}
	}

	helpText := tabStyle.Render(" [?]Help")
	modeText := tabStyle.Render(" MODE: ")
	if m.Editing {
		modeText += warningStyle.Render("EDITING")
	} else {
		modeText += successStyle.Render("VIEW")
	}

	return primaryStyle.Render("┌─") + result + primaryStyle.Render("─┬") + helpText + primaryStyle.Render("─┤") + modeText + primaryStyle.Render("─┐")
}

func (m MainModel) renderContent() string {
	var content string
	switch m.CurrentTab {
	case TabStatus:
		content = m.StatusModel.View()
	case TabProvider:
		m.ProviderModel.Editing = m.Editing
		content = m.ProviderModel.View()
	case TabConnect:
		content = m.ConnectModel.View()
	case TabStats:
		content = m.StatsModel.View()
	}

	lines := splitLines(content)
	usableHeight := m.Height - 4
	if len(lines) > usableHeight {
		lines = lines[:usableHeight]
	}

	var result string
	for i, line := range lines {
		padding := m.Width - stringWidth(line)
		if padding < 0 {
			padding = 0
		}
		if i == 0 {
			result += primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		} else {
			result += "\n" + primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		}
	}

	emptyLines := usableHeight - len(lines)
	for i := 0; i < emptyLines; i++ {
		result += "\n" + primaryStyle.Render("│ ") + spaces(m.Width-2) + primaryStyle.Render(" │")
	}

	return result
}

func (m MainModel) renderStatusBar() string {
	status := successStyle.Render("●") + " CONNECTED"
	download := " ↓ 5.2 MB/s"
	upload := " ↑ 1.1 MB/s"
	rpc := " RPC: " + successStyle.Render("✓")
	balance := " BAL: 50K sats"
	tun := " TUN: " + successStyle.Render("✓")

	if m.Editing {
		status = warningStyle.Render("◐") + " EDITING"
	}

	bar := statusBarStyle.Render(status + download + upload + rpc + balance + tun)
	return primaryStyle.Render("└") + bar + primaryStyle.Render("─┘")
}

func (m MainModel) renderConfigView() string {
	m.ProviderModel.Editing = true
	m.ProviderModel.ActiveField = 0
	content := m.ProviderModel.renderConfig()

	lines := splitLines(content)
	usableHeight := m.Height - 2
	if len(lines) > usableHeight {
		lines = lines[:usableHeight]
	}

	var result string
	for i, line := range lines {
		padding := m.Width - stringWidth(line)
		if padding < 0 {
			padding = 0
		}
		if i == 0 {
			result += primaryStyle.Render("┌─") + line + spaces(padding) + primaryStyle.Render("─┐")
		} else if i == len(lines)-1 {
			result += "\n" + primaryStyle.Render("└─") + line + spaces(padding) + primaryStyle.Render("─┘")
		} else {
			result += "\n" + primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		}
	}
	return result
}

func (m MainModel) renderHelpView() string {
	content := renderHelp()

	lines := splitLines(content)
	usableHeight := m.Height - 2
	if len(lines) > usableHeight {
		lines = lines[:usableHeight]
	}

	var result string
	for i, line := range lines {
		padding := m.Width - stringWidth(line)
		if padding < 0 {
			padding = 0
		}
		if i == 0 {
			result += primaryStyle.Render("┌─") + line + spaces(padding) + primaryStyle.Render("─┐")
		} else if i == len(lines)-1 {
			result += "\n" + primaryStyle.Render("└─") + line + spaces(padding) + primaryStyle.Render("─┘")
		} else {
			result += "\n" + primaryStyle.Render("│ ") + line + spaces(padding) + primaryStyle.Render(" │")
		}
	}
	return result
}

func renderHelp() string {
	return `KEYBOARD SHORTCUTS

Navigation
--------------------------------
Tab/Shift+Tab    Next / Previous tab
<-/->            Navigate tabs
j/k              Navigate fields (in edit mode)
Enter            Start editing / Select
Esc               Stop editing / Back

Actions
--------------------------------
C                   Open configuration editor
R                   Refresh / Rescan
S                   Save / Start
Q                   Quit
?                   Show this help`
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
