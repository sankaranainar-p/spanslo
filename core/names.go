package core

import "go.yaml.in/yaml/v3"

// Names holds the OpenTelemetry GenAI and spanslo names read from
// spec/semconv-mapping.yaml. Callers pass the file bytes; core does no I/O (R-002, R-008).
type Names struct {
	Operations Operations `yaml:"operations"`
	Attributes Attributes `yaml:"attributes"`
	Events     struct {
		EvaluationResult EvaluationResult `yaml:"evaluation_result"`
	} `yaml:"events"`
	Spanslo struct {
		Task TaskNames `yaml:"task"`
		Tool ToolNames `yaml:"tool"`
	} `yaml:"spanslo_attributes"`
}

// Operations are the values of the operation name attribute.
type Operations struct {
	Run       string `yaml:"run"`
	ModelCall string `yaml:"model_call"`
	ToolCall  string `yaml:"tool_call"`
	Plan      string `yaml:"plan"`
}

// Attributes are span attribute names.
type Attributes struct {
	OperationName     string `yaml:"operation_name"`
	ToolName          string `yaml:"tool_name"`
	ToolCallID        string `yaml:"tool_call_id"`
	ToolCallArguments string `yaml:"tool_call_arguments"`
	InputTokens       string `yaml:"input_tokens"`
	OutputTokens      string `yaml:"output_tokens"`
	AgentName         string `yaml:"agent_name"`
	ProviderName      string `yaml:"provider_name"`
	ErrorType         string `yaml:"error_type"`
}

// EvaluationResult names the evaluation result event, its attributes and the task success scoring.
type EvaluationResult struct {
	Name  string `yaml:"name"`
	Attrs struct {
		EvaluationName string `yaml:"evaluation_name"`
		ScoreValue     string `yaml:"score_value"`
		ScoreLabel     string `yaml:"score_label"`
	} `yaml:"attributes"`
	SuccessName      string  `yaml:"success_name"`
	SuccessValuePass float64 `yaml:"success_value_pass"`
	SuccessValueFail float64 `yaml:"success_value_fail"`
	SuccessLabelPass string  `yaml:"success_label_pass"`
	SuccessLabelFail string  `yaml:"success_label_fail"`
}

// TaskNames are spanslo attributes on the root span.
type TaskNames struct {
	FallbackTools string `yaml:"fallback_tools"`
}

// ToolNames are spanslo attributes on execute_tool spans.
type ToolNames struct {
	DestructiveHint string `yaml:"destructive_hint"`
	OpenWorldHint   string `yaml:"open_world_hint"`
}

// LoadNames parses the bytes of spec/semconv-mapping.yaml.
func LoadNames(data []byte) (*Names, error) {
	var n Names
	if err := yaml.Unmarshal(data, &n); err != nil {
		return nil, err
	}
	return &n, nil
}
