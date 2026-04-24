package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PhaseUpdateMsg is sent when a pipeline phase changes state.
type PhaseUpdateMsg struct {
	Index  int
	Status string
}

var (
	phaseStyle   = lipgloss.NewStyle().Bold(true)
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	activeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	headerStyle  = lipgloss.NewStyle().Bold(true).Underline(true)
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

type phase struct {
	name    string
	status  string // "done", "active", "pending"
	elapsed time.Duration
}

type RunModel struct {
	Phases      []phase
	Current     int
	Spinner     spinner.Model
	Progress    int
	Total       int
	CurrentFile string
	Result      *Result
	DryRun      bool
	Done        bool
	Err         error
	Width       int
}

func NewRunModel(dryRun bool) RunModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return RunModel{
		DryRun: dryRun,
		Phases: []phase{
			{name: "Setup"},
			{name: "Vendor"},
			{name: "Rewrite"},
			{name: "Move"},
			{name: "Cleanup"},
		},
		Spinner: s,
	}
}

func (m RunModel) Init() tea.Cmd {
	return m.Spinner.Tick
}

func (m RunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
	case PhaseUpdateMsg:
		m.Phases[msg.Index].status = msg.Status
		m.Current = msg.Index
		return m, m.Spinner.Tick
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m RunModel) View() string {
	var b strings.Builder
	title := "Privatizing..."
	if m.DryRun {
		title = "Privatizing (dry-run)..."
	}
	if m.Done {
		title = "Privatizing... done!"
	}
	b.WriteString(phaseStyle.Render(title))
	b.WriteString("\n\n")

	for _, p := range m.Phases {
		switch p.status {
		case "done":
			fmt.Fprintf(&b, "  %s %-12s completed in %s\n",
				doneStyle.Render("✓"), p.name, p.elapsed.Round(time.Millisecond))
		case "error":
			fmt.Fprintf(&b, "  %s %-12s failed\n",
				errorStyle.Render("✗"), p.name)
		case "active":
			fmt.Fprintf(&b, "  %s %-12s %s\n",
				activeStyle.Render(m.Spinner.View()), p.name,
				m.progressDetail())
		default:
			fmt.Fprintf(&b, "  %s %s\n", pendingStyle.Render("○"), p.name)
		}
	}

	if m.CurrentFile != "" {
		fmt.Fprintf(&b, "\n    %s\n", pathStyle.Render(m.CurrentFile))
	}

	if m.Done && m.Result != nil {
		b.WriteString("\n" + renderSummary(m.Result))
	}

	if m.Err != nil {
		fmt.Fprintf(&b, "\n  Error: %s\n", m.Err)
	}

	if m.Done {
		b.WriteString("\n")
	}
	return b.String()
}

func (m RunModel) progressDetail() string {
	if m.Total > 0 {
		return fmt.Sprintf("processing %d/%d files...", m.Progress, m.Total)
	}
	return ""
}

func renderSummary(r *Result) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("-- Summary --"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "  %d imports rewritten\n", len(r.Rewrites))
	fmt.Fprintf(&b, "  %d directories copied\n", len(r.Copied))
	fmt.Fprintf(&b, "  %d files modified\n", len(r.Modified))
	return b.String()
}

func RenderDryRunResult(r *Result, modulePath string) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("-- Import Rewrites --"))
	b.WriteString("\n\n")
	for _, c := range r.Rewrites {
		fmt.Fprintf(&b, "  %s -> %s\n", c.OldPath, c.NewPath)
	}
	if len(r.Copied) > 0 {
		b.WriteString("\n" + headerStyle.Render("-- Directories to Copy --"))
		b.WriteString("\n\n")
		for _, d := range r.Copied {
			fmt.Fprintf(&b, "  %s\n", d)
		}
	}
	return b.String()
}
