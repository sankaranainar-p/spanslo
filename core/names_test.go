package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadNames(t *testing.T) {
	data, err := os.ReadFile("../spec/semconv-mapping.yaml")
	if err != nil {
		t.Fatal(err)
	}
	n, err := LoadNames(data)
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string][2]string{
		"run":          {n.Operations.Run, "invoke_agent"},
		"model_call":   {n.Operations.ModelCall, "chat"},
		"tool_call":    {n.Operations.ToolCall, "execute_tool"},
		"plan":         {n.Operations.Plan, "plan"},
		"operation":    {n.Attributes.OperationName, "gen_ai.operation.name"},
		"tool_name":    {n.Attributes.ToolName, "gen_ai.tool.name"},
		"tool_call_id": {n.Attributes.ToolCallID, "gen_ai.tool.call.id"},
		"tool_args":    {n.Attributes.ToolCallArguments, "gen_ai.tool.call.arguments"},
		"in_tokens":    {n.Attributes.InputTokens, "gen_ai.usage.input_tokens"},
		"out_tokens":   {n.Attributes.OutputTokens, "gen_ai.usage.output_tokens"},
		"agent":        {n.Attributes.AgentName, "gen_ai.agent.name"},
		"provider":     {n.Attributes.ProviderName, "gen_ai.provider.name"},
		"error_type":   {n.Attributes.ErrorType, "error.type"},
		"eval_event":   {n.Events.EvaluationResult.Name, "gen_ai.evaluation.result"},
		"eval_name":    {n.Events.EvaluationResult.Attrs.EvaluationName, "gen_ai.evaluation.name"},
		"score_value":  {n.Events.EvaluationResult.Attrs.ScoreValue, "gen_ai.evaluation.score.value"},
		"score_label":  {n.Events.EvaluationResult.Attrs.ScoreLabel, "gen_ai.evaluation.score.label"},
		"success":      {n.Events.EvaluationResult.SuccessName, "spanslo.task_success"},
		"label_pass":   {n.Events.EvaluationResult.SuccessLabelPass, "pass"},
		"label_fail":   {n.Events.EvaluationResult.SuccessLabelFail, "fail"},
		"fallback":     {n.Spanslo.Task.FallbackTools, "spanslo.task.fallback_tools"},
		"destructive":  {n.Spanslo.Tool.DestructiveHint, "spanslo.tool.destructive_hint"},
		"open_world":   {n.Spanslo.Tool.OpenWorldHint, "spanslo.tool.open_world_hint"},
	} {
		if got[0] != got[1] {
			t.Errorf("%s: got %q, want %q", name, got[0], got[1])
		}
	}
	if n.Events.EvaluationResult.SuccessValuePass != 1.0 || n.Events.EvaluationResult.SuccessValueFail != 0.0 {
		t.Errorf("success values: %+v", n.Events.EvaluationResult)
	}
}

// R-002: no gen_ai. string literal in non test Go code.
func TestNoGenAILiterals(t *testing.T) {
	root := ".."
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if l, ok := n.(*ast.BasicLit); ok && l.Kind == token.STRING && strings.Contains(l.Value, "gen_ai.") {
				t.Errorf("%s: gen_ai literal %s", fset.Position(l.Pos()), l.Value)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
