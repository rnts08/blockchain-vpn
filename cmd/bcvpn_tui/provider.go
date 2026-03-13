package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type Field struct {
	Name     string
	Value    string
	Editable bool
}

var providerFields = []string{
	"country",
	"autoDetect",
	"maxConsumers",
	"maxDuration",
	"bandwidthLimit",
	"advertisedMbps",
	"paymentModel",
	"price",
	"timeUnit",
	"dataUnit",
	"paymentAddr",
	"enableNAT",
	"enableUPnP",
	"natPMP",
	"isolation",
	"healthCheck",
	"healthInterval",
}

type ProviderModel struct {
	Width          int
	Height         int
	Editing        bool
	ActiveField    int
	Running        bool
	Country        string
	AutoDetect     bool
	MaxConsumers   int
	MaxDuration    int
	BandwidthLimit string
	AdvertisedMbps int
	PaymentModel   string
	Price          int
	TimeUnit       string
	DataUnit       string
	PaymentAddr    string
	EnableNAT      bool
	EnableUPnP     bool
	NATPMP         bool
	Isolation      string
	HealthCheck    bool
	HealthInterval string
	Announced      bool
	LastUpdate     string
}

func NewProviderModel() ProviderModel {
	return ProviderModel{
		Editing:        false,
		ActiveField:    0,
		Running:        false,
		Country:        "US",
		AutoDetect:     true,
		MaxConsumers:   10,
		MaxDuration:    0,
		BandwidthLimit: "10mbit",
		AdvertisedMbps: 10,
		PaymentModel:   "session",
		Price:          100,
		TimeUnit:       "minute",
		DataUnit:       "GB",
		PaymentAddr:    "bc1qxy...z5m",
		EnableNAT:      true,
		EnableUPnP:     true,
		NATPMP:         true,
		Isolation:      "none",
		HealthCheck:    true,
		HealthInterval: "30s",
		Announced:      true,
		LastUpdate:     "2 hours ago",
	}
}

func (m ProviderModel) Init() tea.Cmd {
	return nil
}

func (m ProviderModel) Update(msg tea.Msg) (ProviderModel, tea.Cmd) {
	return m, nil
}

func (m ProviderModel) View() string {
	if m.Running {
		return m.renderRunning()
	}
	return m.renderConfig()
}

func (m ProviderModel) renderConfig() string {
	fields := []string{
		fmt.Sprintf("Country:             [%s        ]  Auto-detect: [%c]", m.Country, boolToChar(m.AutoDetect)),
		"",
		"Connection Limits",
		fmt.Sprintf("  Max Consumers:    [%d    ] (0 = unlimited)", m.MaxConsumers),
		fmt.Sprintf("  Max Duration:    [%d    ] hours (0 = unlimited)", m.MaxDuration),
		"",
		"Bandwidth",
		fmt.Sprintf("  Limit:           [%s  ]", m.BandwidthLimit),
		fmt.Sprintf("  Advertised:      [%d     ] Mbps", m.AdvertisedMbps),
		"",
		fmt.Sprintf("Payment Model        [%s *] [TIME] [DATA]", m.PaymentModel),
		fmt.Sprintf("  Price:           [%d   ] satoshis", m.Price),
		fmt.Sprintf("  Time Unit:       [%s ] (minute/hour)", m.TimeUnit),
		fmt.Sprintf("  Data Unit:       [%s   ] (MB/GB)", m.DataUnit),
		"",
		fmt.Sprintf("Payment Address:     [%s ]", m.PaymentAddr),
		"",
		"Advanced Options",
		fmt.Sprintf("  Enable NAT:       [%c]  UPnP [%c] NAT-PMP [%c]", boolToChar(m.EnableNAT), boolToChar(m.EnableUPnP), boolToChar(m.NATPMP)),
		fmt.Sprintf("  Isolation:        [%s   ] (none/sandbox)", m.Isolation),
		fmt.Sprintf("  Health Checks:   [%c]  Interval: [%s]", boolToChar(m.HealthCheck), m.HealthInterval),
	}

	var result string
	for i, field := range fields {
		prefix := "  "

		displayField := field
		if m.Editing {
			for idx, pf := range providerFields {
				if contains(field, pf) && idx == m.ActiveField {
					displayField = activeFieldStyle.Render(field)
					prefix = activeFieldStyle.Render(" >")
					break
				}
			}
		}

		if i > 0 && i < len(fields)-1 {
			result += prefix + displayField + "\n"
		} else {
			result += displayField + "\n"
		}
	}

	result += "\n BLOCKCHAIN STATUS\n"
	result += " =================\n"
	result += fmt.Sprintf(" Announcement:     %s\n", statusIndicator(m.Announced, "ANNOUNCED", "NOT ANNOUNCED"))
	result += fmt.Sprintf(" Last Update:      %s\n", m.LastUpdate)
	result += " Re-announce:      [Force Re-announce]\n"
	result += "\n"
	result += "    [ Validate ]  [ Save Config ]  [ START ]  [ STOP ]"

	return result
}

func (m ProviderModel) renderRunning() string {
	return ` PROVIDER - RUNNING
 ====================

 ● LISTENING ON PORT 51820 (TLS)
 ● NAT PORT MAPPED (TCP:51820, UDP:51820)
 ● ANNOUNCED TO BLOCKCHAIN
 ● HEALTH CHECKS ACTIVE

 ============================================================

 Active Connections:
 +----+-----------------+---------+-----------+---------+-------+
 | ID | IP Address     | Country | Download  | Upload  | Time  |
 +----+-----------------+---------+-----------+---------+-------+
 | 01 | 10.0.0.2       | DE      | 45.2 MB  | 12.1 MB | 5:23  |
 | 02 | 10.0.0.3       | FR      | 12.8 MB  | 3.2 MB  | 2:10  |
 +----+-----------------+---------+-----------+---------+-------+

 ============================================================

 Revenue This Session: 450 sats
 Total Revenue: 12,450 sats

 ====================

              [ UPDATE PRICE ]  [ STOP PROVIDER ]`
}

func boolToChar(b bool) rune {
	if b {
		return '●'
	}
	return '○'
}

func statusIndicator(active bool, activeStr, inactiveStr string) string {
	if active {
		return successStyle.Render("● ") + activeStr
	}
	return errorStyle.Render("○ ") + inactiveStr
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && contains(s[1:], substr)))
}
