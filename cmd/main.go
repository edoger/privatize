package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/edoger/privatize/internal"
)

var rootCmd = &cobra.Command{
	Use:   "privatize",
	Short: "Privatize Go third-party dependencies",
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .privatize.yaml config file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(".privatize.yaml"); err == nil {
			fmt.Println(".privatize.yaml already exists")
			return nil
		}
		template := `imports: []

rules: {}

exclude:
  - golang.org/x
`
		if err := os.WriteFile(".privatize.yaml", []byte(template), 0644); err != nil {
			return err
		}
		fmt.Println("Created .privatize.yaml")
		return nil
	},
}

var dryRun bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute privatization based on .privatize.yaml",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := internal.NewPipeline(".")
		if err != nil {
			return err
		}
		if !term.IsTerminal(os.Stdout.Fd()) {
			return runPlain(p, dryRun)
		}
		m := newRunUI(p, dryRun)
		_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
		return err
	},
}

func runPlain(p *internal.Pipeline, dryRun bool) error {
	fmt.Println("Privatizing...")
	result, err := p.Run(dryRun, func(int, string) {})
	if err != nil {
		return err
	}
	fmt.Printf("\n  %d imports rewritten\n", len(result.Rewrites))
	fmt.Printf("  %d directories copied\n", len(result.Copied))
	fmt.Printf("  %d files modified\n", len(result.Modified))
	return nil
}

type runUI struct {
	model    internal.RunModel
	pipeline *internal.Pipeline
	progress chan tea.Msg
}

func newRunUI(p *internal.Pipeline, dryRun bool) runUI {
	return runUI{
		model:    internal.NewRunModel(dryRun),
		pipeline: p,
		progress: make(chan tea.Msg, 10),
	}
}

func (m runUI) Init() tea.Cmd {
	return tea.Batch(m.model.Init(), m.waitForProgress(), m.execute())
}

func (m runUI) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		return <-m.progress
	}
}

func (m runUI) execute() tea.Cmd {
	return func() tea.Msg {
		progress := func(phaseIndex int, status string) {
			msg := internal.PhaseUpdateMsg{Index: phaseIndex, Status: status, At: time.Now()}
			select {
			case m.progress <- msg:
			default:
			}
		}
		result, err := m.pipeline.Run(m.model.DryRun, progress)
		if err != nil {
			return errMsg{err: err}
		}
		return doneMsg{result: result}
	}
}

type doneMsg struct {
	result *internal.Result
}

type errMsg struct {
	err error
}

func (m runUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case internal.PhaseUpdateMsg:
		model, cmd := m.model.Update(msg)
		m.model = model.(internal.RunModel)
		return m, tea.Batch(cmd, m.waitForProgress())
	case doneMsg:
		m.model.Result = msg.result
		m.model.Done = true
		return m, tea.Quit
	case errMsg:
		m.model.Err = msg.err
		m.model.Done = true
		return m, tea.Quit
	}
	model, cmd := m.model.Update(msg)
	m.model = model.(internal.RunModel)
	return m, cmd
}

func (m runUI) View() string {
	return m.model.View()
}

func main() {
	runCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview changes without writing")
	rootCmd.AddCommand(initCmd, runCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
