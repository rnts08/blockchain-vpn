package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	fieldStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747")).Bold(true)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#dca747")).Bold(true)
	buttonStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Background(lipgloss.Color("#212121")).Padding(0, 1)
	buttonSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("#212121")).Background(lipgloss.Color("#dca747")).Bold(true).Padding(0, 1)
)

type FieldType int

const (
	FieldToggle FieldType = iota
	FieldDropdown
	FieldNumber
	FieldText
	FieldDuration
	FieldButton
	FieldAction
)

type ProviderField struct {
	Name     string
	Type     FieldType
	Value    interface{}
	Options  []string
	Editable bool
	Section  string
}

type ProviderModel struct {
	Width       int
	Height      int
	Selected    int
	Editing     int
	Running     bool
	Fields      []ProviderField
	DropdownIdx int
	ButtonIdx   int
	InputBuffer string
}

var countries = []string{
	"US", "CA", "GB", "DE", "FR", "NL", "JP", "AU", "SG", "CH",
	"SE", "NO", "FI", "DK", "IT", "ES", "PT", "BR", "MX", "IN",
}

var paymentModels = []string{"session", "time", "data"}

var timeUnits = []string{"minute", "hour"}

var dataUnits = []string{"MB", "GB", "TB"}

var durationUnits = []string{"minutes", "hours"}

func NewProviderModel() ProviderModel {
	m := ProviderModel{
		Width:    80,
		Height:   24,
		Selected: 0,
		Editing:  -1,
		Running:  false,
		Fields: []ProviderField{
			{Name: "Country", Type: FieldDropdown, Value: "US", Options: countries, Editable: true, Section: "Location"},
			{Name: "Auto-detect Country", Type: FieldAction, Value: "Detect", Editable: true, Section: "Location"},
			{Name: "Auto-detect", Type: FieldToggle, Value: true, Editable: true, Section: "Location"},
			{Name: "Max Consumers", Type: FieldNumber, Value: 10, Editable: true, Section: "Connection Limits"},
			{Name: "Max Duration", Type: FieldDuration, Value: 0, Options: durationUnits, Editable: true, Section: "Connection Limits"},
			{Name: "Bandwidth Limit", Type: FieldText, Value: "10mbit", Editable: true, Section: "Bandwidth"},
			{Name: "Advertised Mbps", Type: FieldNumber, Value: 10, Editable: true, Section: "Bandwidth"},
			{Name: "Speed Test", Type: FieldAction, Value: "Test", Editable: true, Section: "Bandwidth"},
			{Name: "Payment Model", Type: FieldDropdown, Value: "session", Options: paymentModels, Editable: true, Section: "Payment"},
			{Name: "Price (sats)", Type: FieldNumber, Value: 100, Editable: true, Section: "Payment"},
			{Name: "Time Unit", Type: FieldDropdown, Value: "minute", Options: timeUnits, Editable: true, Section: "Payment"},
			{Name: "Data Unit", Type: FieldDropdown, Value: "GB", Options: dataUnits, Editable: true, Section: "Payment"},
			{Name: "Payment Address", Type: FieldText, Value: "bc1qxy2kz8dug7n9r4jdq2xkn9m8x5y2z", Editable: true, Section: "Payment"},
			{Name: "Enable NAT", Type: FieldToggle, Value: true, Editable: true, Section: "Advanced"},
			{Name: "Enable UPnP", Type: FieldToggle, Value: true, Editable: true, Section: "Advanced"},
			{Name: "NAT-PMP", Type: FieldToggle, Value: true, Editable: true, Section: "Advanced"},
			{Name: "Isolation", Type: FieldDropdown, Value: "none", Options: []string{"none", "sandbox"}, Editable: true, Section: "Advanced"},
			{Name: "Health Check", Type: FieldToggle, Value: true, Editable: true, Section: "Advanced"},
			{Name: "Health Interval", Type: FieldText, Value: "30s", Editable: true, Section: "Advanced"},
		},
		DropdownIdx: 0,
		ButtonIdx:   0,
	}
	m.updateDynamicFields()
	return m
}

func (m *ProviderModel) updateDynamicFields() {
}

func (m ProviderModel) Init() tea.Cmd {
	return nil
}

func (m ProviderModel) Update(msg tea.Msg) (ProviderModel, tea.Cmd) {
	_ = msg
	if m.Running && m.Selected >= len(m.Fields) {
		m.Selected = len(m.Fields)
	}
	return m, nil
}

func (m *ProviderModel) MoveSelection(delta int) {
	if m.Running {
		if m.Selected == len(m.Fields) {
			m.Selected = (m.Selected + delta + 1)
			if m.Selected > len(m.Fields) {
				m.Selected = len(m.Fields)
			}
		}
		return
	}

	numFields := len(m.Fields) + 4
	if m.Selected >= len(m.Fields) && m.Selected < numFields {
		m.ButtonIdx = m.Selected - len(m.Fields)
		m.Selected = (m.Selected + delta + numFields) % numFields
		if m.Selected >= len(m.Fields) {
			m.ButtonIdx = m.Selected - len(m.Fields)
		}
	} else {
		m.Selected = (m.Selected + delta + numFields) % numFields
		if m.Selected >= len(m.Fields) {
			m.ButtonIdx = m.Selected - len(m.Fields)
		}
	}
}

