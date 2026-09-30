// Package testattachment links test attachment files to JUnit test cases by file name.
//
// The file name convention is <classname>__<name>[__run<k>]__<label>.<ext>, where the label is
// required and must not contain "__", and run<k> selects the k-th occurrence of the test case.
package testattachment

import (
	"errors"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/testasset"
	"github.com/bitrise-io/go-steputils/v2/testreport"
)

const (
	separator = "__"
	runPrefix = separator + "run"
)

var (
	// ErrUnsupportedType is returned when the file extension is not a supported attachment type.
	ErrUnsupportedType = errors.New("unsupported attachment type")
	// ErrNoConvention is returned when the file name does not follow the naming convention at all.
	ErrNoConvention = errors.New("file name does not follow the attachment naming convention")
	// ErrMissingLabel is returned when the file name has no label after the last separator.
	ErrMissingLabel = errors.New("file name has no label")
	// ErrUnknownTest is returned when no test case matches the file name.
	ErrUnknownTest = errors.New("no test case matches the file name")
	// ErrAmbiguousTest is returned when more than one test case maps to the file name.
	ErrAmbiguousTest = errors.New("more than one test case matches the file name")
	// ErrUnknownRun is returned when the requested run is higher than the number of test case occurrences.
	ErrUnknownRun = errors.New("test case has no such run")
)

// Android's /sdcard doesn't allow these characters in file names. They become "_" on both the
// test case side and the file name side, so either spelling matches.
var sanitizer = strings.NewReplacer(
	`"`, "_", "*", "_", "/", "_", ":", "_", "<", "_",
	">", "_", "?", "_", `\`, "_", "|", "_",
)

// Key returns the file name key of a test case: the cleaned class name and name joined by "__".
func Key(className, name string) string {
	return sanitizer.Replace(className) + separator + sanitizer.Replace(name)
}

// Match is a test case occurrence that an attachment file belongs to.
type Match struct {
	TestCase *testreport.TestCase
	// Run is the run number from the file name, or 0 if the file name has none (the first occurrence).
	Run   int
	Label string
}

type testID struct {
	className string
	name      string
}

type entry struct {
	id    testID
	suite *testreport.TestSuite
	// occurrences are counted within one suite, the same way the Tests tab merges reruns.
	occurrences []*testreport.TestCase
}

// Index maps file name keys to the test case occurrences of a report.
type Index struct {
	entries   map[string]*entry
	ambiguous map[string]bool
}

// NewIndex indexes every test case of the report, including nested suites. The index points into
// the report, so the report must not be modified structurally while the index is in use.
func NewIndex(report *testreport.TestReport) Index {
	idx := Index{entries: map[string]*entry{}, ambiguous: map[string]bool{}}
	for i := range report.TestSuites {
		idx.addSuite(&report.TestSuites[i])
	}
	for key := range idx.ambiguous {
		delete(idx.entries, key)
	}
	return idx
}

func (idx Index) addSuite(suite *testreport.TestSuite) {
	for i := range suite.TestCases {
		testCase := &suite.TestCases[i]
		key := Key(testCase.ClassName, testCase.Name)
		id := testID{className: testCase.ClassName, name: testCase.Name}

		existing, ok := idx.entries[key]
		switch {
		case !ok:
			idx.entries[key] = &entry{id: id, suite: suite, occurrences: []*testreport.TestCase{testCase}}
		case existing.id != id, existing.suite != suite:
			idx.ambiguous[key] = true
		default:
			existing.occurrences = append(existing.occurrences, testCase)
		}
	}
	for i := range suite.TestSuites {
		idx.addSuite(&suite.TestSuites[i])
	}
}

// Ambiguous returns the keys that more than one distinct test case, or the same test case in more
// than one suite, maps to, sorted.
func (idx Index) Ambiguous() []string {
	keys := make([]string, 0, len(idx.ambiguous))
	for key := range idx.ambiguous {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Match resolves an attachment file name to a test case occurrence. Only the base name of the
// path is used.
func (idx Index) Match(fileName string) (Match, error) {
	base := filepath.Base(fileName)
	if !testasset.IsSupportedAssetType(base) {
		return Match{}, ErrUnsupportedType
	}

	stem := strings.TrimSuffix(base, filepath.Ext(base))
	sep := strings.LastIndex(stem, separator)
	if sep < 0 {
		return Match{}, ErrNoConvention
	}
	label := stem[sep+len(separator):]
	if label == "" {
		return Match{}, ErrMissingLabel
	}
	key := sanitizer.Replace(stem[:sep])

	// A test whose own name ends in "__run<k>" wins over the run suffix.
	if match, found, err := idx.lookup(key, 0, label); found {
		return match, err
	}

	runSep := strings.LastIndex(key, runPrefix)
	if runSep < 0 {
		return Match{}, ErrUnknownTest
	}
	run, err := strconv.Atoi(key[runSep+len(runPrefix):])
	if err != nil || run < 1 {
		return Match{}, ErrUnknownTest
	}
	if match, found, err := idx.lookup(key[:runSep], run, label); found {
		return match, err
	}
	return Match{}, ErrUnknownTest
}

func (idx Index) lookup(key string, run int, label string) (Match, bool, error) {
	if idx.ambiguous[key] {
		return Match{}, true, ErrAmbiguousTest
	}
	e, ok := idx.entries[key]
	if !ok {
		return Match{}, false, nil
	}

	occurrence := 0
	if run > 0 {
		occurrence = run - 1
	}
	if occurrence >= len(e.occurrences) {
		return Match{}, true, ErrUnknownRun
	}
	return Match{TestCase: e.occurrences[occurrence], Run: run, Label: label}, true, nil
}
