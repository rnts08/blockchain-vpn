package main

import (
	"fmt"
	"strings"

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
	primaryStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747"))
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#f44336"))
	warningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9800"))
	fieldTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	boxStyle       = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#dca747")).
			Foreground(lipgloss.Color("#e0e0e0"))
	focusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#212121")).
			Background(lipgloss.Color("#dca747")).
			Bold(true)
	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#212121")).
			Padding(0, 1)
	selStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#dca747")).
			Background(lipgloss.Color("#333333"))
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#e0e0e0"))
	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#dca747"))
)

type MainModel struct {
	Width          int
	Height         int
	CurrentTab     Tab
	StatusModel    StatusModel
	ProviderModel  ProviderModel
	ConnectModel   ConnectModel
	StatsModel     StatsModel
	QuitConfirm    bool
	UnsavedChanges bool
	Loading        bool
	LoadingMessage string
	HelpOpen       bool
	ScrollOffset   int
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
		if m.QuitConfirm {
			switch msg.String() {
			case "y", "Y":
				return m, tea.Quit
			case "n", "N", "esc":
				m.QuitConfirm = false
			}
			return m, nil
		}

		if m.HelpOpen {
			if msg.String() == "esc" || msg.String() == "?" {
				m.HelpOpen = false
			}
			return m, nil
		}

		switch msg.String() {
		case "tab":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("enter")
			}
			m.CurrentTab = (m.CurrentTab + 1) % 4
			m.resetSelection()
		case "shift+tab":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("enter")
			}
			m.CurrentTab = (m.CurrentTab + 3) % 4
			m.resetSelection()
		case "right":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("enter")
			}
		case "left":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("enter")
			}
		case "s", "S":
			m.CurrentTab = TabStatus
			m.resetSelection()
		case "p", "P":
			m.CurrentTab = TabProvider
			m.resetSelection()
		case "c", "C":
			if m.CurrentTab != TabConnect {
				m.CurrentTab = TabConnect
				m.resetSelection()
			}
		case "o", "O":
			m.CurrentTab = TabStats
			m.resetSelection()
		case "q", "Q":
			m.QuitConfirm = true
			return m, nil
		case "?":
			m.HelpOpen = true
			return m, nil
		case "v", "V":
			m.activateSelection()
		case "up", "k":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("up")
			} else {
				m.moveSelection(-1)
			}
		case "down", "j":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("down")
			} else {
				m.moveSelection(1)
			}
		case "enter", " ":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("enter")
			} else {
				m.activateSelection()
			}
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey(msg.String())
			}
		case "esc":
			if m.CurrentTab == TabProvider && m.ProviderModel.Editing >= 0 {
				m.ProviderModel.HandleEditKey("esc")
			} else {
				m.cancelEdit()
			}
		case "pgup", "PgUp":
			m.ScrollOffset -= 5
			if m.ScrollOffset < 0 {
				m.ScrollOffset = 0
			}
		case "pgdown", "PgDown":
			m.ScrollOffset += 5
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

func (m *MainModel) resetSelection() {
	switch m.CurrentTab {
	case TabProvider:
		m.ProviderModel.Selected = 0
	case TabConnect:
		m.ConnectModel.Selected = 0
	case TabStats:
		m.StatsModel.Selected = 0
	}
}

func (m *MainModel) moveSelection(delta int) {
	switch m.CurrentTab {
	case TabProvider:
		m.ProviderModel.MoveSelection(delta)
		m.updateScrollForSelection(m.ProviderModel.Selected, len(m.ProviderModel.Fields)+4)
	case TabConnect:
		m.ConnectModel.MoveSelection(delta)
		m.updateScrollForSelection(m.ConnectModel.Selected, len(m.ConnectModel.Providers))
	case TabStats:
		m.StatsModel.MoveSelection(delta)
	}
}

