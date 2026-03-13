package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type ProviderModel struct {
	Width          int
	Height         int
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
	return ` ╔═══════════════════════════════════════════════════════════════════════╗ 
 ║                    PROVIDER CONFIGURATION                            ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║                                                                        ║ 
 ║  Country:             [US        ▼]  Auto-detect: [●]                ║ 
 ║                                                                        ║ 
 ║  Connection Limits                                                      ║ 
 ║  ├─ Max Consumers:    [10    ] (0 = unlimited)                       ║ 
 ║  └─ Max Duration:    [0    ] hours (0 = unlimited)                    ║ 
 ║                                                                        ║ 
 ║  Bandwidth                                                           ║ 
 ║  ├─ Limit:           [10mbit  ]                                       ║ 
 ║  └─ Advertised:      [10     ] Mbps                                   ║ 
 ║                                                                        ║ 
 ║  Payment Model        [SESSION ●] [TIME] [DATA]                      ║ 
 ║  ├─ Price:           [100   ] satoshis                                ║ 
 ║  ├─ Time Unit:       [MINUTE ▼] (minute/hour)                        ║ 
 ║  └─ Data Unit:       [MB    ▼] (MB/GB)                               ║ 
 ║                                                                        ║ 
 ║  Payment Address:     [bc1qxy...z5m ]  [Generate New]                ║ 
 ║                                                                        ║ 
 ║  Advanced Options                                                     ║ 
 ║  ├─ Enable NAT:       [●]  UPnP [●] NAT-PMP                          ║ 
 ║  ├─ Isolation:        [NONE   ▼] (none/sandbox)                       ║ 
 ║  └─ Health Checks:   [●]  Interval: [30s]                            ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║  Blockchain Status                                                    ║ 
 ║  ├─ Announcement:     ANNOUNCED (height: 842,153)                    ║ 
 ║  ├─ Last Update:      2 hours ago                                    ║ 
 ║  └─ Re-announce:      [Force Re-announce]                           ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║                                                                        ║ 
 ║              [ Validate ]  [ Save Config ]  [ START ]  [ STOP ]      ║ 
 ║                                                                        ║ 
 ╚═══════════════════════════════════════════════════════════════════════╝`
}

func (m ProviderModel) renderRunning() string {
	return ` ╔═══════════════════════════════════════════════════════════════════════╗ 
 ║                    PROVIDER - RUNNING                                ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║                                                                        ║ 
 ║  ● LISTENING ON PORT 51820 (TLS)                                     ║ 
 ║  ● NAT PORT MAPPED (TCP:51820, UDP:51820)                           ║ 
 ║  ● ANNOUNCED TO BLOCKCHAIN                                            ║ 
 ║  ● HEALTH CHECKS ACTIVE                                               ║ 
 ║                                                                        ║ 
 ║  ═════════════════════════════════════════════════════════════════        ║ 
 ║                                                                        ║ 
 ║  Active Connections:                                                   ║ 
 ║  ┌────────────────────────────────────────────────────────────────┐    ║ 
 ║  │ ID   │ IP Address     │ Country │ Download  │ Upload  │ Time  │    ║ 
 ║  ├──────┼─────────────────┼─────────┼───────────┼─────────┼───────┤    ║ 
 ║  │ 001  │ 10.0.0.2       │ DE      │ 45.2 MB  │ 12.1 MB │ 5:23  │    ║ 
 ║  │ 002  │ 10.0.0.3       │ FR      │ 12.8 MB  │ 3.2 MB  │ 2:10  │    ║ 
 ║  └────────────────────────────────────────────────────────────────┘    ║ 
 ║                                                                        ║ 
 ║  ═════════════════════════════════════════════════════════════════        ║ 
 ║                                                                        ║ 
 ║  Revenue This Session: 450 sats                                       ║ 
 ║  Total Revenue: 12,450 sats                                           ║ 
 ║                                                                        ║ 
 ╠═══════════════════════════════════════════════════════════════════════╣ 
 ║                                                                        ║ 
 ║                       [ UPDATE PRICE ]  [ STOP PROVIDER ]               ║ 
 ║                                                                        ║ 
 ╚═══════════════════════════════════════════════════════════════════════╝`
}
