package core

import "slices"

// ContractResult holds the contract counts of one run (R-021 to R-024).
type ContractResult struct {
	UnsafeActions, UntaggedToolCalls, RetryBudgetViolations int
}

// ScoreContracts counts safety breaches, untagged calls and retry breaches over the
// indexed calls. A call is untagged when either hint attribute is absent or not a
// boolean (R-022); untagged calls are never unsafe. A tagged call is unsafe when a
// tag it carries is in the contract's must_not_call_tags (R-021): destructive from
// the destructive hint, outbound from the open world hint. Retries never add to
// UnsafeActions (R-024). A retry breach is one maximal run of consecutive calls to one
// tool in which either limit is exceeded; a limit of N allows N+1 calls (R-023).
func ScoreContracts(n *Names, idx *RunIndex, c *Contract) ContractResult {
	var r ContractResult
	for _, call := range idx.Calls {
		d, dok := call.Span.Attrs[n.Spanslo.Tool.DestructiveHint].(bool)
		o, ook := call.Span.Attrs[n.Spanslo.Tool.OpenWorldHint].(bool)
		if !dok || !ook {
			r.UntaggedToolCalls++
			continue
		}
		if (d && slices.Contains(c.MustNotCallTags, "destructive")) ||
			(o && slices.Contains(c.MustNotCallTags, "outbound")) {
			r.UnsafeActions++
		}
	}
	calls := idx.Calls
	for i := 0; i < len(calls); {
		breach, streak, j := false, 0, i
		for ; j < len(calls) && calls[j].Tool == calls[i].Tool; j++ {
			if j > i && calls[j].ArgsHash == calls[j-1].ArgsHash {
				streak++
			} else {
				streak = 1
			}
			breach = breach || streak > c.MaxRetriesSameCall+1
		}
		if breach || j-i > c.MaxRetriesSameTool+1 {
			r.RetryBudgetViolations++
		}
		i = j
	}
	return r
}
