package core

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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

// R-002: no gen_ai. string literal, and no literal equal to an operation name or error.type,
// in non test Go code. core/names.go is exempt from the exact match.
var bannedLiterals = []string{"error.type", "invoke_agent", "chat", "execute_tool", "plan"}

func literalViolations(root string) ([]string, error) {
	var out []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		exempt := filepath.ToSlash(rel) == "core/names.go"
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			l, ok := n.(*ast.BasicLit)
			if !ok || l.Kind != token.STRING {
				return true
			}
			v, err := strconv.Unquote(l.Value)
			if err != nil {
				v = l.Value
			}
			if strings.Contains(v, "gen_ai.") || (!exempt && slices.Contains(bannedLiterals, v)) {
				out = append(out, fmt.Sprintf("%s: forbidden literal %s", fset.Position(l.Pos()), l.Value))
			}
			return true
		})
		return nil
	})
	return out, err
}

func TestNoGenAILiterals(t *testing.T) {
	got, err := literalViolations("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range got {
		t.Error(v)
	}
}

// R-002: the scan fails on a temporary non test file holding each banned literal.
func TestLiteralScanCatchesBannedLiterals(t *testing.T) {
	write := func(root, file, lit string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, "core"), 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package x\n\nvar _ = " + strconv.Quote(lit) + "\n"
		if err := os.WriteFile(filepath.Join(root, "core", file), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, lit := range append([]string{"gen_ai.tool.name"}, bannedLiterals...) {
		root := t.TempDir()
		write(root, "other.go", lit)
		if got, err := literalViolations(root); err != nil || len(got) != 1 {
			t.Errorf("%s in other.go: got %v, %v, want one violation", lit, got, err)
		}
	}
	for _, lit := range bannedLiterals {
		root := t.TempDir()
		write(root, "names.go", lit)
		if got, err := literalViolations(root); err != nil || len(got) != 0 {
			t.Errorf("%s in names.go: got %v, %v, want none", lit, got, err)
		}
	}
}
