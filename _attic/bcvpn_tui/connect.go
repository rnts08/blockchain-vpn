package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	connectFieldStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	connectSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747")).Bold(true)
	connectHeaderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747")).Bold(true)
)

type ConnectModel struct {
	Width         int
	Height        int
	Selected      int
	SearchQuery   string
	CountryFilter string
	MinBandwidth  int
	MaxPrice      int
	PaymentFilter string
	MinSlots      int
	Providers     []ProviderEntry
	DetailsOpen   bool
	Connecting    bool
	Progress      int
}

type ProviderEntry struct {
	ID           int
	Country      string
	Host         string
	Latency      string
	Slots        int
	MaxSlots     int
	Price        string
	PaymentModel string
}

func NewConnectModel() ConnectModel {
	return ConnectModel{
		Selected:      0,
		SearchQuery:   "",
		CountryFilter: "ALL",
		MinBandwidth:  0,
		MaxPrice:      0,
		PaymentFilter: "ALL",
		MinSlots:      0,
		Providers: []ProviderEntry{
			{1, "DE", "1.2.3.4:51820", "45ms", 8, 10, "100/s", "session"},
			{2, "FR", "2.3.4.5:51820", "62ms", 5, 10, "150/s", "session"},
			{3, "US", "3.4.5.6:51820", "89ms", 0, 5, "200/s", "session"},
			{4, "GB", "4.5.6.7:51820", "78ms", 10, 10, "50/s", "time"},
			{5, "JP", "5.6.7.8:51820", "156ms", 3, 5, "100/GB", "data"},
			{6, "CA", "6.7.8.9:51820", "95ms", 2, 10, "75/s", "session"},
			{7, "AU", "7.8.9.0:51820", "245ms", 10, 10, "200/s", "session"},
			{8, "DE", "8.9.0.1:51820", "48ms", 6, 10, "80/s", "time"},
		},
		DetailsOpen: false,
		Connecting:  false,
		Progress:    0,
	}
}

func (m ConnectModel) Init() tea.Cmd {
	return nil
}

func (m ConnectModel) Update(msg tea.Msg) (ConnectModel, tea.Cmd) {
	return m, nil
}

func (m *ConnectModel) MoveSelection(delta int) {
	if m.DetailsOpen || m.Connecting {
		return
	}
	m.Selected = (m.Selected + delta + len(m.Providers)) % len(m.Providers)
}

func (m *ConnectModel) Activate() {
	if m.DetailsOpen {
		m.DetailsOpen = false
	} else if m.Connecting {
		// Cancel connection
		m.Connecting = false
	} else if m.Selected < len(m.Providers) {
		// Show details and allow connect
		m.DetailsOpen = true
	}
}

func (m *ConnectModel) Connect() {
	if m.Selected < len(m.Providers) {
		m.Connecting = true
		m.Progress = 0
	}
}

func (m *ConnectModel) CancelEdit() {
	m.DetailsOpen = false
}

func (m ConnectModel) View() string {
	if m.DetailsOpen {
		return m.renderDetails()
	}
	if m.Connecting {
		return m.renderConnecting()
	}
	return m.renderProviderList()
}

func (m ConnectModel) renderProviderList() string {
	sel := m.Selected

	header := connectHeaderStyle.Render(" SEARCH & FILTERS") + "\n"
	header += connectHeaderStyle.Render(" ===============") + "\n"
	header += " Search: [" + m.SearchQuery + strings.Repeat("_", intMax(0, 24-len(m.SearchQuery))) + "]  [Search]  [Rescan]\n\n"
	header += fmt.Sprintf(" Filters:  Country: [%s]  Min BW: [%d]  Max Price: [%d]\n", m.CountryFilter, m.MinBandwidth, m.MaxPrice)
	header += fmt.Sprintf("          Payment: [%s]  Min Slots: [%d]\n\n", m.PaymentFilter, m.MinSlots)
	header += connectHeaderStyle.Render(fmt.Sprintf(" PROVIDERS (%d found)", len(m.Providers))) + "\n"
	header += connectHeaderStyle.Render(" =======================") + "\n"
	header += " #   Country  Host               | Latency  Slots  Price   Payment\n"
	header += " ----+----------+------------------+--------+-------+--------\n"

	result := header
	for i, p := range m.Providers {
		marker := "  "
		rowStyle := connectFieldStyle
		if i == sel {
			marker = "▶"
			rowStyle = connectSelectedStyle
		}
		result += rowStyle.Render(fmt.Sprintf("%s %-2d   %-2s   %-18s | %6s  %d/%d    %-6s %s",
			marker, p.ID, p.Country, p.Host, p.Latency, p.Slots, p.MaxSlots, p.Price, p.PaymentModel)) + "\n"
	}
	result += "\n [↑/↓] Navigate  [Enter] Details  [T]est Speed  [C]onnect"

	return result
}

func (m ConnectModel) renderDetails() string {
	if m.Selected >= len(m.Providers) {
		return "No provider selected"
	}
	p := m.Providers[m.Selected]

	output := connectHeaderStyle.Render(" PROVIDER DETAILS") + "\n"
	output += connectHeaderStyle.Render(" =================") + "\n\n"
	output += connectFieldStyle.Render(fmt.Sprintf(" Host:           %s", p.Host)) + "\n"
	output += connectFieldStyle.Render(fmt.Sprintf(" Country:       %s", p.Country)) + "\n"
	output += connectFieldStyle.Render(" Bandwidth:      100 Mbps") + "\n"
	output += connectFieldStyle.Render(fmt.Sprintf(" Max Consumers:  %d", p.MaxSlots)) + "\n"
	output += connectFieldStyle.Render(fmt.Sprintf(" Available:      %d slots", p.Slots)) + "\n"
	output += connectFieldStyle.Render(fmt.Sprintf(" Price:         %s", p.Price)) + "\n"
	output += connectFieldStyle.Render(fmt.Sprintf(" Payment:       %s", p.PaymentModel)) + "\n"
	output += connectFieldStyle.Render(" Reputation:     ★★★★☆ (42 reviews)") + "\n"
	output += connectFieldStyle.Render(" Protocol:       TLS 1.3 + WireGuard") + "\n\n"
	output += connectHeaderStyle.Render(" PRE-FLIGHT CHECKS") + "\n"
	output += connectHeaderStyle.Render(" ----------------") + "\n"
	output += successStyle.Render(" [✓] Payment possible") + "\n"
	output += successStyle.Render(" [✓] Balance sufficient") + "\n"
	output += successStyle.Render(" [✓] RPC connected") + "\n\n"
	output += " [<- Back]  [CONNECT]  [Test Speed]"

	return output
}

func (m ConnectModel) renderConnecting() string {
	output := " CONNECTING...\n"
	output += " =============\n\n"

	bar := ""
	for i := 0; i < 20; i++ {
		if i < m.Progress*20/100 {
			bar += "="
		} else {
			bar += "-"
		}
	}

	output += fmt.Sprintf(" [%s] %d%%\n\n", bar, m.Progress)

	output += " Step 1: Payment..."
	if m.Progress >= 25 {
		output += " OK\n"
	}
	output += " Step 2: Confirmation..."
	if m.Progress >= 50 {
		output += " OK\n"
	}
	output += " Step 3: TLS Tunnel..."
	if m.Progress >= 75 {
		output += " OK\n"
	}
	output += " Step 4: TUN Setup..."
	if m.Progress >= 100 {
		output += " OK\n"
	}

	return output
}
