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
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error fetching spaces: %v\nPress q to quit.", m.err)
	}
	if len(m.spaces) == 0 {
		return "Loading spaces...\nPress q to quit."
	}

	var b strings.Builder
	b.WriteString("Spaces:\n\n")
	for _, space := range m.spaces {
		b.WriteString(fmt.Sprintf("ID: %s\nName: %s\n\n", space.ID, space.Name))
	}
	b.WriteString("Press q to quit.")
	return b.String()
}

type spacesMsg struct {
	spaces []Space
}

func fetchSpaces() tea.Cmd {
	return func() tea.Msg {
		apiKey := getAPIKey()
		client := &http.Client{}
		req, err := http.NewRequest("GET", "https://api.kinopio.club/spaces", nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch spaces: %s", resp.Status)
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return fmt.Errorf("unexpected content type: %s", contentType)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var spaces []Space
		if err := json.Unmarshal(body, &spaces); err != nil {
			return err
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
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running program:", err)
		os.Exit(1)
	}
}
