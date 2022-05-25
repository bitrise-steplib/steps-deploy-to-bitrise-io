package xcresult3

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"howett.net/plist"
)

// Converter ...
type Converter struct {
	xcresultPth string
}

func majorVersion(document serialized.Object) (int, error) {
	version, err := document.Object("version")
	if err != nil {
		return -1, err
	}

	major, err := version.Value("major")
	if err != nil {
		return -1, err
	}
	return int(major.(uint64)), nil
}

func documentMajorVersion(pth string) (int, error) {
	content, err := fileutil.ReadBytesFromFile(pth)
	if err != nil {
		return -1, err
	}

	var info serialized.Object
	if _, err := plist.Unmarshal(content, &info); err != nil {
		return -1, err
	}

	return majorVersion(info)
}

// Detect ...
func (c *Converter) Detect(files []string) bool {
	if !isXcresulttoolAvailable() {
		log.Debugf("xcresult tool is not available")
		return false
	}

	for _, file := range files {
		if filepath.Ext(file) != ".xcresult" {
			continue
		}

		infoPth := filepath.Join(file, "Info.plist")
		if exist, err := pathutil.IsPathExists(infoPth); err != nil {
			log.Debugf("Failed to find Info.plist at %s: %s", infoPth, err)
			continue
		} else if !exist {
			log.Debugf("No Info.plist found at %s", infoPth)
			continue
		}

		version, err := documentMajorVersion(infoPth)
		if err != nil {
			log.Debugf("failed to get document version: %s", err)
			continue
		}

		if version < 3 {
			log.Debugf("version < 3: %d", version)
			continue
		}

		c.xcresultPth = file
		return true
	}
	return false
}

// XML ...
func (c *Converter) XML() (junit.XML, error) {
	testResultDir := filepath.Dir(c.xcresultPth)

	_, summaries, err := Parse(c.xcresultPth)
	if err != nil {
		return junit.XML{}, err
	}

	var xmlData junit.XML
	{
		testSuiteCount := testSuiteCountInSummaries(summaries)
		xmlData.TestSuites = make([]junit.TestSuite, testSuiteCount)
	}
	summariesCount := len(summaries)
	log.Printf("Summaries Count: %d", summariesCount)
	for summaryIdx, summary := range summaries {
		testSuiteOrder, testsByName := summary.tests()

		for testSuiteIdx, name := range testSuiteOrder {
			tests := testsByName[name]

			testSuite, err := genTestSuite(name, summary, tests, testResultDir, c.xcresultPth)
			if err != nil {
				return junit.XML{}, err
			}

			xmlData.TestSuites[summaryIdx*summariesCount+testSuiteIdx] = testSuite
		}
	}

	return xmlData, nil
}

func testSuiteCountInSummaries(summaries []ActionTestPlanRunSummaries) int {
	testSuiteCount := 0
	for _, summary := range summaries {
		testSuiteOrder, _ := summary.tests()
		testSuiteCount += len(testSuiteOrder)
	}
	return testSuiteCount
}

func genTestSuite(name string,
	summary ActionTestPlanRunSummaries,
	tests []ActionTestSummaryGroup,
	testResultDir string,
	xcresultPath string,
) (junit.TestSuite, error) {
	t1 := time.Now()
	log.Printf("=> Generating test suite [%s] - started ...", name)
	testSuite := junit.TestSuite{
		Name:     name,
		Tests:    len(tests),
		Failures: summary.failuresCount(name),
		Skipped:  summary.skippedCount(name),
		Time:     summary.totalTime(name),
	}

	var genTestSuiteErr error
	var wg sync.WaitGroup
	var mtx sync.RWMutex
	testSuite.TestCases = make([]junit.TestCase, len(tests))
	log.Printf("--> len(tests): %v", len(tests))

	maxParallel := runtime.NumCPU() * 2
	log.Printf("maxParallel: %v", maxParallel)
	testIdx := 0
	for testIdx < len(tests) {
		for i := 0; i < maxParallel && testIdx < len(tests); i++ {
			test := tests[testIdx]
			wg.Add(1)

			go func(test ActionTestSummaryGroup, testIdx int) {
				defer wg.Done()

				testCase, err := genTestCase(test, xcresultPath, testResultDir)
				if err != nil {
					mtx.Lock()
					genTestSuiteErr = err
					mtx.Unlock()
				}

				testSuite.TestCases[testIdx] = testCase
			}(test, testIdx)

			testIdx += 1
		}

		wg.Wait()
	}

	t2 := time.Now()
	log.Printf("-> Generating test suite [%s] - DONE %v", name, t2.Sub(t1))

	return testSuite, genTestSuiteErr
}

func genTestCase(test ActionTestSummaryGroup, xcresultPath, testResultDir string) (junit.TestCase, error) {
	var duartion float64
	if test.Duration.Value != "" {
		var err error
		duartion, err = strconv.ParseFloat(test.Duration.Value, 64)
		if err != nil {
			return junit.TestCase{}, err
		}
	}

	var failure *junit.Failure
	var skipped *junit.Skipped
	switch test.TestStatus.Value {
	case "Failure":
		testSummary, err := test.loadActionTestSummary(xcresultPath)
		if err != nil {
			return junit.TestCase{}, err
		}

		failureMessage := ""
		for _, aTestFailureSummary := range testSummary.FailureSummaries.Values {
			file := aTestFailureSummary.FileName.Value
			line := aTestFailureSummary.LineNumber.Value
			message := aTestFailureSummary.Message.Value

			if len(failureMessage) > 0 {
				failureMessage += "\n"
			}
			failureMessage += fmt.Sprintf("%s:%s - %s", file, line, message)
		}

		failure = &junit.Failure{
			Value: failureMessage,
		}
	case "Skipped":
		skipped = &junit.Skipped{}
	}

	if err := test.exportScreenshots(xcresultPath, testResultDir); err != nil {
		return junit.TestCase{}, err
	}

	return junit.TestCase{
		Name:      test.Name.Value,
		ClassName: strings.Split(test.Identifier.Value, "/")[0],
		Failure:   failure,
		Skipped:   skipped,
		Time:      duartion,
	}, nil
}
