package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MrHalder/moor/internal/process"
	"github.com/MrHalder/moor/internal/scanner"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View states
type viewState int

const (
	viewTable viewState = iota
	viewActions
	viewDetail
	viewFilter
	viewConfirmKill
)

// Action menu items
type action int

const (
	actionKill action = iota
	actionForceKill
	actionDetail
	actionCopyPID
	actionBack
	actionCount
)

func (a action) String() string {
	switch a {
	case actionKill:
		return "Kill (graceful SIGTERM)"
	case actionForceKill:
		return "Force Kill (SIGKILL)"
	case actionDetail:
		return "View Details"
	case actionCopyPID:
		return "Copy PID"
	case actionBack:
		return "Back"
	default:
		return ""
	}
}

func (a action) Icon() string {
	switch a {
	case actionKill:
		return "  "
	case actionForceKill:
		return "  "
	case actionDetail:
		return "  "
	case actionCopyPID:
		return "  "
	case actionBack:
		return "  "
	default:
		return "  "
	}
}

// Sort columns
type sortColumn int

const (
	sortByPort sortColumn = iota
	sortByPID
	sortByProcess
	sortByProto
	sortByState
	sortColumnCount
)

func (s sortColumn) String() string {
	switch s {
	case sortByPort:
		return "port"
	case sortByPID:
		return "pid"
	case sortByProcess:
		return "process"
	case sortByProto:
		return "proto"
	case sortByState:
		return "state"
	default:
		return "port"
	}
}

// Messages
type tickMsg time.Time

type scanResultMsg struct {
	ports          []scanner.PortInfo
	needsElevation bool
	err            error
}

type killResultMsg struct {
	port    uint16
	pid     int32
	name    string
	success bool
	err     error
}

// Model is the top-level Bubble Tea model.
type Model struct {
	scanner    scanner.PortScanner
	procMgr    process.Manager
	theme      Theme
	keys       KeyMap
	help       help.Model
	filterInput textinput.Model

	// State
	view           viewState
	ports          []scanner.PortInfo
	filteredPorts  []scanner.PortInfo
	cursor         int
	sortCol        sortColumn
	showAll        bool
	needsElevation bool
	filterText     string
	statusMsg      string
	statusExpiry   time.Time
	showHelp       bool
	width          int
	height         int
	refreshInterval time.Duration

	// Action menu
	actionCursor int

	// Kill confirmation
	killTarget  *scanner.PortInfo
	killForce   bool
	gracePeriod time.Duration
}

// New creates a new TUI Model.
func New(refreshInterval, gracePeriod time.Duration) Model {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.CharLimit = 50

	h := help.New()
	h.ShowAll = false

	mgr := process.NewManager()
	if gracePeriod > 0 {
		mgr.GracePeriod = gracePeriod
	}

	return Model{
		scanner:         scanner.NewScanner(),
		procMgr:         mgr,
		theme:           DefaultTheme(),
		keys:            DefaultKeyMap(),
		help:            h,
		filterInput:     ti,
		view:            viewTable,
		sortCol:         sortByPort,
		refreshInterval: refreshInterval,
		gracePeriod:     gracePeriod,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.scanCmd(),
		m.tickCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tickMsg:
		return m, m.scanCmd()

	case scanResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("scan error: %v", msg.err)
			m.statusExpiry = time.Now().Add(5 * time.Second)
		} else {
			m.ports = msg.ports
			m.needsElevation = msg.needsElevation
			m.applyFilterAndSort()
		}
		return m, m.tickCmd()

	case killResultMsg:
		if msg.success {
			m.statusMsg = fmt.Sprintf("Killed '%s' (PID %d) on port %d", sanitizeDisplay(msg.name), msg.pid, msg.port)
		} else {
			m.statusMsg = fmt.Sprintf("Failed to kill PID %d: %v", msg.pid, msg.err)
		}
		m.statusExpiry = time.Now().Add(5 * time.Second)
		m.view = viewTable
		m.killTarget = nil
		return m, m.scanCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filter input mode
	if m.view == viewFilter {
		return m.handleFilterKey(msg)
	}

	// Kill confirmation mode
	if m.view == viewConfirmKill {
		return m.handleConfirmKillKey(msg)
	}

	// Action menu
	if m.view == viewActions {
		return m.handleActionsKey(msg)
	}

	// Detail view
	if m.view == viewDetail {
		switch {
		case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
			m.view = viewTable
			return m, nil
		case key.Matches(msg, m.keys.Kill):
			if m.cursor < len(m.filteredPorts) {
				target := m.filteredPorts[m.cursor]
				m.killTarget = &target
				m.killForce = false
				m.view = viewConfirmKill
			}
			return m, nil
		case key.Matches(msg, m.keys.ForceKill):
			if m.cursor < len(m.filteredPorts) {
				target := m.filteredPorts[m.cursor]
				m.killTarget = &target
				m.killForce = true
				m.view = viewConfirmKill
			}
			return m, nil
		}
		return m, nil
	}

	// Table view
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filteredPorts)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Detail):
		if m.cursor < len(m.filteredPorts) {
			m.actionCursor = 0
			m.view = viewActions
		}

	case key.Matches(msg, m.keys.Kill):
		if m.cursor < len(m.filteredPorts) {
			target := m.filteredPorts[m.cursor]
			m.killTarget = &target
			m.killForce = false
			m.view = viewConfirmKill
		}

	case key.Matches(msg, m.keys.ForceKill):
		if m.cursor < len(m.filteredPorts) {
			target := m.filteredPorts[m.cursor]
			m.killTarget = &target
			m.killForce = true
			m.view = viewConfirmKill
		}

	case key.Matches(msg, m.keys.Filter):
		m.view = viewFilter
		m.filterInput.SetValue(m.filterText)
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Sort):
		m.sortCol = (m.sortCol + 1) % sortColumnCount
		m.applyFilterAndSort()
		m.statusMsg = fmt.Sprintf("Sort: %s", m.sortCol)
		m.statusExpiry = time.Now().Add(2 * time.Second)

	case key.Matches(msg, m.keys.ToggleAll):
		m.showAll = !m.showAll
		if m.showAll {
			m.statusMsg = "Showing all connections"
		} else {
			m.statusMsg = "Showing LISTEN only"
		}
		m.statusExpiry = time.Now().Add(2 * time.Second)
		return m, m.scanCmd()

	case key.Matches(msg, m.keys.Refresh):
		m.statusMsg = "Refreshing..."
		m.statusExpiry = time.Now().Add(1 * time.Second)
		return m, m.scanCmd()

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
	}

	return m, nil
}

