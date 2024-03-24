package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
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
	filepicker   filepicker.Model
	selectedFile string
}

const (
	padding  = 2
	maxWidth = 80
)

var Crf = "30"
var StripAudio = true

var appStyle = lipgloss.NewStyle().Margin(1, 2, 0, 2)
var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.filepicker.Init(),
		m.form.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// if m.form.State == huh.StateCompleted {
	// 	fmt.Println("form completed!!!!!!")
	// 	// return m, tea.Quit
	// }

	// if the form is completed, start the conversion
	if m.form.State == huh.StateCompleted && m.x265progress.Percent() == 0.0 {
		go Convert(m.selectedFile, "./optimized/output.mp4", "libx265")
		return m, m.x265progress.SetPercent(0.001)
	}

	// quit if the user presses q or ctrl+c
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
			// return m, tea.Batch(
			// 	tea.ClearScreen,
			// 	tea.Println("Quitting..."),
			// 	tea.Quit,
			// )
		}
	}

	// Update the form
	if m.selectedFile != "" && m.form.State != huh.StateCompleted {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		return m, tea.Sequence(
			cmd,
			func() tea.Msg {
				if m.form.State == huh.StateCompleted {
					m.Update(nil)
				}
				return nil
			})
	}

	var cmd tea.Cmd

	if m.selectedFile == "" {
		m.filepicker, cmd = m.filepicker.Update(msg)
		// Did the user select a file?
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			// Get the path of the selected file.
			m.selectedFile = path
			// return m, cmd
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case progressMsg:
		// quit if conversions are done
		if m.x265progress.Percent() >= 1.0 && m.vp9progress.Percent() >= 1.0 {
			return m, tea.Quit
		}
		// start vp9 conversion if x265 is done
		if m.x265progress.Percent() >= 1.00 && m.vp9progress.Percent() == 0.0 {
			go Convert(m.selectedFile, "./optimized/output2.webm", "libvpx-vp9")
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

	return m, nil
}

func (m Model) View() string {
	// display file picker if no file is selected
	if m.selectedFile == "" {
		var s strings.Builder
		s.WriteString("Choose file:")
		s.WriteString("\n\n" + m.filepicker.View())
		return appStyle.Render(s.String())
	}

	result := ""
	// file has been selected - show the form if not completed
	if m.form.State != huh.StateCompleted {
		result += m.form.View()
	}

	if m.x265progress.Percent() > 0.0 {
		result += "Converting to x265"
		result += "\n" + m.x265progress.View()
		result += "\n\n" + "Converting to VP9"
		result += "\n" + m.vp9progress.View()
	}
	if result != "" {
		result += "\n\n" + helpStyle.Render("Press q to quit")
		return appStyle.Render(result)
	}
	return ""
}

func main() {
	// make optimized directory if not exists
	if _, err := os.Stat("./optimized"); os.IsNotExist(err) {
		os.Mkdir("./optimized", 0755)
	}

	// initialize model
	fp := filepicker.New()
	fp.AllowedTypes = []string{".mp4", ".mkv", ".mov", ".avi", ".wmv", ".webm"}

	m := Model{
		x265progress: progress.New(progress.WithDefaultGradient()),
		vp9progress:  progress.New(progress.WithDefaultGradient()),
		filepicker:   fp,
		selectedFile: "",
		// weird thing where you can't press enter on the text immediately
		form: huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Constant rate factor").
					Description("Higher value means higher quality and file size.").
					Placeholder("30").
					Value(&Crf).
					Validate(func(str string) error {
						// Convert string to int
						msg := "Must be a number between 10 and 50"
						num, err := strconv.Atoi(str)
						if err != nil || num < 10 || num > 50 {
							return errors.New(msg)
						}
						return nil
					}),

				huh.NewConfirm().
					Title("Strip audio?").
					Affirmative("Yes").
					Negative("No").
					Value(&StripAudio),
			),
		),
	}

	Program = tea.NewProgram(m)
	if _, err := Program.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
