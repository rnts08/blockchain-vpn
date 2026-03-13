package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type ConnectModel struct {
	Width         int
	Height        int
	SearchQuery   string
	CountryFilter string
	MinBandwidth  int
	MaxPrice      int
	PaymentFilter string
	MinSlots      int
	Providers     []ProviderEntry
	Selected      int
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
		Selected:    0,
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
	header := " Search: [________________________]  [Search]  [Rescan]\n\n"
	header += " Filters:  Country: [ALL    ]  Min BW: [0  ] Mbps  Max Price: [0   ] sats\n"
	header += "          Payment: [ALL    ]  Min Slots: [0  ]\n\n"
	header += " ========================================================================\n\n"
	header += " PROVIDERS FOUND: 47                                    Sort: [PRICE]\n"
	header += " ========================================================================\n"
	header += " #   Country  Host         | Latency  Slots  Price   Payment  Actions\n"
	header += " -------------------------+---------- ------ ------- -------- --------\n"

	result := header
	for i, p := range m.Providers {
		marker := " "
		if i == m.Selected {
			marker = ">"
		}
		result += fmt.Sprintf(" %s%d   %s  %s | %s     %d/%d   %s   %s   [CONN]\n",
			marker, p.ID, p.Country, p.Host, p.Latency, p.Slots, p.MaxSlots, p.Price, p.PaymentModel)
	}
	result += " ========================================================================\n"

	return result
}

func (m ConnectModel) renderDetails() string {
	if m.Selected >= len(m.Providers) {
		return "No provider selected"
	}
	p := m.Providers[m.Selected]

	output := fmt.Sprintf(" Selected: #%d - %s (%s)\n\n", p.ID, p.Host, p.Country)
	output += " ========================================================================\n"
	output += " DETAILS                                         [ Speed Test ]\n"
	output += " ========================================================================\n"
	output += " Country:           Germany (DE)\n"
	output += " Bandwidth:         100 Mbps (Advertised: 10 Mbps)\n"
	output += " Max Consumers:     10\n"
	output += fmt.Sprintf(" Available Slots:    %d\n", p.Slots)
	output += " Payment Model:     Per Session\n"
	output += fmt.Sprintf(" Price:            %s\n", p.Price)
	output += " Reputation:       ***** (42 reviews)\n"
	output += " Protocol:        TLS 1.3 + WireGuard\n"
	output += " NAT Traversal:    UPnP + NAT-PMP\n"
	output += " ========================================================================\n"
	output += " PRE-FLIGHT CHECKS\n"
	output += " [X] Payment Possible      [X] Balance Sufficient (50,000 sats)\n"
	output += " [X] RPC Connected        [X] Tunnel Available\n"
	output += " ========================================================================\n"
	output += "\n"
	output += "        [ <- Back ]        [ CONNECT ]        [ Test Speed -> ]\n"

	return output
}

func (m ConnectModel) renderConnecting() string {
	bar := ""
	for i := 0; i < 20; i++ {
		if i < m.Progress*20/100 {
			bar += "="
		} else {
			bar += "-"
		}
	}

	output := "                    CONNECTING TO PROVIDER...\n\n"
	output += fmt.Sprintf(" [%s] %d%%\n\n", bar, m.Progress)

	output += " Step 1: Creating Payment Transaction..."
	if m.Progress >= 20 {
		output += " OK\n"
	} else {
		output += "\n"
	}

	output += " Step 2: Waiting for Confirmation......"
	if m.Progress >= 40 {
		output += " OK\n"
	} else {
		output += "\n"
	}

	output += " Step 3: Establishing TLS Tunnel........"
	if m.Progress >= 60 {
		output += " OK\n"
	} else {
		output += "\n"
	}

	output += " Step 4: Setting up TUN Interface......"
	if m.Progress >= 80 {
		output += " OK\n"
	} else {
		output += "\n"
	}

	output += " Step 5: Running Security Tests........"
	if m.Progress >= 100 {
		output += " OK\n"
	} else {
		output += "\n"
	}

	output += "\n Payment: 100 sats to bc1q...xyz (tx: 842a1...b2c3)\n"

	return output
}
