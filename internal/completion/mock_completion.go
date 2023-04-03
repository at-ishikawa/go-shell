// Code generated by MockGen. DO NOT EDIT.
// Source: ./completion.go

// Package completion is a generated GoMock package.
package completion

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCompletion is a mock of Completion interface.
type MockCompletion struct {
	ctrl     *gomock.Controller
	recorder *MockCompletionMockRecorder
}

// MockCompletionMockRecorder is the mock recorder for MockCompletion.
type MockCompletionMockRecorder struct {
	mock *MockCompletion
}

// NewMockCompletion creates a new mock instance.
func NewMockCompletion(ctrl *gomock.Controller) *MockCompletion {
	mock := &MockCompletion{ctrl: ctrl}
	mock.recorder = &MockCompletionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCompletion) EXPECT() *MockCompletionMockRecorder {
	return m.recorder
}

// Complete mocks base method.
func (m *MockCompletion) Complete(rows []string, options CompleteOptions) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Complete", rows, options)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Complete indicates an expected call of Complete.
func (mr *MockCompletionMockRecorder) Complete(rows, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Complete", reflect.TypeOf((*MockCompletion)(nil).Complete), rows, options)
}

// CompleteMulti mocks base method.
func (m *MockCompletion) CompleteMulti(rows []string, options CompleteOptions) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CompleteMulti", rows, options)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CompleteMulti indicates an expected call of CompleteMulti.
func (mr *MockCompletionMockRecorder) CompleteMulti(rows, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompleteMulti", reflect.TypeOf((*MockCompletion)(nil).CompleteMulti), rows, options)
}
