package summarizertest

// summarizertest exists to provide a helper function
// for the summarizer. We need a separate package to
// prevent import cycles.

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
	"github.com/elastic/go-lookslike/validator"
)

// This duplicates hbtest.SummaryChecks to avoid an import cycle.
// It could be refactored out, but it just isn't worth it.
func SummaryValidator(up uint16, down uint16) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"summary": summaryIsdef(up, down),
	})
}

func summaryIsdef(up uint16, down uint16) isdef.IsDef {
	return isdef.Is("summary", func(path llpath.Path, v interface{}) *llresult.Results {
		js, ok := v.(summarizer.JobSummary)
		if !ok {
			return llresult.SimpleResult(path, false, fmt.Sprintf("expected a *JobSummary, got %v", v))
		}

		if js.Up != up || js.Down != down {
			return llresult.SimpleResult(path, false, fmt.Sprintf("expected up/down to be %d/%d, got %d/%d", up, down, js.Up, js.Down))
		}

		return llresult.ValidResult(path)
	})
}
