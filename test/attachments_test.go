package test

import (
	"bytes"
	"encoding/xml"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/testreport"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_linkAttachments(t *testing.T) {
	reportDir := filepath.Join("/bitrise/test_results/step", "report")

	tests := []struct {
		name            string
		testCases       []testreport.TestCase
		attachmentNames []string
		want            [][]testreport.Property
		wantWarnings    []string
	}{
		{
			name:            "links a file in a subfolder under the name it is uploaded with",
			testCases:       []testreport.TestCase{{ClassName: "com.example.LoginTest", Name: "emptyState"}},
			attachmentNames: []string{"screenshots/com.example.LoginTest__emptyState__1.png"},
			want: [][]testreport.Property{
				{{Name: "attachment_0", Value: "screenshots/com.example.LoginTest__emptyState__1.png"}},
			},
		},
		{
			name: "links a run suffix to that occurrence of the test case",
			testCases: []testreport.TestCase{
				{ClassName: "com.example.LoginTest", Name: "wrongPassword"},
				{ClassName: "com.example.LoginTest", Name: "wrongPassword"},
			},
			attachmentNames: []string{
				"com.example.LoginTest__wrongPassword__1.png",
				"com.example.LoginTest__wrongPassword__run2__1.png",
			},
			want: [][]testreport.Property{
				{{Name: "attachment_0", Value: "com.example.LoginTest__wrongPassword__1.png"}},
				{{Name: "attachment_0", Value: "com.example.LoginTest__wrongPassword__run2__1.png"}},
			},
		},
		{
			name:      "orders the attachments of a test case by label, numbers compared by value",
			testCases: []testreport.TestCase{{ClassName: "LoginTest", Name: "emptyState"}},
			attachmentNames: []string{
				"LoginTest__emptyState__10.png",
				"LoginTest__emptyState__2.png",
				"LoginTest__emptyState__api34_1.log",
				"LoginTest__emptyState__1.png",
			},
			want: [][]testreport.Property{{
				{Name: "attachment_0", Value: "LoginTest__emptyState__1.png"},
				{Name: "attachment_1", Value: "LoginTest__emptyState__2.png"},
				{Name: "attachment_2", Value: "LoginTest__emptyState__10.png"},
				{Name: "attachment_3", Value: "LoginTest__emptyState__api34_1.log"},
			}},
		},
		{
			name: "keeps existing properties, skips referenced files and numbers after the highest attachment",
			testCases: []testreport.TestCase{{
				ClassName: "LoginTest",
				Name:      "emptyState",
				Properties: &testreport.Properties{Property: []testreport.Property{
					{Name: "device", Value: "api34"},
					{Name: "attachment_3", Value: "LoginTest__emptyState__1.png"},
				}},
			}},
			attachmentNames: []string{"LoginTest__emptyState__1.png", "LoginTest__emptyState__2.png"},
			want: [][]testreport.Property{{
				{Name: "device", Value: "api34"},
				{Name: "attachment_3", Value: "LoginTest__emptyState__1.png"},
				{Name: "attachment_4", Value: "LoginTest__emptyState__2.png"},
			}},
		},
		{
			name:            "ignores files without the naming convention without a warning",
			testCases:       []testreport.TestCase{{ClassName: "LoginTest", Name: "emptyState"}},
			attachmentNames: []string{"screenshot_1.png", "logcat.txt", "LoginTest__emptyState__1.gif"},
			want:            [][]testreport.Property{nil},
		},
		{
			name:            "warns about files that follow the convention but match no test case",
			testCases:       []testreport.TestCase{{ClassName: "LoginTest", Name: "emptyState"}},
			attachmentNames: []string{"LoginTest__missing__1.png", "LoginTest__emptyState__run2__1.png", "LoginTest__emptyState__.png"},
			want:            [][]testreport.Property{nil},
			wantWarnings: []string{
				"Test attachment LoginTest__missing__1.png is not linked to a test case: no test case matches the file name",
				"Test attachment LoginTest__emptyState__run2__1.png is not linked to a test case: test case has no such run",
				"Test attachment LoginTest__emptyState__.png is not linked to a test case: file name has no label",
			},
		},
		{
			name:            "links neither file when two have the same test case and label",
			testCases:       []testreport.TestCase{{ClassName: "LoginTest", Name: "emptyState"}},
			attachmentNames: []string{"a/LoginTest__emptyState__1.png", "b/LoginTest__emptyState__1.png", "LoginTest__emptyState__01.png"},
			want: [][]testreport.Property{
				{{Name: "attachment_0", Value: "LoginTest__emptyState__01.png"}},
			},
			wantWarnings: []string{
				"Test attachment a/LoginTest__emptyState__1.png is not linked to a test case: another file has the same test case and label",
				"Test attachment b/LoginTest__emptyState__1.png is not linked to a test case: another file has the same test case and label",
			},
		},
		{
			name: "warns about a file that more than one test case matches",
			testCases: []testreport.TestCase{
				{ClassName: "LoginTest", Name: "a:b"},
				{ClassName: "LoginTest", Name: "a/b"},
			},
			attachmentNames: []string{"LoginTest__a_b__1.png"},
			want:            [][]testreport.Property{nil, nil},
			wantWarnings: []string{
				"Test attachment LoginTest__a_b__1.png is not linked to a test case: more than one test case matches the file name",
			},
		},
		{
			name: "doesn't warn about ambiguous test cases that no file refers to",
			testCases: []testreport.TestCase{
				{ClassName: "LoginTest", Name: "a:b"},
				{ClassName: "LoginTest", Name: "a/b"},
			},
			attachmentNames: []string{"screenshot.png"},
			want:            [][]testreport.Property{nil, nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := testreport.TestReport{TestSuites: []testreport.TestSuite{{Name: "suite", TestCases: tt.testCases}}}
			var attachmentPaths []string
			for _, name := range tt.attachmentNames {
				attachmentPaths = append(attachmentPaths, filepath.Join(reportDir, name))
			}
			var logs bytes.Buffer

			linkAttachments(&report, attachmentPaths, "report", log.NewLogger(log.WithOutput(&logs)))

			var got [][]testreport.Property
			for _, testCase := range report.TestSuites[0].TestCases {
				if testCase.Properties == nil {
					got = append(got, nil)
					continue
				}
				got = append(got, testCase.Properties.Property)
			}
			assert.Equal(t, tt.want, got)
			for _, warning := range tt.wantWarnings {
				assert.Contains(t, logs.String(), warning)
			}
			if len(tt.wantWarnings) == 0 {
				assert.Empty(t, logs.String())
			}
		})
	}
}

