package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.
	NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var spinnerStyle = lipgloss.
	NewStyle().
	Bold(true).
	Background(lipgloss.Color("57")).
	Foreground(lipgloss.Color("15"))

type Repository struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	StargazersCount int    `json:"stargazers_count"`
}

type Repositories struct {
	data []Repository
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string { return e.err.Error() }

type model struct {
	repositories Repositories
	textInput    textinput.Model
	username     string
	table        table.Model
	err          error
	spinner      spinner.Model
	loading      bool
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func initialModel() model {
	// text input
	ti := textinput.New()
	ti.Placeholder = "Your GitHub username..."
	ti.Width = 100
	ti.Focus()

	// table
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Description", Width: 40},
		{Title: "Stars", Width: 30},
	}
	rows := []table.Row{}
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithWidth(100),
	)
	// table styles
	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(ts)

	// spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	// s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	return model{
		textInput:    ti,
		repositories: Repositories{},
		err:          nil,
		table:        t,
		spinner:      s,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd      tea.Cmd
		tableCmd   tea.Cmd
		spinnerCmd tea.Cmd
	)

	switch msg := msg.(type) {

	case Repositories:
		m.repositories = msg
		rows := []table.Row{}

		for _, repo := range m.repositories.data {
			description := repo.Description
			if description == "" {
				description = "-no description-"
			}
			row := table.Row{
				repo.Name, description, strconv.Itoa(repo.StargazersCount),
			}
			rows = append(rows, row)
		}

		m.table.SetRows(rows)
		m.table.Focus()
		m.loading = false

	// keys
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.table.Focused() {
				m.table.Blur()
				m.textInput.Focus()
			} else {
				m.table.Focus()
				m.textInput.Blur()
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			m.username = m.textInput.Value()
			m.textInput.Blur()
			m.spinner.Tick()
			m.loading = true
			return m, tea.Batch(fetchRepositories(m.username), m.spinner.Tick)
		}

	// error
	case errMsg:
		m.err = msg

	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.table, tableCmd = m.table.Update(msg)
	m.spinner, spinnerCmd = m.spinner.Update(msg)

	return m, tea.Batch(tiCmd, tableCmd, spinnerCmd)
}

func (m model) View() string {
	var spinnerView string

	if m.loading {
		spinnerView = spinnerStyle.Render(m.spinner.View() + " Fetching repositories...")
	} else {
		spinnerView = ""
	}

	return fmt.Sprintf(
		"Let's fetch your GitHub repos!\n\n%s\n%s\n%s",
		m.textInput.View(),
		spinnerView,
		baseStyle.Render(m.table.View()),
	)
}

func fetchRepositories(username string) tea.Cmd {
	return func() tea.Msg {
		s := &http.Client{Timeout: time.Second * 10}
		resp, err := s.Get("https://api.github.com/users/" + username + "/repos")
		if err != nil {
			return errMsg{err}
		}
		defer resp.Body.Close()

		repositories := []Repository{}
		if err = json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
			return errMsg{err}
		}

		return Repositories{data: repositories}
	}
}
