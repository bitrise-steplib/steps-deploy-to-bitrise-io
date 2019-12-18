package junit

import (
	"encoding/xml"
	"testing"
)

func TestTestSuite_Equal(t *testing.T) {
	tests := []struct {
		name  string
		suitA TestSuite
		suitB TestSuite
		want  bool
	}{
		{
			name:  "empty test cases",
			suitA: TestSuite{},
			suitB: TestSuite{},
			want:  true,
		},
		{
			name: "compare to empty",
			suitA: TestSuite{
				XMLName:  xml.Name{Space: "space", Local: "local"},
				Name:     "name",
				Tests:    0,
				Failures: 0,
				Errors:   0,
				Time:     0.0,
				TestCases: []TestCase{
					TestCase{
						XMLName:   xml.Name{Space: "space", Local: "local"},
						Name:      "name",
						ClassName: "className",
						Time:      0.0,
						Failure:   "",
						Error: &Error{
							XMLName: xml.Name{Space: "space", Local: "local"},
							Message: "message",
							Value:   "value",
						},
						SystemErr: "systemErr",
					},
				},
			},
			suitB: TestSuite{},
			want:  false,
		},
		{
			name: "compare the same",
			suitA: TestSuite{
				XMLName:  xml.Name{Space: "space", Local: "local"},
				Name:     "name",
				Tests:    0,
				Failures: 0,
				Errors:   0,
				Time:     0.0,
				TestCases: []TestCase{
					TestCase{
						XMLName:   xml.Name{Space: "space", Local: "local"},
						Name:      "name",
						ClassName: "className",
						Time:      0.0,
						Failure:   "",
						Error: &Error{
							XMLName: xml.Name{Space: "space", Local: "local"},
							Message: "message",
							Value:   "value",
						},
						SystemErr: "systemErr",
					},
				},
			},
			suitB: TestSuite{
				XMLName:  xml.Name{Space: "space", Local: "local"},
				Name:     "name",
				Tests:    0,
				Failures: 0,
				Errors:   0,
				Time:     0.0,
				TestCases: []TestCase{
					TestCase{
						XMLName:   xml.Name{Space: "space", Local: "local"},
						Name:      "name",
						ClassName: "className",
						Time:      0.0,
						Failure:   "",
						Error: &Error{
							XMLName: xml.Name{Space: "space", Local: "local"},
							Message: "message",
							Value:   "value",
						},
						SystemErr: "systemErr",
					},
				},
			},
			want: true,
		},
		{
			name: "different name",
			suitA: TestSuite{
				Name: "name",
			},
			suitB: TestSuite{
				Name: "different name",
			},
			want: false,
		},
		{
			name: "different case",
			suitA: TestSuite{
				TestCases: []TestCase{
					TestCase{
						Name: "name",
					},
				},
			},
			suitB: TestSuite{
				TestCases: []TestCase{
					TestCase{
						Name: "different name",
					},
				},
			},
			want: false,
		},
		{
			name: "same cases, different order",
			suitA: TestSuite{
				TestCases: []TestCase{
					TestCase{
						Name: "name 1",
					},
					TestCase{
						Name: "name 2",
					},
				},
			},
			suitB: TestSuite{
				TestCases: []TestCase{
					TestCase{
						Name: "name 2",
					},
					TestCase{
						Name: "name 1",
					},
				},
			},
			want: true,
		},
		{
			name: "different amount of cases",
			suitA: TestSuite{
				TestCases: []TestCase{
					TestCase{
						Name: "name 1",
					},
				},
			},
			suitB: TestSuite{
				TestCases: []TestCase{
					TestCase{
						Name: "name 1",
					},
					TestCase{
						Name: "name 1",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.suitA.Equal(tt.suitB); got != tt.want {
				t.Errorf("TestSuite.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
