package mocks

import "github.com/stretchr/testify/mock"

type PathChecker struct {
	mock.Mock
}

func (_m *PathChecker) IsPathExists(pth string) (bool, error) {
	args := _m.Called(pth)
	var err error
	if len(args) > 1 {
		err = args.Error(1)
	}
	return args.Bool(0), err
}

func (_m *PathChecker) IsDirExists(pth string) (bool, error) {
	args := _m.Called(pth)
	var err error
	if len(args) > 1 {
		err = args.Error(1)
	}
	return args.Bool(0), err
}
