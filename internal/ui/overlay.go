package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

var ansiRegex = regexp.MustCompile("[\u001b\u009b][[\\]()#;?]*(?:(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]|\u0007)")

// OverlayManager tracks floating modal overlay visibility state.
type OverlayManager struct {
	ShowConfirm     bool
	ShowUndoHistory bool
	ShowMemory      bool
	ShowTemplates   bool
	ShowHelp        bool
	ShowAskUser     bool
}

// AnyActive reports whether any overlay modal is currently visible.
func (o *OverlayManager) AnyActive() bool {
	return o.ShowConfirm || o.ShowUndoHistory || o.ShowMemory || o.ShowTemplates || o.ShowHelp || o.ShowAskUser
}

// CloseAll dismisses every overlay.
func (o *OverlayManager) CloseAll() {
	o.ShowConfirm = false
	o.ShowUndoHistory = false
	o.ShowMemory = false
	o.ShowTemplates = false
	o.ShowHelp = false
	o.ShowAskUser = false
}

// RenderModal centers a modal on top of a dimmed background.
func (o *OverlayManager) RenderModal(bg, modal string, width, height int) string {
	if modal == "" {
		return bg
	}

	bgLines := strings.Split(bg, "\n")
	modalLines := strings.Split(modal, "\n")

	if len(bgLines) < height {
		padding := make([]string, height-len(bgLines))
		bgLines = append(bgLines, padding...)
	}

	modalHeight := len(modalLines)
	modalWidth := 0
	for _, l := range modalLines {
		w := lipgloss.Width(l)
		if w > modalWidth {
			modalWidth = w
		}
	}

	startRow := (height - modalHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (width - modalWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	theme := styles.DefaultTheme
	dimStyle := theme.DimmedOverlayStyle

	for i, mLine := range modalLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}

		plainBgLine := ansiRegex.ReplaceAllString(bgLines[row], "")
		plainBgWidth := len([]rune(plainBgLine))

		if plainBgWidth < startCol {
			plainBgLine = plainBgLine + strings.Repeat(" ", startCol-plainBgWidth)
			plainBgWidth = startCol
		}

		runes := []rune(plainBgLine)
		leftPart := string(runes[:startCol])

		rightPart := ""
		if plainBgWidth > startCol+modalWidth {
			rightPart = string(runes[startCol+modalWidth:])
		}

		mWidth := lipgloss.Width(mLine)
		paddedMLine := mLine
		if mWidth < modalWidth {
			paddedMLine = mLine + strings.Repeat(" ", modalWidth-mWidth)
		}

		bgLines[row] = dimStyle.Render(leftPart) + paddedMLine + dimStyle.Render(rightPart)
	}

	for row := 0; row < len(bgLines); row++ {
		if row < startRow || row >= startRow+modalHeight {
			plainBgLine := ansiRegex.ReplaceAllString(bgLines[row], "")
			bgLines[row] = dimStyle.Render(plainBgLine)
		}
	}

	return strings.Join(bgLines, "\n")
}
