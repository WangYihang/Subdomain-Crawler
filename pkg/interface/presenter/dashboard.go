package presenter

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dashboard is a TUI dashboard for crawling progress
type Dashboard struct {
	metrics          *entity.Metrics
	recentSubdomains []string // Recent discovered subdomains
	width            int
	height           int
	startTime        time.Time
	mu               sync.RWMutex
}

type tickMsg time.Time

// NewDashboard creates a new TUI dashboard
func NewDashboard() *Dashboard {
	return &Dashboard{
		metrics:   &entity.Metrics{},
		startTime: time.Now(),
	}
}

// Init initializes the dashboard
func (d *Dashboard) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

// Update handles dashboard updates
func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "ctrl+c":
			return d, tea.Quit
		}

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		return d, nil

	case tickMsg:
		// Continue ticking to keep the display updating
		return d, tickCmd()
	}

	return d, nil
}

// View renders the dashboard
func (d *Dashboard) View() string {
	if d.width == 0 {
		return "Initializing..."
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	var sections []string

	// Header
	header := d.renderHeader()
	sections = append(sections, header)
	headerHeight := lipgloss.Height(header)

	// Footer
	footer := d.renderFooter()
	footerHeight := lipgloss.Height(footer)

	// Calculate dimensions for grid
	availableHeight := d.height - headerHeight - footerHeight
	if availableHeight < 0 {
		availableHeight = 0
	}
	halfHeight := availableHeight / 2

	halfWidth := d.width / 2
	leftWidth := halfWidth
	rightWidth := d.width - leftWidth

	// Row 1: General Stats (Left) | HTTP Stats (Right)
	row1 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		d.renderGeneralStats(leftWidth, halfHeight),
		d.renderHTTPStats(rightWidth, halfHeight),
	)
	sections = append(sections, row1)

	// Row 2: DNS Stats (Left) | Recent Discoveries (Right)
	// Adjust height for the second row to fill any remaining space due to integer division
	remainingHeight := availableHeight - halfHeight
	row2 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		d.renderDNSStats(leftWidth, remainingHeight),
		d.renderRecentDiscoveries(rightWidth, remainingHeight),
	)
	sections = append(sections, row2)

	// Footer
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// OnMetricsUpdate implements application.MetricsObserver
func (d *Dashboard) OnMetricsUpdate(metrics *entity.Metrics) {
	d.mu.Lock()
	d.metrics = metrics
	d.mu.Unlock()
}

func (d *Dashboard) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999"))

	elapsed := time.Since(d.startTime)
	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60
	seconds := int(elapsed.Seconds()) % 60

	var timeStr string
	if hours > 0 {
		timeStr = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		timeStr = fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		timeStr = fmt.Sprintf("%ds", seconds)
	}

	// Current time for reference
	now := time.Now().Format("15:04:05")

	title := titleStyle.Render("🔍 Subdomain Crawler")
	timeInfo := timeStyle.Render(fmt.Sprintf(" Running: %s | Time: %s", timeStr, now))

	return title + timeInfo
}

func (d *Dashboard) renderGeneralStats(width, height int) string {
	statStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		Width(width - 2).  // Adjust for border
		Height(height - 2) // Adjust for border

	stats := []string{
		"📊 General Statistics",
		"",
		fmt.Sprintf("Queue Length:      %d", d.metrics.QueueLength),
		fmt.Sprintf("Active Workers:    %d / %d", d.metrics.ActiveWorkers, d.metrics.TotalWorkers),
		fmt.Sprintf("Tasks Enqueued:    %d", d.metrics.TasksEnqueued),
		fmt.Sprintf("Tasks Processed:   %d", d.metrics.TasksProcessed),
		fmt.Sprintf("Unique Subdomains: %d", d.metrics.UniqueSubdomains),
		fmt.Sprintf("Errors:            %d", d.metrics.ErrorCount),
	}

	// Calculate task rate
	elapsed := time.Since(d.startTime).Seconds()
	if elapsed > 0 {
		taskRate := float64(d.metrics.TasksProcessed) / elapsed
		stats = append(stats,
			"",
			fmt.Sprintf("Task Rate:         %.1f tasks/s", taskRate),
		)
	}

	return statStyle.Render(strings.Join(stats, "\n"))
}

