package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerModel struct {
	spinner   spinner.Model
	startTime time.Time
	messages  []string
	loadIndex int
}

func newSpinnerModel(messages []string) *spinnerModel {
	s := spinner.New()
	s.Style = spinnerStyle
	s.Spinner = spinner.Pulse
	return &spinnerModel{
		spinner:   s,
		startTime: time.Now(),
		messages:  messages,
	}
}

func (s *spinnerModel) Init() tea.Cmd {
	return tea.Batch(s.spinner.Tick, s.loadTicker)
}

func (s *spinnerModel) loadTicker() tea.Msg {
	time.Sleep(2 * time.Second)
	return tickMsg{}
}

func (s *spinnerModel) Update(msg tea.Msg) (*spinnerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		s.loadIndex++
		if s.loadIndex >= len(s.messages) {
			s.loadIndex = len(s.messages) - 1
		}
		return s, s.loadTicker
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *spinnerModel) View() string {
	elapsed := time.Since(s.startTime).Round(time.Second)
	msg := s.messages[s.loadIndex]
	return fmt.Sprintf("\n  %s %s\n\n  elapsed: %s\n", s.spinner.View(), msg, elapsed)
}

func (s *spinnerModel) Tick() tea.Cmd {
	return s.spinner.Tick
}
