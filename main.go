package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	spaces string
	err    error
}

func (m model) Init() tea.Cmd {
	// Call fetchSpaces when the program starts
	return tea.Cmd(fetchSpaces)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spacesMsg:
		m.spaces = string(msg)
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
	if m.spaces == "" {
		return "Loading spaces...\nPress q to quit."
	}
	return fmt.Sprintf("Spaces:\n%s\nPress q to quit.", m.spaces)
}

type spacesMsg string

func fetchSpaces() tea.Msg {
	return func() tea.Msg {
		apiKey := getAPIKey()
		client := &http.Client{}
		req, err := http.NewRequest("GET", "https://api.kinopio.club/spaces", nil)
		if err != nil {
			return err
		}

		// Set the API key in the request header
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return spacesMsg(body)
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
