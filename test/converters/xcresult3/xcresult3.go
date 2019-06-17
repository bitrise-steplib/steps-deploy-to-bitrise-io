package xcresult3

// Parse parses the given xcresult file's ActionsInvocationRecord and the list of ActionTestPlanRunSummaries.
func Parse(pth string) (*ActionsInvocationRecord, []ActionTestPlanRunSummaries, error) {
	var r ActionsInvocationRecord
	if err := xcresulttoolGet(pth, "", &r); err != nil {
		return nil, nil, err
	}

	var summaries []ActionTestPlanRunSummaries
	for _, action := range r.Actions.Values {
		refID := action.ActionResult.TestsRef.ID.Value
		var s ActionTestPlanRunSummaries
		if err := xcresulttoolGet(pth, refID, &s); err != nil {
			return nil, nil, err
		}
		summaries = append(summaries, s)
	}
	return &r, summaries, nil
}
