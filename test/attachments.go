package test

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/testattachment"
	"github.com/bitrise-io/go-steputils/v2/testreport"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/maruel/natural"
)

const attachmentPropertyPrefix = "attachment_"

type linkedAttachment struct {
	label    string
	fileName string
}

// linkAttachments adds an attachment_N property to the test case that each convention-named file
// belongs to. The property value is the file name the file is uploaded under, which is what the
// backend resolves it by.
func linkAttachments(report *testreport.TestReport, attachmentPaths []string, reportName string, logger log.Logger) {
	idx := testattachment.NewIndex(report)
	referenced := referencedAttachments(report)

	byTestCase := map[*testreport.TestCase][]linkedAttachment{}
	for _, path := range attachmentPaths {
		fileName := relativeFilePath(path, reportName)
		if referenced[fileName] {
			continue
		}

		match, err := idx.Match(path)
		if errors.Is(err, testattachment.ErrUnsupportedType) || errors.Is(err, testattachment.ErrNoConvention) {
			continue
		}
		if err != nil {
			logger.Warnf("Test attachment %s is not linked to a test case: %s", fileName, err)
			continue
		}

		byTestCase[match.TestCase] = append(byTestCase[match.TestCase], linkedAttachment{label: match.Label, fileName: fileName})
	}

	for testCase, attachments := range byTestCase {
		baseNameCounts := map[string]int{}
		for _, attachment := range attachments {
			baseNameCounts[filepath.Base(attachment.fileName)]++
		}

		var kept []linkedAttachment
		for _, attachment := range attachments {
			if baseNameCounts[filepath.Base(attachment.fileName)] > 1 {
				logger.Warnf("Test attachment %s is not linked to a test case: another file in the report has the same name", attachment.fileName)
				continue
			}
			kept = append(kept, attachment)
		}

		slices.SortStableFunc(kept, func(a, b linkedAttachment) int {
			return natural.Compare(a.label, b.label)
		})
		addAttachmentProperties(testCase, kept)
	}
}

func referencedAttachments(report *testreport.TestReport) map[string]bool {
	referenced := map[string]bool{}
	var addSuite func(suite *testreport.TestSuite)
	addSuite = func(suite *testreport.TestSuite) {
		for i := range suite.TestCases {
			properties := suite.TestCases[i].Properties
			if properties == nil {
				continue
			}
			for _, property := range properties.Property {
				if strings.HasPrefix(property.Name, attachmentPropertyPrefix) {
					referenced[property.Value] = true
				}
			}
		}
		for i := range suite.TestSuites {
			addSuite(&suite.TestSuites[i])
		}
	}
	for i := range report.TestSuites {
		addSuite(&report.TestSuites[i])
	}
	return referenced
}

// addAttachmentProperties numbers the new properties after the highest existing attachment_N, so
// they never collide with the ones already in the XML.
func addAttachmentProperties(testCase *testreport.TestCase, attachments []linkedAttachment) {
	if len(attachments) == 0 {
		return
	}
	if testCase.Properties == nil {
		testCase.Properties = &testreport.Properties{}
	}

	next := 0
	for _, property := range testCase.Properties.Property {
		suffix, ok := strings.CutPrefix(property.Name, attachmentPropertyPrefix)
		if !ok {
			continue
		}
		if n, err := strconv.Atoi(suffix); err == nil && n >= next {
			next = n + 1
		}
	}

	for i, attachment := range attachments {
		testCase.Properties.Property = append(testCase.Properties.Property, testreport.Property{
			Name:  fmt.Sprintf("%s%d", attachmentPropertyPrefix, next+i),
			Value: attachment.fileName,
		})
	}
}