func (m *ProviderModel) Activate() {
	if m.Running {
		if m.Selected == len(m.Fields) {
			fmt.Println("STOP PROVIDER")
		}
		return
	}

	if m.Selected >= len(m.Fields) {
		btn := m.Selected - len(m.Fields)
		switch btn {
		case 0:
			fmt.Println("VALIDATE")
		case 1:
			fmt.Println("SAVE")
		case 2:
			fmt.Println("START PROVIDER")
			m.Running = true
		case 3:
			fmt.Println("STOP PROVIDER")
		}
		return
	}

	field := &m.Fields[m.Selected]
	if !field.Editable {
		return
	}

	switch field.Type {
	case FieldToggle:
		if field.Name == "Auto-detect" && field.Value == true {
			field.Value = false
		} else {
			field.Value = !field.Value.(bool)
		}
	case FieldDropdown:
		m.DropdownIdx = 0
		m.Editing = m.Selected
	case FieldNumber, FieldText, FieldDuration:
		m.Editing = m.Selected
	case FieldAction:
		if field.Name == "Auto-detect Country" {
			fmt.Println("DETECT COUNTRY")
		} else if field.Name == "Speed Test" {
			fmt.Println("SPEED TEST")
		}
	}
}

func (m *ProviderModel) HandleEditKey(key string) {
	if m.Editing < 0 || m.Editing >= len(m.Fields) {
		return
	}
	field := &m.Fields[m.Editing]

	switch field.Type {
	case FieldDropdown:
		if key == "down" || key == "j" {
			m.DropdownIdx = (m.DropdownIdx + 1) % len(field.Options)
			field.Value = field.Options[m.DropdownIdx]
		} else if key == "up" || key == "k" {
			m.DropdownIdx = (m.DropdownIdx - 1 + len(field.Options)) % len(field.Options)
			field.Value = field.Options[m.DropdownIdx]
		} else if key == "enter" || key == "esc" {
			m.Editing = -1
			m.InputBuffer = ""
		}
	case FieldNumber:
		if key >= "0" && key <= "9" {
			m.InputBuffer += key
		} else if key == "up" || key == "k" {
			if m.InputBuffer != "" {
				val := 0
				fmt.Sscanf(m.InputBuffer, "%d", &val)
				field.Value = val
				m.InputBuffer = ""
			} else {
				field.Value = field.Value.(int) + 1
			}
		} else if key == "down" || key == "j" {
			if m.InputBuffer != "" {
				val := 0
				fmt.Sscanf(m.InputBuffer, "%d", &val)
				if val > 0 {
					field.Value = val
				}
				m.InputBuffer = ""
			} else if field.Value.(int) > 0 {
				field.Value = field.Value.(int) - 1
			}
		} else if key == "enter" {
			if m.InputBuffer != "" {
				val := 0
				fmt.Sscanf(m.InputBuffer, "%d", &val)
				field.Value = val
				m.InputBuffer = ""
			}
			m.Editing = -1
			m.Selected = (m.Selected + 1) % len(m.Fields)
		} else if key == "esc" {
			m.Editing = -1
			m.InputBuffer = ""
		}
	case FieldText:
		m.Editing = -1
	case FieldDuration:
		if key == "up" || key == "k" {
			field.Value = field.Value.(int) + 1
		} else if key == "down" || key == "j" {
			if field.Value.(int) > 0 {
				field.Value = field.Value.(int) - 1
			}
		} else if key == "enter" || key == "esc" {
			m.Editing = -1
		}
	}
}

func (m *ProviderModel) CancelEdit() {
	m.Editing = -1
}

func (m ProviderModel) View() string {
	if m.Running {
		return m.renderRunning()
	}
	return m.renderConfig()
}

func (m ProviderModel) renderConfig() string {
	var lines []string

	currentSection := ""
	for i, field := range m.Fields {
		if field.Section != currentSection {
			if currentSection != "" {
				lines = append(lines, "")
			}
			lines = append(lines, headerStyle.Render(" "+strings.ToUpper(field.Section)))
			lines = append(lines, headerStyle.Render(" "+strings.Repeat("─", len(field.Section))))
			currentSection = field.Section
		}

		selected := m.Selected == i
		editing := m.Editing == i
		lines = append(lines, m.renderField(field, selected, editing))
	}

	lines = append(lines, "")
	lines = append(lines, headerStyle.Render(" BLOCKCHAIN STATUS"))
	lines = append(lines, headerStyle.Render(" ================"))
	lines = append(lines, " "+successStyle.Render("●")+" Announcement:     ANNOUNCED (height: 842,153)")
	lines = append(lines, " Last Update:        2 hours ago")
	lines = append(lines, "")

	buttons := []string{"Validate", "Save", "START", "Stop"}
	btnLines := "    "
	for i, btn := range buttons {
		if m.Selected == len(m.Fields)+i {
			btnLines += buttonSelected.Render(" "+btn+" ") + "  "
		} else {
			btnLines += buttonStyle.Render(" "+btn+" ") + "  "
		}
	}
	lines = append(lines, btnLines)

	return joinLines(lines)
}

