package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/memory"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// MemoryView displays and manages project and global memory facts (Ctrl+E overlay).
type MemoryView struct {
	manager       *memory.MemoryManager
	width         int
	height        int
	facts         []*memory.MemoryFact
	selectedIndex int
	inputMode     bool
	textInput     textinput.Model
}

// NewMemoryView creates a new MemoryView overlay.
func NewMemoryView(mm *memory.MemoryManager) *MemoryView {
	ti := textinput.New()
	ti.Placeholder = "Enter a fact about this project..."
	ti.Width = 50

	mv := &MemoryView{
		manager:   mm,
		textInput: ti,
	}
	mv.Refresh()
	return mv
}

// SetSize updates layout sizes.
func (mv *MemoryView) SetSize(w, h int) {
	mv.width = w
	mv.height = h
}

// Refresh reloads memory facts from the database.
func (mv *MemoryView) Refresh() {
	facts, err := mv.manager.ListFacts()
	if err == nil {
		mv.facts = facts
	} else {
		mv.facts = nil
	}

	if mv.selectedIndex >= len(mv.facts) {
		mv.selectedIndex = len(mv.facts) - 1
	}
	if mv.selectedIndex < 0 {
		mv.selectedIndex = 0
	}
}

// MoveSelection moves list cursor up or down.
func (mv *MemoryView) MoveSelection(delta int) {
	if len(mv.facts) == 0 || mv.inputMode {
		return
	}
	mv.selectedIndex += delta
	if mv.selectedIndex < 0 {
		mv.selectedIndex = 0
	}
	if mv.selectedIndex >= len(mv.facts) {
		mv.selectedIndex = len(mv.facts) - 1
	}
}

// HandleKey processes keyboard input when the memory overlay is focused.
func (mv *MemoryView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool, bool) {
	// Returns: (cmd, shouldClose, shouldRefresh)
	if mv.inputMode {
		switch msg.Type {
		case tea.KeyEsc:
			mv.inputMode = false
			mv.textInput.Blur()
			mv.textInput.Reset()
			return nil, false, false
		case tea.KeyEnter:
			val := strings.TrimSpace(mv.textInput.Value())
			if val != "" {
				// Determine current project scope
				err := mv.manager.AddFact("project", val, "user", 1.0)
				if err == nil {
					mv.Refresh()
				}
			}
			mv.inputMode = false
			mv.textInput.Blur()
			mv.textInput.Reset()
			return nil, false, true
		default:
			var cmd tea.Cmd
			mv.textInput, cmd = mv.textInput.Update(msg)
			return cmd, false, false
		}
	}

	switch msg.Type {
	case tea.KeyEsc:
		return nil, true, false
	case tea.KeyUp:
		mv.MoveSelection(-1)
	case tea.KeyDown:
		mv.MoveSelection(1)
	}

	switch msg.String() {
	case "a", "A":
		mv.inputMode = true
		mv.textInput.Focus()
		return textinput.Blink, false, false
	case "d", "D":
		if len(mv.facts) > 0 {
			selected := mv.facts[mv.selectedIndex]
			_ = mv.manager.ForgetFact(selected.ID)
			mv.Refresh()
			return nil, false, true
		}
	case "c", "C":
		_ = mv.manager.ClearProject()
		mv.Refresh()
		return nil, false, true
	}

	return nil, false, false
}

// View renders the memory overlay interface.
func (mv *MemoryView) View() string {
	theme := styles.DefaultTheme

	var sb strings.Builder
	sb.WriteString(theme.TitleStyle.Render("🧠 Agent Memory Fact Center"))
	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("Project and global details injected dynamically into the model prompt"))
	sb.WriteString("\n\n")

	// Separate global facts and project facts for display
	var projectFacts []*memory.MemoryFact
	var globalFacts []*memory.MemoryFact
	for _, f := range mv.facts {
		if f.Scope == "global" {
			globalFacts = append(globalFacts, f)
		} else {
			projectFacts = append(projectFacts, f)
		}
	}

	renderFactLine := func(f *memory.MemoryFact, idx int) string {
		marker := "  "
		style := theme.NormalTextStyle
		if idx == mv.selectedIndex {
			marker = "➔ "
			style = theme.HighlightStyle
		}

		sourceBadge := "[user]"
		if f.Source == "extracted" {
			sourceBadge = lipgloss.NewStyle().Foreground(theme.PrimaryColor).Render("[auto]")
		} else {
			sourceBadge = lipgloss.NewStyle().Foreground(theme.SuccessColor).Render("[user]")
		}

		return style.Render(fmt.Sprintf("%s%s %s\n", marker, sourceBadge, f.Content))
	}

	sb.WriteString(theme.SubtitleStyle.Render("Project Facts"))
	sb.WriteString("\n")
	divWidth := mv.width
	if divWidth <= 12 {
		divWidth = 60
	}
	sb.WriteString(strings.Repeat("─", divWidth-12))
	sb.WriteString("\n")
	if len(projectFacts) == 0 {
		sb.WriteString("  No project-specific facts recorded yet.\n")
	} else {
		// Find display index in original facts slice
		for _, pf := range projectFacts {
			origIdx := -1
			for j, f := range mv.facts {
				if f.ID == pf.ID {
					origIdx = j
					break
				}
			}
			sb.WriteString(renderFactLine(pf, origIdx))
		}
	}
	sb.WriteString("\n")

	sb.WriteString(theme.SubtitleStyle.Render("Global Facts"))
	sb.WriteString("\n")
	divWidth2 := mv.width
	if divWidth2 <= 12 {
		divWidth2 = 60
	}
	sb.WriteString(strings.Repeat("─", divWidth2-12))
	sb.WriteString("\n")
	if len(globalFacts) == 0 {
		sb.WriteString("  No global facts recorded yet.\n")
	} else {
		for _, gf := range globalFacts {
			origIdx := -1
			for j, f := range mv.facts {
				if f.ID == gf.ID {
					origIdx = j
					break
				}
			}
			sb.WriteString(renderFactLine(gf, origIdx))
		}
	}
	sb.WriteString("\n")

	if mv.inputMode {
		sb.WriteString(theme.SubtitleStyle.Render("New Fact: "))
		sb.WriteString(mv.textInput.View())
		sb.WriteString("\n\n")
		sb.WriteString(theme.HelpStyle.Render("[Enter] Save project fact  |  [Esc] Cancel\n"))
	} else {
		sb.WriteString(theme.HelpStyle.Render("[A] Add project fact  |  [D] Delete selected fact  |  [C] Clear project facts  |  [Esc] Close\n"))
	}

	boxWidth := 75
	if mv.width > 0 && mv.width < 85 {
		boxWidth = mv.width - 10
	}

	modal := theme.FloatingModalStyle.Width(boxWidth).Render(sb.String())

	if mv.width > 0 && mv.height > 0 {
		return lipgloss.Place(mv.width, mv.height, lipgloss.Center, lipgloss.Center, modal)
	}

	return modal
}
