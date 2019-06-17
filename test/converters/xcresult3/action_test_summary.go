package xcresult3

// Attachment ...
type Attachment struct {
	Filename struct {
		Value string `json:"_value"`
	} `json:"filename"`

	PayloadRef struct {
		ID struct {
			Value string `json:"_value"`
		}
	} `json:"payloadRef"`
}

// Attachments ...
type Attachments struct {
	Values []Attachment `json:"_values"`
}

// ActionTestActivitySummary ...
type ActionTestActivitySummary struct {
	Attachments Attachments `json:"attachments"`
}

// ActivitySummaries ...
type ActivitySummaries struct {
	Values []ActionTestActivitySummary `json:"_values"`
}

// ActionTestSummary ...
type ActionTestSummary struct {
	ActivitySummaries ActivitySummaries `json:"activitySummaries"`
}
