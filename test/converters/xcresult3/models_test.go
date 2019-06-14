package xcresult3

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bitrise-steplib/steps-xcode-test/pretty"
)

func TestActionTestSummaryGroup_producingTargetAndTestCaseName(t *testing.T) {
	tests := []struct {
		name         string
		identifier   string
		wantTarget   string
		wantTestCase string
	}{
		{
			name:         "simple test",
			identifier:   "Xcode11TestUITests2/testFail()",
			wantTarget:   "Xcode11TestUITests2",
			wantTestCase: "Xcode11TestUITests2.testFail()",
		},
		{
			name:         "invalid format",
			identifier:   "Xcode11TestUITests2testFail()",
			wantTarget:   "",
			wantTestCase: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := ActionTestSummaryGroup{}
			g.Identifier.Value = tt.identifier

			gotTarget, gotTestCase := g.producingTargetAndTestCaseName()
			if gotTarget != tt.wantTarget {
				t.Errorf("ActionTestSummaryGroup.producingTargetAndTestCaseName() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
			if gotTestCase != tt.wantTestCase {
				t.Errorf("ActionTestSummaryGroup.producingTargetAndTestCaseName() gotTestCase = %v, want %v", gotTestCase, tt.wantTestCase)
			}
		})
	}
}

func TestActionTestSummaryGroup_testsWithStatus(t *testing.T) {

	tests := []struct {
		name       string
		group      ActionTestSummaryGroup
		subtests   []ActionTestSummaryGroup
		wantGroups []ActionTestSummaryGroup
	}{
		{
			name: "status in the root ActionTestSummaryGroup",
			group: ActionTestSummaryGroup{
				TestStatus: TestStatus{Value: "success"},
			},
			wantGroups: []ActionTestSummaryGroup{ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}}},
		},
		{
			name: "status in a sub ActionTestSummaryGroup",
			group: ActionTestSummaryGroup{
				Subtests: Subtests{
					Values: []ActionTestSummaryGroup{
						ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}},
					},
				},
			},
			wantGroups: []ActionTestSummaryGroup{ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}}},
		},
		{
			name:       "no status",
			group:      ActionTestSummaryGroup{},
			wantGroups: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGroups := tt.group.testsWithStatus()
			if !reflect.DeepEqual(gotGroups, tt.wantGroups) {
				t.Errorf("ActionTestSummaryGroup.testsWithStatus() gotTarget = %v, want %v", gotGroups, tt.wantGroups)
			}
		})
	}
}