func (m Model) handleActionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.actionCursor > 0 {
			m.actionCursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.actionCursor < int(actionCount)-1 {
			m.actionCursor++
		}
	case key.Matches(msg, m.keys.Detail): // Enter
		return m.executeAction(action(m.actionCursor))
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
		m.view = viewTable
	}
	return m, nil
}

func (m Model) executeAction(a action) (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.filteredPorts) {
		m.view = viewTable
		return m, nil
	}

	target := m.filteredPorts[m.cursor]

	switch a {
	case actionKill:
		m.killTarget = &target
		m.killForce = false
		m.view = viewConfirmKill
	case actionForceKill:
		m.killTarget = &target
		m.killForce = true
		m.view = viewConfirmKill
	case actionDetail:
		m.view = viewDetail
	case actionCopyPID:
		m.statusMsg = fmt.Sprintf("PID %d (copy not available in TUI — use: echo %d | pbcopy)", target.PID, target.PID)
		m.statusExpiry = time.Now().Add(5 * time.Second)
		m.view = viewTable
	case actionBack:
		m.view = viewTable
	}
	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filterText = m.filterInput.Value()
		m.view = viewTable
		m.filterInput.Blur()
		m.applyFilterAndSort()
		m.cursor = 0
		return m, nil
	case "esc":
		m.filterText = ""
		m.view = viewTable
		m.filterInput.Blur()
		m.filterInput.SetValue("")
		m.applyFilterAndSort()
		m.cursor = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	// Live filter as you type
	m.filterText = m.filterInput.Value()
	m.applyFilterAndSort()
	m.cursor = 0
	return m, cmd
}

func (m Model) handleConfirmKillKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		if m.killTarget != nil && m.killTarget.PID > 0 {
			target := *m.killTarget
			force := m.killForce
			gracePeriod := m.gracePeriod
			return m, func() tea.Msg {
				mgr := process.NewManager()
				if gracePeriod > 0 {
					mgr.GracePeriod = gracePeriod
				}
				err := mgr.Kill(context.Background(), target.PID, force)
				return killResultMsg{
					port:    target.LocalPort,
					pid:     target.PID,
					name:    target.ProcessName,
					success: err == nil,
					err:     err,
				}
			}
		}
		m.view = viewTable
		m.killTarget = nil
		return m, nil
	case key.Matches(msg, m.keys.Cancel):
		m.view = viewTable
		m.killTarget = nil
		return m, nil
	}
	return m, nil
}

