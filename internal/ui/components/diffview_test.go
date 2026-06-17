package components

import (
	"strings"
	"testing"

	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

func TestDiffView(t *testing.T) {
	diffText := `diff --git a/test.txt b/test.txt
index 123456..7890ab 100644
--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
-old line
+new line
 unchanged line`

	styles.DefaultTheme.SuccessColor = "#50fa7b"
	styles.DefaultTheme.DangerColor = "#ff5555"
	styles.DefaultTheme.AccentColor = "#8be9fd"
	styles.DefaultTheme.PrimaryColor = "#bd93f9"
	styles.DefaultTheme.FgColor = "#f8f8f2"

	dv := NewDiffView(diffText, 80, 20)

	if dv.width != 80 || dv.height != 20 {
		t.Errorf("expected dimensions 80x20, got %dx%d", dv.width, dv.height)
	}

	content := dv.viewport.View()
	// Since we wrap with viewport and ANSI escapes, we check if the lines were parsed and styled
	if !strings.Contains(content, "new line") {
		t.Error("expected content to contain 'new line'")
	}
	if !strings.Contains(content, "old line") {
		t.Error("expected content to contain 'old line'")
	}

	// Test size update
	dv.SetSize(60, 15)
	if dv.width != 60 || dv.height != 15 {
		t.Errorf("expected dimensions 60x15, got %dx%d", dv.width, dv.height)
	}
}
