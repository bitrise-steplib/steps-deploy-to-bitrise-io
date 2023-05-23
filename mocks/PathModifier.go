package mocks

import "github.com/stretchr/testify/mock"

type PathModifier struct {
	mock.Mock
}

func (_m *PathModifier) AbsPath(pth string) (string, error) {
	args := _m.Called(pth)
	var err error
	if len(args) > 1 {
		err = args.Error(1)
	}
	return args.String(0), err
}
