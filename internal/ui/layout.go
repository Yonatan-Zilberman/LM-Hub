package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// LayoutKind describes the main TUI panel arrangement.
type LayoutKind int

const (
	// LayoutSingle is a full-width content area (Ask mode default).
	LayoutSingle LayoutKind = iota
	// LayoutSplitRight places content left and a status sidebar on the right.
	LayoutSplitRight
	// LayoutOverlay is used when a modal covers the main layout.
	LayoutOverlay
)

// LayoutConfig holds computed dimensions for the current frame.
type LayoutConfig struct {
	Kind          LayoutKind
	Width         int
	Height        int
	ContentWidth  int
	SidebarWidth  int
	SidebarHeight int
	ShowSidebar   bool
}

// LayoutManager computes panel dimensions based on terminal size and active view.
type LayoutManager struct{}

// NewLayoutManager creates a LayoutManager instance.
func NewLayoutManager() *LayoutManager {
	return &LayoutManager{}
}

// Compute returns layout dimensions for the given view and terminal size.
func (lm *LayoutManager) Compute(activeView ActiveView, width, height int, isLoadingModel bool) LayoutConfig {
	cfg := LayoutConfig{
		Kind:          LayoutSingle,
		Width:         width,
		Height:        height,
		ContentWidth:  width,
		SidebarWidth:  30,
		SidebarHeight: height - 10,
	}

	if cfg.SidebarHeight < 10 {
		cfg.SidebarHeight = 10
	}

	cfg.ShowSidebar = width > 90 &&
		!isLoadingModel &&
		activeView != ViewModelSelect &&
		activeView != ViewMetrics &&
		(activeView == ViewChat || activeView == ViewPlanChat || activeView == ViewPlan || activeView == ViewBuild)

	if cfg.ShowSidebar {
		cfg.Kind = LayoutSplitRight
		cfg.ContentWidth = width - cfg.SidebarWidth
	}

	return cfg
}

// JoinContentAndSidebar renders the main content beside an optional right sidebar.
func (lm *LayoutManager) JoinContentAndSidebar(content, sidebar string, cfg LayoutConfig) string {
	if !cfg.ShowSidebar || sidebar == "" {
		return content
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, content, sidebar)
}
