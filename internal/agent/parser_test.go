package agent

import (
	"errors"
	"testing"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

func TestParseToolCall_Native(t *testing.T) {
	native := []api.ToolCall{
		{
			Function: api.FunctionCall{
				Name:      "read_file",
				Arguments: `{"path":"main.go"}`,
			},
		},
	}

	calls, err := ParseToolCall("some noise", native)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "read_file" || calls[0].Args["path"] != "main.go" {
		t.Errorf("expected native parse success, got: %v", calls)
	}
}

func TestParseToolCall_XML(t *testing.T) {
	xmlContent := `
I need to check files first.
<thought>Let's use the read_file tool.</thought>
<tool_call>
{
  "name": "read_file",
  "args": {
    "path": "config.yaml"
  }
}
</tool_call>
`

	calls, err := ParseToolCall(xmlContent, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "read_file" || calls[0].Args["path"] != "config.yaml" {
		t.Errorf("expected XML parse success, got: %v", calls)
	}
}

func TestParseToolCall_Markdown(t *testing.T) {
	mdContent := "Here is the JSON call you requested:\n" +
		"```json\n" +
		"{\n" +
		"  \"name\": \"write_file\",\n" +
		"  \"args\": {\n" +
		"    \"path\": \"test.txt\",\n" +
		"    \"content\": \"hello\"\n" +
		"  }\n" +
		"}\n" +
		"```\n"

	calls, err := ParseToolCall(mdContent, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "write_file" || calls[0].Args["path"] != "test.txt" {
		t.Errorf("expected Markdown parse success, got: %v", calls)
	}
}

func TestParseToolCall_RegexBraces(t *testing.T) {
	content := `
Random prose and then:
{"name": "create_dir", "args": {"path": "internal", "recursive": true}}
more noise.
`

	calls, err := ParseToolCall(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "create_dir" || calls[0].Args["path"] != "internal" {
		t.Errorf("expected Regex/Brace parse success, got: %v", calls)
	}
}

func TestParseToolCall_Failure(t *testing.T) {
	content := "This response has no tool calls, it is just conversational text."
	_, err := ParseToolCall(content, nil)
	if err == nil {
		t.Errorf("expected parse error")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected error to be of type *ParseError, got: %T", err)
	}
}

func TestExtractThought(t *testing.T) {
	// 1. Closed tag
	content1 := "<thought>I am reasoning.</thought> Here is my answer."
	thought1 := ExtractThought(content1)
	if thought1 != "I am reasoning." {
		t.Errorf("expected 'I am reasoning.', got: %q", thought1)
	}

	// 2. Unclosed tag
	content2 := "Prose first. <thought>I started reasoning but didn't close it"
	thought2 := ExtractThought(content2)
	if thought2 != "I started reasoning but didn't close it" {
		t.Errorf("expected reasoning text, got: %q", thought2)
	}

	// 3. No tag
	content3 := "Just simple prose."
	thought3 := ExtractThought(content3)
	if thought3 != "" {
		t.Errorf("expected empty thought, got: %q", thought3)
	}
}
