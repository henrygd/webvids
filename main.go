package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const VERSION = "0.0.2"

type Model struct {
	x265progress     progress.Model
	vp9progress      progress.Model
	form             *huh.Form // huh.Form is just a tea.Model
	filepicker       filepicker.Model
	selectedFilePath string
	selectedFileName string
}

const (
	padding  = 2
	maxWidth = 80
)

var Program *tea.Program
var Cmd *exec.Cmd
var Crf = "30"
var StripAudio = false
var Preview = false

var appStyle = lipgloss.NewStyle().Margin(1, 2, 0, 2)
var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
var headingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.filepicker.Init(),
		m.form.Init(),
	)
}

var converting = false

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// if the form is completed, start the conversion
	if m.form.State == huh.StateCompleted && !converting {
		converting = true
		go Convert(m.selectedFilePath, fmt.Sprintf("./optimized/%s.mp4", m.selectedFileName), "libx265")
		return m, m.x265progress.SetPercent(0.001)
	}

	// quit if the user presses q or ctrl+c
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Sequence(
				func() tea.Msg {
					if Cmd != nil && Cmd.Process != nil {
						Cmd.Process.Kill()
					}
					return nil
				},
				tea.Quit,
			)
		}
	}

	// Update the form
	if m.selectedFilePath != "" && m.form.State != huh.StateCompleted {
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

	if m.selectedFilePath == "" {
		m.filepicker, cmd = m.filepicker.Update(msg)
		// Did the user select a file?
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			// Get the path of the selected file.
			m.selectedFilePath = path
			// get the file name without extension
			m.selectedFileName = strings.TrimSuffix(filepath.Base(m.selectedFilePath), filepath.Ext(m.selectedFilePath))
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
			go Convert(m.selectedFilePath, fmt.Sprintf("./optimized/%s.webm", m.selectedFileName), "libvpx-vp9")
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
	if m.selectedFilePath == "" {
		var s strings.Builder
		s.WriteString(headingStyle.Render("Choose file:"))
		s.WriteString("\n\n" + m.filepicker.View())
		return appStyle.Render(s.String())
	}

	// file has been selected - show the form if not completed
	if m.form.State != huh.StateCompleted {
		return appStyle.Render(m.form.View())
	}

	result := ""
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
	// handle update
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--update" {
			Update()
			os.Exit(0)
		}
	}

	// initialize model
	fp := filepicker.New()
	fp.AllowedTypes = []string{".mp4", ".mkv", ".mov", ".avi", ".wmv", ".webm"}

	m := Model{
		x265progress:     progress.New(progress.WithDefaultGradient()),
		vp9progress:      progress.New(progress.WithDefaultGradient()),
		filepicker:       fp,
		selectedFilePath: "",
		// weird thing where you can't press enter on the text immediately
		form: huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Constant rate factor").
					Description("Lowering value will increase quality and file size.").
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
					Description("Choose yes if using video as muted background").
					Affirmative("Yes").
					Negative("No").
					Value(&StripAudio),

				huh.NewConfirm().
					Title("Preview?").
					Description("Converts only first three seconds of video").
					Affirmative("Yes").
					Negative("No").
					Value(&Preview),
			),
		),
	}

	Program = tea.NewProgram(m)
	if _, err := Program.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
