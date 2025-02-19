package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	spaces []Space
	cursor int // Which space the cursor is pointing at
	err    error
}

type Space struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (m model) Init() tea.Cmd {
	return fetchSpaces()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spacesMsg:
		m.spaces = msg.spaces
		return m, nil
	case error:
		m.err = msg
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j":
			if m.cursor < len(m.spaces)-1 {
				m.cursor++
			}
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			// Display the ID of the selected space
			if m.cursor >= 0 && m.cursor < len(m.spaces) {
				fmt.Printf("Selected Space ID: %s\n", m.spaces[m.cursor].ID)
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error fetching spaces:\n%v\n\nPress q to quit.", m.err)
	}
	if len(m.spaces) == 0 {
		return "Loading spaces...\nPress q to quit."
	}

	var b strings.Builder
	b.WriteString("Spaces:\n\n")
	for i, space := range m.spaces {
		cursor := " " // No cursor
		if m.cursor == i {
			cursor = ">" // Cursor
		}
		b.WriteString(fmt.Sprintf("%s %s\n", cursor, space.Name))
	}
	b.WriteString("\nUse j/k to navigate and Enter to select. Press q to quit.")
	return b.String()
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
			return fmt.Errorf("Error creating request: %v", err)
		}

		req.Header.Set("Authorization", apiKey)
		req.Header.Set("Content-Type", "application/json")

		logRequest(req)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("Error performing request: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Error reading response body: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			var errorDetails map[string]interface{}
			jsonErr := json.Unmarshal(body, &errorDetails)
			if jsonErr != nil {
				return fmt.Errorf("Failed to fetch spaces: %s\nResponse body: %s", resp.Status, string(body))
			}
			errorDetailsStr, _ := json.MarshalIndent(errorDetails, "", "  ")
			return fmt.Errorf("Failed to fetch spaces: %s\nError details:\n%s", resp.Status, string(errorDetailsStr))
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return fmt.Errorf("Unexpected content type: %s", contentType)
		}

		var spaces []Space
		if err := json.Unmarshal(body, &spaces); err != nil {
			return fmt.Errorf("Error unmarshaling response: %v", err)
		}

		return spacesMsg{spaces: spaces}
	}
}

func logRequest(req *http.Request) {
	fmt.Printf("Request Method: %s\n", req.Method)
	fmt.Printf("Request URL: %s\n", req.URL)
	fmt.Println("Request Headers:")
	for k, v := range req.Header {
		fmt.Printf("  %s: %s\n", k, v)
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
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running program:", err)
		os.Exit(1)
	}
}
