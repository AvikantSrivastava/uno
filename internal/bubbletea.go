package internal

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qeesung/image2ascii/convert"
)

// Model represents the state of the UNO game
type Model struct{}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update updates the model based on the received message

// View renders the current state of the game
func (m Model) View() string {
	return ""
}
func UNoOLogoPrint() {
	// Relative path to your file
	absolutePath := "D:\\WEb development\\uno\\UNO_Logo.png"
	convertOptions := convert.DefaultOptions
	convertOptions.FixedWidth = 58
	convertOptions.FixedHeight = 16

	converter := convert.NewImageConverter()
	fmt.Print(converter.ImageFile2ASCIIString(absolutePath, &convertOptions), "\n")

}
