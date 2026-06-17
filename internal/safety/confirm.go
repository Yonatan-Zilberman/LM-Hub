package safety

import (
	"github.com/yonatanzilberman/lmhub/internal/tools"
)

// ConfirmMsg is dispatched to the Bubbletea loop to prompt the user for execution approval.
type ConfirmMsg struct {
	ToolName     string                 `json:"tool_name"`
	Args         map[string]interface{} `json:"args"`
	Level        tools.PermissionLevel  `json:"level"`
	Description  string                 `json:"description"`
	ResponseChan chan bool              `json:"-"`
}

// NewConfirmMsg creates a new confirmation request and returns the message along with its response channel.
func NewConfirmMsg(toolName string, args map[string]interface{}, level tools.PermissionLevel, desc string) (ConfirmMsg, chan bool) {
	ch := make(chan bool, 1)
	return ConfirmMsg{
		ToolName:     toolName,
		Args:         args,
		Level:        level,
		Description:  desc,
		ResponseChan: ch,
	}, ch
}