func (m *MainModel) updateScrollForSelection(selected, total int) {
	usableHeight := m.Height - 4
	if selected < m.ScrollOffset {
		m.ScrollOffset = selected
	} else if selected >= m.ScrollOffset+usableHeight {
		m.ScrollOffset = selected - usableHeight + 1
	}
}

func (m *MainModel) activateSelection() {
	switch m.CurrentTab {
	case TabProvider:
		m.ProviderModel.Activate()
	case TabConnect:
		m.ConnectModel.Activate()
	case TabStats:
		m.StatsModel.Activate()
	}
}

func (m *MainModel) cancelEdit() {
	switch m.CurrentTab {
	case TabProvider:
		m.ProviderModel.CancelEdit()
	case TabConnect:
		m.ConnectModel.CancelEdit()
	}
}

func (m MainModel) View() string {
	if m.QuitConfirm {
		return m.renderQuitConfirm()
	}

	if m.HelpOpen {
		return m.renderHelpView()
	}

	header := m.renderTabs()
	content := m.renderContent()
	footer := m.renderStatusBar()

	return header + "\n" + content + "\n" + footer
}

func (m MainModel) renderQuitConfirm() string {
	msg := "Are you sure you want to quit?"
	if m.UnsavedChanges {
		msg = "WARNING: You have unsaved changes! Are you sure you want to quit?"
	}

	lines := []string{
		"",
		" ┌─────────────────────────────────────┐",
		" │         CONFIRM QUIT               │",
		" ├─────────────────────────────────────┤",
		" │ " + msg + string(spaces(intMax(0, 35-len(msg)))) + " │",
		" │                                     │",
		" │     [Y] Yes  /  [N] No              │",
		" └─────────────────────────────────────┘",
	}

	usableHeight := m.Height
	centerY := (usableHeight - len(lines)) / 2

	var result string
	for i := 0; i < centerY; i++ {
		result += "\n" + primaryStyle.Render("│") + spaces(m.Width-2) + primaryStyle.Render("│")
	}

	for i, line := range lines {
		padding := (m.Width - len(line)) / 2
		if padding < 0 {
			padding = 0
		}
		if i == 0 {
			result += "\n" + spaces(padding) + line
		} else {
			result += "\n" + spaces(padding) + line
		}
	}

	for i := centerY + len(lines); i < usableHeight-1; i++ {
		result += "\n" + primaryStyle.Render("│") + spaces(m.Width-2) + primaryStyle.Render("│")
	}

	result += "\n" + primaryStyle.Render("└") + strings.Repeat("─", m.Width-2) + primaryStyle.Render("┘")

	return result
}

func (m MainModel) renderHelpView() string {
	lines := []string{
		" ┌─────────────────────────────────────┐",
		" │           KEYBOARD SHORTCUTS         │",
		" ├─────────────────────────────────────┤",
		" │                                     │",
		" │ Navigation                          │",
		" │   [Tab]         Next tab            │",
		" │   [Shift+Tab]  Previous tab         │",
		" │   [↑/↓] or j/k Select fields       │",
		" │   [Enter]      Activate/Edit        │",
		" │   [Esc]        Cancel/Back          │",
		" │                                     │",
		" │ Tabs                                │",
		" │   [S]tatus     Status view          │",
		" │   [P]rovider   Provider config      │",
		" │   [C]onnect    Connect to provider  │",
		" │   [O]verview   Stats overview       │",
		" │                                     │",
		" │ Actions                             │",
		" │   [Q]uit       Quit (confirms)     │",
		" │   [?]          This help           │",
		" │   [V]          Validate config      │",
		" └─────────────────────────────────────┘",
	}

	usableHeight := m.Height
	centerY := (usableHeight - len(lines)) / 2

	var result string
	for i := 0; i < centerY; i++ {
		result += "\n" + primaryStyle.Render("│") + spaces(m.Width-2) + primaryStyle.Render("│")
	}

	for _, line := range lines {
		padding := (m.Width - len(line)) / 2
		if padding < 0 {
			padding = 0
		}
		result += "\n" + spaces(padding) + line
	}

	for i := centerY + len(lines); i < usableHeight-1; i++ {
		result += "\n" + primaryStyle.Render("│") + spaces(m.Width-2) + primaryStyle.Render("│")
	}

	result += "\n" + primaryStyle.Render("└") + strings.Repeat("─", m.Width-2) + primaryStyle.Render("┘")

	return result
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m MainModel) renderTabs() string {
	tabs := []struct {
		name     string
		shortcut string
	}{
		{"[S]tatus", "S"},
		{"[P]rovider", "P"},
		{"[C]onnect", "C"},
		{"[O]verview", "O"},
	}

	var tabLine string
	for i, tab := range tabs {
		if Tab(i) == m.CurrentTab {
			tabLine += focusedStyle.Render(" "+tab.name+" ") + " "
		} else {
			tabLine += tabStyle.Render(" "+tab.name+" ") + " "
		}
	}

	helpText := tabStyle.Render(" [?]")
	headerLine := primaryStyle.Render("┌") + strings.Repeat("─", m.Width-2) + primaryStyle.Render("┐")
	tabBar := primaryStyle.Render("│") + tabLine + helpText + spaces(m.Width-2-stringWidth(tabLine+helpText)) + primaryStyle.Render("│")
	separator := primaryStyle.Render("├") + strings.Repeat("─", m.Width-2) + primaryStyle.Render("┤")

	return headerLine + "\n" + tabBar + "\n" + separator
}

