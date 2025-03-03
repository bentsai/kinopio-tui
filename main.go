package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	list          list.Model
	spinner       spinner.Model
	cardTable     table.Model
	err           error
	loading       bool
	currentView   string
	spaces        []Space
	selectedSpace Space
	selectedCard  Card
}

type Card struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	BackgroundColor string `json:"backgroundColor"` // Add backgroundColor field
}

type Box struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Space struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Url   string `json:"url"`
	Cards []Card `json:"cards"`
	Boxes []Box  `json:"boxes"`
}

func (m *model) Init() tea.Cmd {
	m.loading = true
	m.currentView = "list"
	return tea.Batch(fetchSpaces(), m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spacesMsg:
		m.spaces = msg.spaces
		items := make([]list.Item, len(msg.spaces))
		for i, space := range msg.spaces {
			items[i] = listItem{space}
		}
		m.list.SetItems(items)
		m.loading = false
	case spaceDetailsMsg:
		m.selectedSpace = msg.Space
		m.loading = false
		m.currentView = "details"
		m.list.Title = msg.Space.Name
		detailItems := []list.Item{
			detailListItem{"Cards", fmt.Sprintf("%d cards", len(msg.Space.Cards))},
			detailListItem{"Boxes", fmt.Sprintf("%d boxes", len(msg.Space.Boxes))},
		}
		m.list.SetItems(detailItems)
	case error:
		m.err = msg
		m.loading = false
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.currentView == "list" {
				if item, ok := m.list.SelectedItem().(listItem); ok {
					m.loading = true
					return m, fetchSpaceDetails(item.Space.ID)
				}
			} else if m.currentView == "details" {
				if item, ok := m.list.SelectedItem().(detailListItem); ok && item.title == "Cards" {
					m.currentView = "cards"
					m.list.Title = m.selectedSpace.Name + " → Cards"
					cardItems := make([]list.Item, len(m.selectedSpace.Cards))
					for i, card := range m.selectedSpace.Cards {
						cardItems[i] = cardListItem{card}
					}
					m.list.SetItems(cardItems)
				}
			} else if m.currentView == "cards" {
				if item, ok := m.list.SelectedItem().(cardListItem); ok {
					m.selectedCard = item.Card
					m.currentView = "cardDetails"
					return m, m.showCardDetails()
				}
			}
		case "b":
			if m.currentView == "details" {
				m.currentView = "list"
				m.list.Title = "Spaces"
				items := make([]list.Item, len(m.spaces))
				for i, space := range m.spaces {
					items[i] = listItem{space}
				}
				m.list.SetItems(items)
			} else if m.currentView == "cards" {
				m.currentView = "details"
				m.list.Title = m.selectedSpace.Name
				detailItems := []list.Item{
					detailListItem{"Cards", fmt.Sprintf("%d cards", len(m.selectedSpace.Cards))},
					detailListItem{"Boxes", fmt.Sprintf("%d boxes", len(m.selectedSpace.Boxes))},
				}
				m.list.SetItems(detailItems)
			} else if m.currentView == "cardDetails" {
				m.currentView = "cards"
				cardItems := make([]list.Item, len(m.selectedSpace.Cards))
				for i, card := range m.selectedSpace.Cards {
					cardItems[i] = cardListItem{card}
				}
				m.list.SetItems(cardItems)
			}
		}
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.currentView == "cardDetails" {
		var cmd tea.Cmd
		m.cardTable, cmd = m.cardTable.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) showCardDetails() tea.Cmd {
	columns := []table.Column{
		{Title: "Field", Width: 15},
		{Title: "Value", Width: 65},
	}

	// Determine the background color to use
	bgColor := m.selectedCard.BackgroundColor
	if bgColor == "" {
		bgColor = "#e3e3e3" // Default color if none is specified
	}

	// Use lipgloss to apply the background color to the cell
	bgColorStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor)).Render(bgColor)

	rows := []table.Row{
		{"name", m.selectedCard.Name},
		{"x", fmt.Sprintf("%d", m.selectedCard.X)},
		{"y", fmt.Sprintf("%d", m.selectedCard.Y)},
		{"backgroundColor", bgColorStyle},
	}

	m.cardTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(8),
	)

	// Apply styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	m.cardTable.SetStyles(s)

	return nil
}

func (m *model) View() string {
	if m.loading {
		return fmt.Sprintf("\n\n   %s Loading...\n\nPress q to quit.", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("Error:\n%v\n\nPress q to quit.", m.err)
	}

	if m.currentView == "cardDetails" {
		return lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Render(m.cardTable.View()) + "\nPress b to go back."
	}

	helpText := "\nPress Enter to view details, b to go back, q to quit."
	return m.list.View() + helpText
}

type listItem struct {
	Space Space
}

func (i listItem) FilterValue() string { return i.Space.Name }
func (i listItem) Title() string       { return i.Space.Name }
func (i listItem) Description() string {
	return fmt.Sprintf("https://kinopio.club/%s", i.Space.Url)
}

type detailListItem struct {
	title       string
	description string
}

func (i detailListItem) FilterValue() string { return i.title }
func (i detailListItem) Title() string       { return i.title }
func (i detailListItem) Description() string { return i.description }

type cardListItem struct {
	Card Card
}

func (i cardListItem) FilterValue() string { return i.Card.Name }
func (i cardListItem) Title() string       { return i.Card.Name }
func (i cardListItem) Description() string {
	return fmt.Sprintf("(%d, %d)", i.Card.X, i.Card.Y)
}

type spacesMsg struct {
	spaces []Space
}

type spaceDetailsMsg struct {
	Space Space
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

func fetchSpaceDetails(spaceID string) tea.Cmd {
	return func() tea.Msg {
		apiKey := getAPIKey()
		client := &http.Client{}
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.kinopio.club/space/%s", spaceID), nil)
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
				return fmt.Errorf("failed to fetch space details: %s\nResponse body: %s", resp.Status, string(body))
			}
			errorDetailsStr, _ := json.MarshalIndent(errorDetails, "", "  ")
			return fmt.Errorf("failed to fetch space details: %s\nError details:\n%s", resp.Status, string(errorDetailsStr))
		}

		var space Space
		if err := json.Unmarshal(body, &space); err != nil {
			return fmt.Errorf("error unmarshaling space details: %v", err)
		}

		return spaceDetailsMsg{Space: space}
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
	l.SetFilteringEnabled(true) // Enable filtering for fuzzy search

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	m := &model{
		list:    l,
		spinner: sp,
	}
	p := tea.NewProgram(m, tea.WithAltScreen()) // Use alternate screen buffer to clear screen
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running program:", err)
		os.Exit(1)
	}
}
