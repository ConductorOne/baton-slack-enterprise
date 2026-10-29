package gorules_test

import (
	"testing"

	"github.com/quasilyte/go-ruleguard/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestRules(t *testing.T) {
	testdata := analysistest.TestData()
	rules := "rules.go"
	if err := analyzer.Analyzer.Flags.Set("rules", rules); err != nil {
		t.Fatalf("failed to set rules: %s", err)
	}
	analysistest.Run(t, testdata, analyzer.Analyzer, ".")
}