func (m MainModel) renderContent() string {
	var content string
	switch m.CurrentTab {
	case TabStatus:
		content = m.StatusModel.View()
	case TabProvider:
		m.ProviderModel.Width = m.Width
		content = m.ProviderModel.View()
	case TabConnect:
		m.ConnectModel.Width = m.Width
		content = m.ConnectModel.View()
	case TabStats:
		m.StatsModel.Width = m.Width
		content = m.StatsModel.View()
	}

	lines := splitLines(content)
	usableHeight := m.Height - 6

	if len(lines) > usableHeight {
		if m.ScrollOffset > len(lines)-usableHeight {
			m.ScrollOffset = len(lines) - usableHeight
		}
		if m.ScrollOffset < 0 {
			m.ScrollOffset = 0
		}
		lines = lines[m.ScrollOffset : m.ScrollOffset+usableHeight]
	} else {
		m.ScrollOffset = 0
	}

	var result string
	for _, line := range lines {
		contentWidth := stringWidth(line)
		rightPadding := m.Width - 2 - contentWidth - 1
		if rightPadding < 0 {
			rightPadding = 0
		}
		result += primaryStyle.Render("│") + line + spaces(rightPadding) + primaryStyle.Render("│") + "\n"
	}

	emptyLines := usableHeight - len(lines)
	for i := 0; i < emptyLines; i++ {
		result += primaryStyle.Render("│") + spaces(m.Width-2) + primaryStyle.Render("│") + "\n"
	}

	return result
}

func (m MainModel) renderStatusBar() string {
	status := successStyle.Render("CONNECTED")
	download := " ↓ 5.2 MB/s"
	upload := " ↑ 1.1 MB/s"
	rpc := " RPC: " + successStyle.Render("✓")
	balance := " BAL: 50K sats"
	tun := " TUN: " + successStyle.Render("✓")

	var loading string
	if m.Loading {
		loading = " " + warningStyle.Render("⚙ "+m.LoadingMessage)
	}

	content := status + download + upload + rpc + balance + tun + loading
	padding := m.Width - 2 - stringWidth(content) - 1
	if padding < 0 {
		padding = 0
	}

	return primaryStyle.Render("│") + successStyle.Render(status) + fieldTextStyle.Render(download) +
		fieldTextStyle.Render(upload) + fieldTextStyle.Render(rpc) + fieldTextStyle.Render(balance) +
		fieldTextStyle.Render(tun) + loading + spaces(padding) + primaryStyle.Render("│")
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
		fmt.Println("Error:", err)
	}
}
