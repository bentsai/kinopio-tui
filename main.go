package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	list    list.Model
	spinner spinner.Model
	err     error
	loading bool // Track loading state
}

type Card struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Space struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
}

func (m *model) Init() tea.Cmd {
	m.loading = true // Set loading state to true initially
	return tea.Batch(fetchSpaces(), m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spacesMsg:
		items := make([]list.Item, len(msg.spaces))
		for i, space := range msg.spaces {
			items[i] = listItem{space}
		}
		m.list.SetItems(items)
		m.loading = false // Data has been loaded
	case error:
		m.err = msg
		m.loading = false // Stop loading on error
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4) // Adjust for any header/footer
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			// Display the ID of the selected space
			if item, ok := m.list.SelectedItem().(listItem); ok {
				fmt.Printf("Selected Space ID: %s\n", item.Space.ID)
			}
		}
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.loading {
		return fmt.Sprintf("\n\n   %s Loading spaces...\n\nPress q to quit.", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("Error fetching spaces:\n%v\n\nPress q to quit.", m.err)
	}
	return m.list.View()
}

type listItem struct {
	Space Space
}

func (i listItem) FilterValue() string { return i.Space.Name }
func (i listItem) Title() string       { return i.Space.Name }
func (i listItem) Description() string {
	return fmt.Sprintf("https://kinopio.club/%s", i.Space.Url)
}

type spacesMsg struct {
	spaces []Space
}

func fetchSpaces() tea.Cmd {
	return func() tea.Msg {
		apiKey := getAPIKey()
		client := &http.Client{}
		req, err := http.NewRequest("GET", "https://api.kinopio.club/user/spaces", nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Authorization", apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error performing request: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			var errorDetails map[string]interface{}
			jsonErr := json.Unmarshal(body, &errorDetails)
			if jsonErr != nil {
				return fmt.Errorf("failed to fetch spaces: %s\nResponse body: %s", resp.Status, string(body))
			}
			errorDetailsStr, _ := json.MarshalIndent(errorDetails, "", "  ")
			return fmt.Errorf("failed to fetch spaces: %s\nError details:\n%s", resp.Status, string(errorDetailsStr))
		}

		var spaces []Space
		if err := json.Unmarshal(body, &spaces); err != nil {
			return fmt.Errorf("error unmarshaling response: %v", err)
		}

		return spacesMsg{spaces: spaces}
	}
}

func getAPIKey() string {
	apiKey := os.Getenv("KINOPIO_API_KEY")
	if apiKey == "" {
		fmt.Println("API key is not set")
		os.Exit(1)
	}
	return apiKey
}

func main() {
	itemDelegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, itemDelegate, 0, 0) // Start with zero size, we'll adjust it later
	l.Title = "Spaces"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	m := &model{
		list:    l,
		spinner: sp,
	}
	p := tea.NewProgram(m) // Removed tea.WithAltScreen() for debugging
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running program:", err)
		os.Exit(1)
	}
}
