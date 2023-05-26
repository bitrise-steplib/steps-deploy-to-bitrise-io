package xcresult3

import (
	"strconv"
	"fmt"
)

type DecoratedTestSummaryGroups struct {
	tests     []ActionTestSummaryGroup
	failures  int
	skipped   int
	flaky     int
	totalTime float64
}

func parseDuration(test ActionTestSummaryGroup) (float64) {
	if test.Duration.Value != "" {
		d, err := strconv.ParseFloat(test.Duration.Value, 64)
		if err == nil {
			return d
		}
	}
	return 0.0
}

func decorateTests(summaryGroups []ActionTestSummaryGroup) (DecoratedTestSummaryGroups) {

    var failure int = 0
    var skipped int = 0
    var flaky   int = 0
    var time float64 = 0.0

	for testIndex, test := range summaryGroups {
		if test.TestStatus.Value == "Failure" {
			failure++
		} else if test.TestStatus.Value == "Skipped" {
			skipped++
		}

		if test.Duration.Value != "" {
			d, err := strconv.ParseFloat(test.Duration.Value, 64)
			if err == nil {
				time += d
			}
		}

        // Flaky correction
		if testIndex > 0 && test.Identifier.Value == summaryGroups[testIndex-1].Identifier.Value {
			summaryGroups[testIndex-1].TestStatus.Value = "Flaky"
			flaky++
			failure--
			summaryGroups[testIndex].Duration.Value = fmt.Sprintf("%f", parseDuration(test) + parseDuration(summaryGroups[testIndex-1]))
		}
	}


	decoratedTestSummaryGroups := DecoratedTestSummaryGroups {
		tests: summaryGroups,
		failures: failure,
		skipped: skipped,
		flaky: flaky,
		totalTime: time,
	}

	return decoratedTestSummaryGroups
}
