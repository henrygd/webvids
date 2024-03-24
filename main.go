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
	form         *huh.Form // huh.Form is just a tea.Model
}

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

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
		// quit if conversions are done
		if m.x265progress.Percent() >= 1.00 && m.vp9progress.Percent() >= 1.00 {
			return m, tea.Quit
		}
		// fmt.Println(msg.percent)
		// start vp9 conversion if x265 is done
		if m.x265progress.Percent() >= 1.00 && m.vp9progress.Percent() == 0.0 {
			go Convert("test2.mp4", "./optimized/output2.webm", "libvpx-vp9")
			return m, m.vp9progress.SetPercent(0.001)
		}
		// Update the progress bar
		if msg.conversion == "libx265" {
			return m, m.x265progress.SetPercent(float64(msg.percent))
		}
		if msg.conversion == "libvpx-vp9" {
			return m, m.vp9progress.SetPercent(float64(msg.percent))
		}

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		if CurentConversion == "libx265" {
			progressModel, cmd := m.x265progress.Update(msg)
			m.x265progress = progressModel.(progress.Model)
			return m, cmd
		}
		if CurentConversion == "libvpx-vp9" {
			progressModel, cmd := m.vp9progress.Update(msg)
			m.vp9progress = progressModel.(progress.Model)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		// Calculate the maximum width of the progress bars
		bars := []progress.Model{m.x265progress, m.vp9progress}
		maxWidth := msg.Width - padding*2 - 4
		for _, bar := range bars {
			if bar.Width > maxWidth {
				maxWidth = bar.Width
			}
		}
		return m, nil
	}

	// if the form is completed, start the conversion
	if m.form.State == huh.StateCompleted {
		if m.x265progress.Percent() == 0.0 {
			go Convert("test2.mp4", "./optimized/output.mp4", "libx265")
			return m, m.x265progress.SetPercent(0.001)
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.form.State != huh.StateCompleted {
		return m.form.View()
	}
	// sb := strings.Builder{}
	pad := strings.Repeat(" ", padding)
	result := ""
	if m.x265progress.Percent() > 0.0 {
		result += pad + "Converting to x265"
		result += "\n" + pad + m.x265progress.View()
		// sb.WriteString(pad + helpStyle("Converting to x265\n"))
		// sb.WriteString("\n" +
		// 	pad + m.x265progress.View() + "\n\n")
		result += "\n\n" + pad + "Converting to VP9"
		result += "\n" + pad + m.vp9progress.View()
	}
	// if m.vp9progress.Percent() > 0.0 {
	// 	result += "\n\n" + pad + "Converting to VP9"
	// 	result += "\n" + pad + m.vp9progress.View()
	// }
	if result != "" {
		result += "\n\n" + pad + helpStyle("Press q to quit")
		return result
	}
	// if sb.Len() > 0 {
	// 	sb.WriteString(pad + helpStyle("Press q to quit"))
	// 	return sb.String()
	// }
	burger := m.form.GetString("burger")
	return fmt.Sprintf("You chose %s", burger)
}

func main() {
	// make optimized directory if not exists
	if _, err := os.Stat("./optimized"); os.IsNotExist(err) {
		os.Mkdir("./optimized", 0755)
	}

	Program = tea.NewProgram(initialModel())
	if _, err := Program.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