func (m ProviderModel) renderField(field ProviderField, selected, editing bool) string {
	prefix := "  "
	valueStyle := fieldStyle
	if selected {
		prefix = "▶ "
		valueStyle = selectedStyle
	}

	switch field.Type {
	case FieldToggle:
		marker := "○"
		if field.Value.(bool) {
			marker = successStyle.Render("●")
		}
		if editing {
			marker = warningStyle.Render("[■]")
		}
		return prefix + valueStyle.Render(field.Name+":") + " " + marker

	case FieldDropdown:
		val := fmt.Sprintf("%v", field.Value)
		arrow := "▼"
		if editing {
			arrow = warningStyle.Render("▼")
		}
		return prefix + valueStyle.Render(field.Name+":") + " [" + val + "]" + arrow

	case FieldNumber:
		val := fmt.Sprintf("%v", field.Value)
		if m.InputBuffer != "" && editing {
			return prefix + valueStyle.Render(field.Name+":") + " [" + warningStyle.Render(m.InputBuffer+"_") + "]"
		}
		if field.Name == "Max Duration" {
			opts := field.Options
			durationIdx := 0
			if val == "0" {
				durationIdx = 0
			} else {
				durationIdx = 1
			}
			return prefix + valueStyle.Render(field.Name+":") + " [" + val + "] " + opts[durationIdx]
		}
		if editing {
			return prefix + valueStyle.Render(field.Name+":") + " [" + warningStyle.Render(val) + "]"
		}
		return prefix + valueStyle.Render(field.Name+":") + " [" + val + "]"

	case FieldText:
		val := field.Value.(string)
		if editing {
			return prefix + valueStyle.Render(field.Name+":") + " [" + warningStyle.Render("________") + "]"
		}
		return prefix + valueStyle.Render(field.Name+":") + " [" + val + "]"

	case FieldDuration:
		val := field.Value.(int)
		opts := field.Options
		unitIdx := 0
		if val >= 60 {
			unitIdx = 1
		}
		if editing {
			return prefix + valueStyle.Render(field.Name+":") + " [" + warningStyle.Render(fmt.Sprintf("%d", val)) + "] " + opts[unitIdx]
		}
		return prefix + valueStyle.Render(field.Name+":") + " [" + fmt.Sprintf("%d", val) + "] " + opts[unitIdx]

	case FieldAction:
		actionVal := field.Value.(string)
		if selected {
			return fmt.Sprintf("%s %s[%s]", prefix, field.Name+": ", actionVal)
		}
		return fmt.Sprintf("%s %s %s", prefix, field.Name+":", actionVal)

	default:
		return fmt.Sprintf("%s %s: %v", prefix, field.Name, field.Value)
	}
}

func (m ProviderModel) renderRunning() string {
	lines := []string{
		" PROVIDER - RUNNING",
		" ==================",
		"",
		" ● LISTENING ON PORT 51820 (TLS)",
		" ● NAT PORT MAPPED (TCP:51820, UDP:51820)",
		" ● ANNOUNCED TO BLOCKCHAIN",
		" ● HEALTH CHECKS ACTIVE",
		"",
		" Active Connections:",
		" +----+-----------------+---------+-----------+---------+-------+",
		" | ID | IP Address     | Country | Download  | Upload  | Time  |",
		" +----+-----------------+---------+-----------+---------+-------+",
		" | 01 | 10.0.0.2       | DE      | 45.2 MB  | 12.1 MB | 5:23  |",
		" | 02 | 10.0.0.3       | FR      | 12.8 MB  | 3.2 MB  | 2:10  |",
		" +----+-----------------+---------+-----------+---------+-------+",
		"",
		" Revenue This Session: 450 sats",
		" Total Revenue: 12,450 sats",
		"",
	}

	btnLines := "               "
	buttons := []string{"UPDATE PRICE", "STOP PROVIDER"}
	for i, btn := range buttons {
		if m.Selected == len(m.Fields)+i {
			btnLines += focusedStyle.Render(" "+btn+" ") + "  "
		} else {
			btnLines += tabStyle.Render(" "+btn+" ") + "  "
		}
	}
	lines = append(lines, btnLines)

	return joinLines(lines)
}

func selectable(selected bool, text string) string {
	if selected {
		return "▶ " + text
	}
	return "  " + text
}

func announceStatus(announced bool) string {
	if announced {
		return " ● Announcement:     ANNOUNCED (height: 842,153)"
	}
	return " ○ Announcement:     NOT ANNOUNCED"
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