func (m *Model) applyFilterAndSort() {
	filtered := make([]scanner.PortInfo, 0, len(m.ports))

	for _, p := range m.ports {
		if !m.showAll && !strings.EqualFold(p.State, "LISTEN") {
			continue
		}
		if m.filterText != "" {
			search := strings.ToLower(m.filterText)
			haystack := strings.ToLower(fmt.Sprintf("%s %s %d %d %s %s %s",
				p.Protocol, p.LocalAddr, p.LocalPort, p.PID, p.ProcessName, p.User, p.State))
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, p)
	}

	sort.Slice(filtered, func(i, j int) bool {
		switch m.sortCol {
		case sortByPID:
			return filtered[i].PID < filtered[j].PID
		case sortByProcess:
			return strings.ToLower(filtered[i].ProcessName) < strings.ToLower(filtered[j].ProcessName)
		case sortByProto:
			return filtered[i].Protocol < filtered[j].Protocol
		case sortByState:
			return filtered[i].State < filtered[j].State
		default: // sortByPort
			if filtered[i].LocalPort != filtered[j].LocalPort {
				return filtered[i].LocalPort < filtered[j].LocalPort
			}
			return filtered[i].Protocol < filtered[j].Protocol
		}
	})

	m.filteredPorts = filtered
	if m.cursor >= len(m.filteredPorts) {
		m.cursor = max(0, len(m.filteredPorts)-1)
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Title
	title := m.theme.Title.Render("moor")
	mode := "LISTEN"
	if m.showAll {
		mode = "ALL"
	}
	subtitle := m.theme.Subtitle.Render(fmt.Sprintf(" %d ports [%s] sort:%s", len(m.filteredPorts), mode, m.sortCol))
	sections = append(sections, title+subtitle)

	// Table or detail or confirmation
	switch m.view {
	case viewActions:
		sections = append(sections, m.renderActions())
	case viewDetail:
		sections = append(sections, m.renderDetail())
	case viewConfirmKill:
		sections = append(sections, m.renderTable())
		sections = append(sections, m.renderKillConfirm())
	case viewFilter:
		sections = append(sections, m.renderTable())
		sections = append(sections, m.renderFilterBar())
	default:
		sections = append(sections, m.renderTable())
	}

	// Status bar
	sections = append(sections, m.renderStatusBar())

	// Help
	sections = append(sections, m.help.View(m.keys))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderTable() string {
	if len(m.filteredPorts) == 0 {
		return m.theme.Dim.Render("  No ports to display.\n")
	}

	// Calculate available rows for the table
	tableHeight := m.height - 8 // title + status + help + padding
	if m.showHelp {
		tableHeight -= 4
	}
	if tableHeight < 3 {
		tableHeight = 3
	}

	// Column widths
	cols := []struct {
		name  string
		width int
	}{
		{"PROTO", 6},
		{"ADDRESS", 20},
		{"PORT", 6},
		{"PID", 8},
		{"PROCESS", 24},
		{"USER", 12},
		{"STATE", 12},
	}

	// Adjust widths to available space
	totalWidth := 0
	for _, c := range cols {
		totalWidth += c.width + 2
	}

	var sb strings.Builder

	// Header
	headerParts := make([]string, len(cols))
	for i, c := range cols {
		label := c.name
		if sortColumn(i) == m.sortCol {
			label += " ▼"
		}
		headerParts[i] = fmt.Sprintf("%-*s", c.width, label)
	}
	sb.WriteString(m.theme.Header.Render(strings.Join(headerParts, "  ")))
	sb.WriteString("\n")

	// Rows
	start := 0
	if m.cursor >= tableHeight {
		start = m.cursor - tableHeight + 1
	}
	end := min(start+tableHeight, len(m.filteredPorts))

	for i := start; i < end; i++ {
		p := m.filteredPorts[i]
		row := fmt.Sprintf("%-6s  %-20s  %-6d  %-8s  %-24s  %-12s  %-12s",
			p.Protocol,
			truncate(sanitizeDisplay(p.LocalAddr), 20),
			p.LocalPort,
			pidStr(p.PID),
			truncate(sanitizeDisplay(p.ProcessName), 24),
			truncate(sanitizeDisplay(p.User), 12),
			p.State,
		)

		if i == m.cursor {
			sb.WriteString(m.theme.SelectedRow.Render(row))
		} else {
			sb.WriteString(m.stateStyle(p.State).Render(row))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Model) renderDetail() string {
	if m.cursor >= len(m.filteredPorts) {
		return ""
	}

	p := m.filteredPorts[m.cursor]
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(m.theme.Title.Render(fmt.Sprintf("Port %d Detail", p.LocalPort)))
	sb.WriteString("\n")

	details := []struct {
		key, value string
	}{
		{"Protocol", p.Protocol},
		{"Local Address", fmt.Sprintf("%s:%d", p.LocalAddr, p.LocalPort)},
		{"Remote", fmt.Sprintf("%s:%d", p.RemoteAddr, p.RemotePort)},
		{"State", p.State},
		{"PID", pidStr(p.PID)},
		{"Process", p.ProcessName},
		{"User", p.User},
		{"Command", sanitizeDisplay(p.CommandLine)},
	}

	for _, d := range details {
		if d.value == "" || d.value == ":0" {
			continue
		}
		line := m.theme.DetailKey.Render(d.key) + m.theme.DetailValue.Render(d.value)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.theme.Help.Render("  esc back  k kill  K force kill"))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderActions() string {
	if m.cursor >= len(m.filteredPorts) {
		return ""
	}

	p := m.filteredPorts[m.cursor]
	var sb strings.Builder

	// Port header
	name := sanitizeDisplay(p.ProcessName)
	if name == "" {
		name = "unknown"
	}

	sb.WriteString("\n")
	header := fmt.Sprintf("  %s:%d  (%s, PID %s)", sanitizeDisplay(p.LocalAddr), p.LocalPort, name, pidStr(p.PID))
	sb.WriteString(m.theme.Title.Render(header))
	sb.WriteString("\n\n")
	sb.WriteString(m.theme.Subtitle.Render("  Select an action:"))
	sb.WriteString("\n\n")

	// Action items
	for i := 0; i < int(actionCount); i++ {
		a := action(i)
		cursor := "   "
		style := m.theme.Dim
		if i == m.actionCursor {
			cursor = " > "
			style = m.theme.SelectedRow
		}
		line := fmt.Sprintf("%s%s%s", cursor, a.Icon(), a.String())
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.theme.Help.Render("  ↑/↓ navigate  enter select  esc back"))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderKillConfirm() string {
	if m.killTarget == nil {
		return ""
	}

	method := "SIGTERM (graceful)"
	if m.killForce {
		method = "SIGKILL (force)"
	}

	name := sanitizeDisplay(m.killTarget.ProcessName)
	if name == "" {
		name = "unknown"
	}

	msg := fmt.Sprintf("\n  Kill '%s' (PID %d) on port %d with %s? [y/n] ",
		name, m.killTarget.PID, m.killTarget.LocalPort, method)

	return m.theme.Warning.Render(msg)
}

func (m Model) renderFilterBar() string {
	return fmt.Sprintf("\n  Filter: %s", m.filterInput.View())
}

func (m Model) renderStatusBar() string {
	var parts []string

	if m.filterText != "" {
		parts = append(parts, m.theme.StatusKey.Render("filter:")+m.theme.StatusValue.Render(m.filterText))
	}

	if m.needsElevation {
		parts = append(parts, m.theme.Warning.Render("limited view — use sudo"))
	}

	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		parts = append(parts, m.theme.StatusValue.Render(m.statusMsg))
	}

	if len(parts) == 0 {
		return ""
	}

	return m.theme.StatusBar.Render(strings.Join(parts, "  │  "))
}

func (m Model) stateStyle(state string) lipgloss.Style {
	switch strings.ToUpper(state) {
	case "LISTEN":
		return m.theme.Listen
	case "ESTABLISHED":
		return m.theme.Established
	default:
		return m.theme.Dim
	}
}

// Commands

func (m Model) scanCmd() tea.Cmd {
	showAll := m.showAll
	s := m.scanner
	return func() tea.Msg {
		ctx := context.Background()
		var result *scanner.ScanResult
		var err error
		if showAll {
			result, err = s.ListAll(ctx)
		} else {
			result, err = s.ListListening(ctx)
		}
		if err != nil {
			return scanResultMsg{err: err}
		}
		return scanResultMsg{
			ports:          result.Ports,
			needsElevation: result.NeedsElevation,
		}
	}
}

func (m Model) tickCmd() tea.Cmd {
	interval := m.refreshInterval
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Helpers

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func sanitizeDisplay(s string) string {
	return strings.Map(func(r rune) rune {
		// Strip C0 control characters, DEL, and C1 control characters
		// C1 (U+0080–U+009F) includes OSC, CSI, DCS which can manipulate terminals
		if r < 32 || r == 127 || (r >= 0x80 && r <= 0x9F) {
			return -1
		}
		return r
	}, s)
}

func pidStr(pid int32) string {
	if pid <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d", pid)
}
