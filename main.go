package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var Program *tea.Program

type Model struct {
	x265progress progress.Model
	vp9progress  progress.Model
	progress     progress.Model
	form         *huh.Form // huh.Form is just a tea.Model
}

var form = huh.NewForm(
	huh.NewGroup(
		huh.NewSelect[string]().
			Title("Choose your burger").
			Options(
				huh.NewOption("Charmburger Classic", "classic"),
				huh.NewOption("Chickwich", "chickwich"),
				huh.NewOption("Fishburger", "fishburger"),
				huh.NewOption("Charmpossible™ Burger", "charmpossible"),
			),
	),
)

func initialModel() Model {
	return Model{
		x265progress: progress.New(progress.WithDefaultGradient()),
		vp9progress:  progress.New(progress.WithDefaultGradient()),
		progress:     progress.New(progress.WithDefaultGradient()),
		form:         form,
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update the form
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	switch msg := msg.(type) {
	// quit if the user presses q or ctrl+c
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case progressMsg:
		if msg > 1.1 {
			return m, tea.Quit
		}
		return m, m.progress.SetPercent(float64(msg))

		// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	// quit if the form is completed
	if m.form.State == huh.StateCompleted {
		if m.progress.Percent() == 0.0 {
			go Convert()
			return m, m.progress.SetPercent(0.001)
		}
	}

	return m, cmd
}

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

func (m Model) View() string {
	if m.form.State != huh.StateCompleted {
		return m.form.View()
	}
	if m.progress.Percent() > 0.0 {
		// return fmt.Sprintf("Progress: %.0f%%", m.vp9progress.Percent()*100)
		pad := strings.Repeat(" ", padding)
		return "\n" +
			pad + m.progress.View() + "\n\n" +
			pad + helpStyle("Press any key to quit")
	}
	burger := m.form.GetString("burger")
	return fmt.Sprintf("You chose %s", burger)
}

func main() {
	Program = tea.NewProgram(initialModel())
	if _, err := Program.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