func (d *Dashboard) renderHTTPStats(width, height int) string {
	statStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF6B6B")).
		Padding(1, 2).
		Width(width - 2).  // Adjust for border
		Height(height - 2) // Adjust for border

	stats := []string{
		"🌐 HTTP Statistics",
		"",
		fmt.Sprintf("Total Requests:    %d", d.metrics.HTTPRequests),
		fmt.Sprintf("Successful:        %d", d.metrics.SuccessCount),
		fmt.Sprintf("Failed:            %d", d.metrics.HTTPRequests-d.metrics.SuccessCount),
	}

	// Calculate HTTP rate and success rate
	elapsed := time.Since(d.startTime).Seconds()
	if elapsed > 0 {
		httpRate := float64(d.metrics.HTTPRequests) / elapsed
		stats = append(stats,
			"",
			fmt.Sprintf("Request Rate:      %.1f req/s", httpRate),
		)
	}

	if d.metrics.HTTPRequests > 0 {
		successRate := float64(d.metrics.SuccessCount) / float64(d.metrics.HTTPRequests) * 100
		stats = append(stats,
			fmt.Sprintf("Success Rate:      %.1f%%", successRate),
		)
	}

	return statStyle.Render(strings.Join(stats, "\n"))
}

func (d *Dashboard) renderDNSStats(width, height int) string {
	statStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Padding(1, 2).
		Width(width - 2).  // Adjust for border
		Height(height - 2) // Adjust for border

	stats := []string{
		"🔍 DNS Statistics",
		"",
		fmt.Sprintf("Total Queries:     %d", d.metrics.DNSRequests),
	}

	// Calculate DNS rate
	elapsed := time.Since(d.startTime).Seconds()
	if elapsed > 0 {
		dnsRate := float64(d.metrics.DNSRequests) / elapsed
		stats = append(stats,
			"",
			fmt.Sprintf("Query Rate:        %.1f req/s", dnsRate),
		)
	}

	return statStyle.Render(strings.Join(stats, "\n"))
}

func (d *Dashboard) renderRecentDiscoveries(width, height int) string {
	discoveryStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#04B575")).
		Padding(1, 2).
		Width(width - 2).  // Adjust for border
		Height(height - 2) // Adjust for border

	recentCount := len(d.recentSubdomains)
	recentCopy := make([]string, recentCount)
	copy(recentCopy, d.recentSubdomains)

	lines := []string{
		fmt.Sprintf("🔍 Recent Discoveries (Total: %d)", recentCount),
		"",
	}

	if recentCount == 0 {
		lines = append(lines, "No subdomains discovered yet...")
	} else {
		// Calculate how many lines we can show based on height
		// Height - 2 (border) - 2 (padding) - 2 (title + empty line)
		maxLines := height - 6
		if maxLines < 0 {
			maxLines = 0
		}

		maxShow := maxLines
		start := 0
		if recentCount > maxShow {
			start = recentCount - maxShow
		}

		for i := start; i < recentCount; i++ {
			lines = append(lines, fmt.Sprintf("  • %s", recentCopy[i]))
		}

		// If we truncated, we might want to show that, but simple logic is fine for now
		// Or we can just show the last N that fit
	}

	return discoveryStyle.Render(strings.Join(lines, "\n"))
}

func (d *Dashboard) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(1, 0)

	return footerStyle.Render("Press 'q' or 'Ctrl+C' to quit")
}

// AddSubdomain adds a newly discovered subdomain to the recent list
func (d *Dashboard) AddSubdomain(subdomain string) {
	d.mu.Lock()
	d.recentSubdomains = append(d.recentSubdomains, subdomain)

	// Keep only the last 50 for memory efficiency
	if len(d.recentSubdomains) > 50 {
		d.recentSubdomains = d.recentSubdomains[len(d.recentSubdomains)-50:]
	}
	d.mu.Unlock()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Run starts the dashboard
func (d *Dashboard) Run() error {
	p := tea.NewProgram(d, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
