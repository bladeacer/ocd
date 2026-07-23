package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

func TestNewSpinnerModel(t *testing.T) {
	s := newSpinnerModel([]string{"Loading..."})
	if s == nil {
		t.Fatal("expected non-nil spinner")
	}
	if len(s.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(s.messages))
	}
}

func TestSpinnerModelView(t *testing.T) {
	s := newSpinnerModel([]string{"Loading test..."})
	v := s.View()
	if !strings.Contains(v, "Loading test...") {
		t.Errorf("expected 'Loading test...' in view, got %q", v)
	}
	if !strings.Contains(v, "elapsed:") {
		t.Errorf("expected 'elapsed:' in view, got %q", v)
	}
}

func TestSpinnerModelUpdateTick(t *testing.T) {
	s := newSpinnerModel([]string{"One", "Two", "Three"})
	s.startTime = time.Now().Add(-5 * time.Second)
	s.loadIndex = 0

	updated, cmd := s.Update(tickMsg{})
	if cmd == nil {
		t.Error("expected non-nil command after tickMsg")
	}
	if updated.loadIndex != 1 {
		t.Errorf("expected loadIndex=1, got %d", updated.loadIndex)
	}
	_ = updated.View()
}

func TestSpinnerModelUpdateSpinnerTick(t *testing.T) {
	s := newSpinnerModel([]string{"Loading..."})
	updated, cmd := s.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Error("expected non-nil command after spinner.TickMsg")
	}
	_ = updated
}

func TestSpinnerModelTickBeyondMax(t *testing.T) {
	s := newSpinnerModel([]string{"One", "Two"})
	s.loadIndex = 1

	updated, _ := s.Update(tickMsg{})
	if updated.loadIndex != 1 {
		t.Errorf("expected loadIndex=1 (capped), got %d", updated.loadIndex)
	}
}

func TestSpinnerModelInit(t *testing.T) {
	s := newSpinnerModel([]string{"Loading..."})
	cmd := s.Init()
	if cmd == nil {
		t.Error("expected non-nil command from Init")
	}
}

func TestSpinnerModelTick(t *testing.T) {
	s := newSpinnerModel([]string{"Loading..."})
	cmd := s.Tick()
	if cmd == nil {
		t.Error("expected non-nil command from Tick")
	}
}

func TestSpinnerModelLoadTickerReturnsTickMsg(t *testing.T) {
	s := newSpinnerModel([]string{"Loading..."})
	msg := s.loadTicker()
	if _, ok := msg.(tickMsg); !ok {
		t.Errorf("expected tickMsg, got %T", msg)
	}
}