func Test_linkAttachments_nestedSuites(t *testing.T) {
	report := testreport.TestReport{TestSuites: []testreport.TestSuite{{
		Name: "outer",
		TestSuites: []testreport.TestSuite{{
			Name: "inner",
			TestCases: []testreport.TestCase{{
				ClassName: "LoginTest",
				Name:      "emptyState",
				Properties: &testreport.Properties{Property: []testreport.Property{
					{Name: "attachment_0", Value: "LoginTest__emptyState__1.png"},
				}},
			}},
		}},
	}}}
	attachmentPaths := []string{
		filepath.Join("/bitrise/test_results/step/report", "LoginTest__emptyState__1.png"),
		filepath.Join("/bitrise/test_results/step/report", "LoginTest__emptyState__2.png"),
	}

	linkAttachments(&report, attachmentPaths, "report", log.NewLogger())

	assert.Equal(t, []testreport.Property{
		{Name: "attachment_0", Value: "LoginTest__emptyState__1.png"},
		{Name: "attachment_1", Value: "LoginTest__emptyState__2.png"},
	}, report.TestSuites[0].TestSuites[0].TestCases[0].Properties.Property)
}

func Test_linkAttachments_sameTestCaseInTwoSuites(t *testing.T) {
	newReport := func() testreport.TestReport {
		return testreport.TestReport{TestSuites: []testreport.TestSuite{
			{Name: "debug", TestCases: []testreport.TestCase{{ClassName: "LoginTest", Name: "emptyState"}}},
			{Name: "release", TestCases: []testreport.TestCase{{ClassName: "LoginTest", Name: "emptyState"}}},
		}}
	}

	t.Run("doesn't warn when no file refers to the test case", func(t *testing.T) {
		report := newReport()
		var logs bytes.Buffer

		linkAttachments(&report, []string{"/bitrise/test_results/step/report/screenshot.png"}, "report", log.NewLogger(log.WithOutput(&logs)))

		assert.Empty(t, logs.String())
		assert.Nil(t, report.TestSuites[0].TestCases[0].Properties)
		assert.Nil(t, report.TestSuites[1].TestCases[0].Properties)
	})

	t.Run("warns about a file that refers to the test case", func(t *testing.T) {
		report := newReport()
		var logs bytes.Buffer

		linkAttachments(&report, []string{"/bitrise/test_results/step/report/LoginTest__emptyState__1.png"}, "report", log.NewLogger(log.WithOutput(&logs)))

		assert.Contains(t, logs.String(), "Test attachment LoginTest__emptyState__1.png is not linked to a test case: more than one test case matches the file name")
		assert.Nil(t, report.TestSuites[0].TestCases[0].Properties)
		assert.Nil(t, report.TestSuites[1].TestCases[0].Properties)
	})
}

func Test_ParseTestResults_linksConventionNamedAttachments(t *testing.T) {
	testsDir := t.TempDir()
	testDir := filepath.Join(testsDir, "step")
	phaseDir := filepath.Join(testDir, "report")

	require.NoError(t, createDummyFilesInDirWithContent(testDir, `{"title": "test title"}`, []string{"step-info.json"}))
	require.NoError(t, createDummyFilesInDirWithContent(phaseDir, `{"test-name": "UI tests"}`, []string{"test-info.json"}))
	require.NoError(t, createDummyFilesInDirWithContent(phaseDir, `<testsuite name="LoginTest"><testcase classname="com.example.LoginTest" name="emptyState"/></testsuite>`, []string{"result.xml"}))
	require.NoError(t, createDummyFilesInDirWithContent(phaseDir, "png", []string{"com.example.LoginTest__emptyState__1.png", "screenshot.png"}))

	results, err := ParseTestResults(testsDir, false, pathutil.NewPathChecker(), pathutil.NewPathModifier(), log.NewLogger())
	require.NoError(t, err)
	require.Len(t, results, 1)

	var report testreport.TestReport
	require.NoError(t, xml.Unmarshal(results[0].XMLContent, &report))
	testCase := report.TestSuites[0].TestCases[0]
	require.NotNil(t, testCase.Properties)
	assert.Equal(t, []testreport.Property{{Name: "attachment_0", Value: "com.example.LoginTest__emptyState__1.png"}}, stripXMLNames(testCase.Properties.Property))
	assert.Len(t, results[0].AttachmentPaths, 2)
}

func stripXMLNames(properties []testreport.Property) []testreport.Property {
	for i := range properties {
		properties[i].XMLName = xml.Name{}
	}
	return properties
}
