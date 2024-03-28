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
	"github.com/charmbracelet/log"
	flag "github.com/spf13/pflag"
)

const VERSION = "0.1.1"

type Model struct {
	x265progress     progress.Model
	av1progress      progress.Model
	form             *huh.Form // huh.Form is just a tea.Model
	filepicker       filepicker.Model
	selectedFilePath string
	selectedFileName string
	done             []string
}

var Program *tea.Program
var Cmd *exec.Cmd
var Crf = "28"
var StripAudio = false
var Preview = false
var skipX265 bool
var skipAV1 bool
var converting = false

var appStyle = lipgloss.NewStyle().Margin(1, 2, 0, 2)
var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
var headingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))

func (m Model) Init() tea.Cmd {
	if m.selectedFileName == "" {
		return tea.Sequence(
			m.filepicker.Init(),
			m.form.Init(),
		)
	}
	return m.form.Init()
}

func startX265Conversion(m Model) tea.Cmd {
	go Convert(m.selectedFilePath, fmt.Sprintf("./optimized/%s.mp4", m.selectedFileName), "libx265")
	return nil
}

func startAV1Conversion(m Model) tea.Cmd {
	go Convert(m.selectedFilePath, fmt.Sprintf("./optimized/%s.webm", m.selectedFileName), "libsvtav1")
	return nil
}

func startConversion(m Model) tea.Cmd {
	if !skipX265 {
		return startX265Conversion(m)
	}
	// skipping x265
	if !skipAV1 {
		return startAV1Conversion(m)
	}

	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// quit if the user presses q or ctrl+c
	if msg, ok := msg.(tea.KeyMsg); ok {
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

	if m.selectedFilePath == "" {
		var cmd tea.Cmd
		m.filepicker, cmd = m.filepicker.Update(msg)
		// Did the user select a file?
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			// Get the path of the selected file.
			m.selectedFilePath = path
			// get the file name without extension
			m.selectedFileName = getFileNameFromPath(m.selectedFilePath)
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	// Update the progress bar percentages
	case progressMsg:
		if msg.conversion == "libx265" {
			return m, m.x265progress.SetPercent(float64(msg.percent))
		}
		if msg.conversion == "libsvtav1" {
			return m, m.av1progress.SetPercent(float64(msg.percent))
		}

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		if CurentConversion == "libx265" {
			progressModel, cmd := m.x265progress.Update(msg)
			m.x265progress = progressModel.(progress.Model)
			return m, cmd
		}
		if CurentConversion == "libsvtav1" {
			progressModel, cmd := m.av1progress.Update(msg)
			m.av1progress = progressModel.(progress.Model)
			return m, cmd
		}

	// if a conversion is done, quit or start the next
	case conversionDone:
		if msg == "libx265" {
			m.x265progress.SetPercent(1.0)
			if !skipAV1 {
				return m, startAV1Conversion(m)
			}
			return m, tea.Quit
		}
		if msg == "libsvtav1" {
			m.av1progress.SetPercent(1.0)
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// Calculate the maximum width of the progress bars
		bars := []progress.Model{m.x265progress, m.av1progress}
		maxWidth := msg.Width - 2*2 - 4
		for _, bar := range bars {
			if bar.Width > maxWidth {
				maxWidth = bar.Width
			}
		}
		return m, nil
	}

	// if the form is completed, start the conversion
	if m.form.State == huh.StateCompleted && !converting {
		converting = true
		return m, startConversion(m)
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

	result := "Converting to x265"
	result += "\n" + m.x265progress.View()
	result += "\n\n" + "Converting to AV1"
	result += "\n" + m.av1progress.View()
	result += "\n\n" + helpStyle.Render("Press q to quit")
	return appStyle.Render(result)
}

func main() {
	allowedTypes := []string{".mp4", ".mkv", ".mov", ".avi", ".wmv", ".webm"}
	selectedFilePath := ""
	selectedFileName := ""

	// handle flags
	versionFlag := flag.BoolP("version", "v", false, "Print version and exit")
	updateFlag := flag.BoolP("update", "u", false, "Update to the latest version")
	flag.BoolVar(&skipX265, "skip-x265", false, "Skip x265 conversion")
	flag.BoolVar(&skipAV1, "skip-av1", false, "Skip AV1 conversion")

	// Override default Usage function to suppress the "pflag: help requested" message
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: webvids [OPTIONS] [FILE]\n\nOptions:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	flag.Parse()

	if *versionFlag {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	if *updateFlag {
		Update()
		os.Exit(0)
	}
	if skipX265 && skipAV1 {
		log.Error("Cannot skip both x265 and AV1")
		os.Exit(1)
	}

	// verify that ffmpeg is installed
	_, err := exec.LookPath("ffmpeg")
	CheckError(err)

	// if user passed in file path
	tail := flag.Args()
	if len(tail) > 0 {
		_, err := os.Stat(tail[0])
		CheckError(err)

		// check if the file is an allowed type
		for _, allowedType := range allowedTypes {
			if strings.HasSuffix(tail[0], allowedType) {
				selectedFilePath = tail[0]
				selectedFileName = getFileNameFromPath(selectedFilePath)
				break
			}
		}

		// if file is not an allowed type, exit
		if selectedFilePath == "" {
			log.Errorf("File not allowed. Allowed types: %s", strings.Join(allowedTypes, ", "))
			os.Exit(1)
		}
	}

	// create file picker if no file is passed in
	var fp filepicker.Model
	if selectedFilePath == "" {
		fp = filepicker.New()
		fp.AllowedTypes = allowedTypes
	}

	// initialize model
	m := Model{
		x265progress:     progress.New(progress.WithDefaultGradient()),
		av1progress:      progress.New(progress.WithDefaultGradient()),
		filepicker:       fp,
		selectedFilePath: selectedFilePath,
		selectedFileName: selectedFileName,
		done:             []string{},
		form: huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Constant rate factor").
					Description("Lowering value will increase quality and file size.").
					Placeholder(Crf).
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

func getFileNameFromPath(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func CheckError(err error) {
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
