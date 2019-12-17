package converters

import (
	"reflect"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
)

func equalTestCase(a, b junit.TestCase) bool {
	return reflect.DeepEqual(a, b)
}

func equivalentTestCases(a, b []junit.TestCase) bool {
	if len(a) != len(b) {
		return false
	}

	bCopy := append([]junit.TestCase{}, b...)
	for _, caseA := range a {
		found := false
		for j, caseB := range bCopy {
			if equalTestCase(caseA, caseB) {
				found = true
				// remove already found elements
				copy(bCopy[j:], bCopy[j+1:])
				bCopy = bCopy[:len(bCopy)-1]
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func equivalentTestSuite(a, b junit.TestSuite) bool {
	// Clean TestCases to let reflect.DeepEqual work as expected,
	// compare TestCases later.
	aTestCases := a.TestCases
	a.TestCases = nil

	bTestCases := b.TestCases
	b.TestCases = nil

	if !reflect.DeepEqual(a, b) {
		return false
	}

	return equivalentTestCases(aTestCases, bTestCases)
}

func equivalentTestSuites(a, b []junit.TestSuite) bool {
	if len(a) != len(b) {
		return false
	}

	bCopy := append([]junit.TestSuite{}, b...)
	for _, suitA := range a {
		found := false
		for j, suitB := range bCopy {
			if equivalentTestSuite(suitA, suitB) {
				found = true
				// remove already found elements
				copy(bCopy[j:], bCopy[j+1:])
				bCopy = bCopy[:len(bCopy)-1]
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func equivalentJunitXML(a, b junit.XML) bool {
	aTestSuits := a.TestSuites
	a.TestSuites = nil

	bTestSuits := b.TestSuites
	b.TestSuites = nil

	if !reflect.DeepEqual(a, b) {
		return false
	}

	return equivalentTestSuites(aTestSuits, bTestSuits)
}
