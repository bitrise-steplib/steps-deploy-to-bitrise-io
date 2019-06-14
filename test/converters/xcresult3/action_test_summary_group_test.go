package xcresult3

import (
	"reflect"
	"testing"
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