func TestActionTestPlanRunSummaries_tests(t *testing.T) {
	tests := []struct {
		name      string
		summaries ActionTestPlanRunSummaries
		want      map[string][]ActionTestSummaryGroup
	}{
		{
			name: "single test with status",
			summaries: ActionTestPlanRunSummaries{
				Summaries: Summaries{
					Values: []Summary{
						Summary{
							TestableSummaries: TestableSummaries{
								Values: []ActionTestableSummary{
									ActionTestableSummary{
										Name: Name{Value: "test case 1"},
										Tests: Tests{
											Values: []ActionTestSummaryGroup{
												ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[string][]ActionTestSummaryGroup{
				"test case 1": []ActionTestSummaryGroup{
					ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}},
				},
			},
		},
		{
			name: "single test with status + subtests with status",
			summaries: ActionTestPlanRunSummaries{
				Summaries: Summaries{
					Values: []Summary{
						Summary{
							TestableSummaries: TestableSummaries{
								Values: []ActionTestableSummary{
									ActionTestableSummary{
										Name: Name{Value: "test case 1"},
										Tests: Tests{
											Values: []ActionTestSummaryGroup{
												ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}},
											},
										},
									},
									ActionTestableSummary{
										Name: Name{Value: "test case 2"},
										Tests: Tests{
											Values: []ActionTestSummaryGroup{
												ActionTestSummaryGroup{
													Subtests: Subtests{
														Values: []ActionTestSummaryGroup{
															ActionTestSummaryGroup{
																TestStatus: TestStatus{Value: "success"},
															},
															ActionTestSummaryGroup{
																TestStatus: TestStatus{Value: "success"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[string][]ActionTestSummaryGroup{
				"test case 1": []ActionTestSummaryGroup{
					ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}},
				},
				"test case 2": []ActionTestSummaryGroup{
					ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}},
					ActionTestSummaryGroup{TestStatus: TestStatus{Value: "success"}},
				},
			},
		},
		{
			name:      "no test with status",
			summaries: ActionTestPlanRunSummaries{},
			want:      map[string][]ActionTestSummaryGroup{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summaries.tests(); !reflect.DeepEqual(got, tt.want) {
				fmt.Println("want: ", pretty.Object(tt.want))
				fmt.Println("got: ", pretty.Object(got))
				t.Errorf("ActionTestPlanRunSummaries.tests() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionTestPlanRunSummaries_failuresCount(t *testing.T) {
	tests := []struct {
		name                string
		summaries           ActionTestPlanRunSummaries
		testableSummaryName string
		wantFailure         int
	}{
		{
			name: "single failure",
			summaries: ActionTestPlanRunSummaries{
				Summaries: Summaries{
					Values: []Summary{
						Summary{
							TestableSummaries: TestableSummaries{
								Values: []ActionTestableSummary{
									ActionTestableSummary{
										Name: Name{Value: "test case"},
										Tests: Tests{
											Values: []ActionTestSummaryGroup{
												ActionTestSummaryGroup{TestStatus: TestStatus{Value: "Failure"}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			testableSummaryName: "test case",
			wantFailure:         1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFailure := tt.summaries.failuresCount(tt.testableSummaryName); gotFailure != tt.wantFailure {
				t.Errorf("ActionTestPlanRunSummaries.failuresCount() = %v, want %v", gotFailure, tt.wantFailure)
			}
		})
	}
}

func TestActionTestPlanRunSummaries_totalTime(t *testing.T) {
	tests := []struct {
		name                string
		summaries           ActionTestPlanRunSummaries
		testableSummaryName string
		wantTime            float64
	}{
		{
			name: "single test",
			summaries: ActionTestPlanRunSummaries{
				Summaries: Summaries{
					Values: []Summary{
						Summary{
							TestableSummaries: TestableSummaries{
								Values: []ActionTestableSummary{
									ActionTestableSummary{
										Name: Name{Value: "test case"},
										Tests: Tests{
											Values: []ActionTestSummaryGroup{
												ActionTestSummaryGroup{
													Duration:   Duration{Value: "10"},
													TestStatus: TestStatus{Value: "Failure"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			testableSummaryName: "test case",
			wantTime:            10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotTime := tt.summaries.totalTime(tt.testableSummaryName); gotTime != tt.wantTime {
				t.Errorf("ActionTestPlanRunSummaries.totalTime() = %v, want %v", gotTime, tt.wantTime)
			}
		})
	}
}

func TestTestFailureSummary_fileAndLineNumber(t *testing.T) {
	tests := []struct {
		name     string
		summary  TestFailureSummary
		wantFile string
		wantLine string
	}{
		{
			name: "",
			summary: TestFailureSummary{
				DocumentLocationInCreatingWorkspace: DocumentLocationInCreatingWorkspace{
					URL: URL{Value: "file:/Xcode11TestUITests2.swift#CharacterRangeLen=0&EndingLineNumber=33&StartingLineNumber=33"},
				},
			},
			wantFile: "file:/Xcode11TestUITests2.swift",
			wantLine: "CharacterRangeLen=0&EndingLineNumber=33&StartingLineNumber=33",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFile, gotLine := tt.summary.fileAndLineNumber()
			if gotFile != tt.wantFile {
				t.Errorf("TestFailureSummary.fileAndLineNumber() gotFile = %v, want %v", gotFile, tt.wantFile)
			}
			if gotLine != tt.wantLine {
				t.Errorf("TestFailureSummary.fileAndLineNumber() gotLine = %v, want %v", gotLine, tt.wantLine)
			}
		})
	}
}

func TestActionsInvocationRecord_failure(t *testing.T) {
	tests := []struct {
		name   string
		record ActionsInvocationRecord
		test   ActionTestSummaryGroup
		want   string
	}{
		{
			name: "Simple test",
			record: ActionsInvocationRecord{
				Issues: Issues{
					TestFailureSummaries: TestFailureSummaries{
						Values: []TestFailureSummary{
							TestFailureSummary{
								ProducingTarget: ProducingTarget{Value: "Xcode11TestUITests2"},
								TestCaseName:    TestCaseName{Value: "Xcode11TestUITests2.testFail()"},
								Message:         Message{Value: "XCTAssertEqual failed: (\"1\") is not equal to (\"0\")"},
								DocumentLocationInCreatingWorkspace: DocumentLocationInCreatingWorkspace{
									URL: URL{Value: "file:/Xcode11TestUITests2.swift#CharacterRangeLen=0&EndingLineNumber=33&StartingLineNumber=33"},
								},
							},
						},
					},
				},
			},
			test: ActionTestSummaryGroup{
				Identifier: Identifier{Value: "Xcode11TestUITests2/testFail()"},
			},
			want: `file:/Xcode11TestUITests2.swift:CharacterRangeLen=0&EndingLineNumber=33&StartingLineNumber=33 - XCTAssertEqual failed: ("1") is not equal to ("0")`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.record.failure(tt.test); got != tt.want {
				t.Errorf("ActionsInvocationRecord.failure() = %v, want %v", got, tt.want)
			}
		})
	}
}
