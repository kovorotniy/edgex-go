//
// Copyright (c) 2019 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing permissions and limitations under
// the License.
//
// SPDX-License-Identifier: Apache-2.0'
//

package option

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
)

func TestProcessOptionNormal(t *testing.T) {
	testExecutor := &mockOptionsExecutor{}
	// normal case
	testExecutor.On("executeOptions", getMockArguments()...).
		Return(normal, nil).Once()

	assert := assert.New(t)

	options := PkiInitOption{}
	optsExecutor, _, _ := NewPkiInitOption(options)
	optsExecutor.(*PkiInitOption).executor = testExecutor
	exitCode, err := optsExecutor.ProcessOptions()
	assert.Equal(normal.intValue(), exitCode)
	assert.Nil(err)

	testExecutor.AssertExpectations(t)
}

func TestProcessOptionError(t *testing.T) {
	testExecutor := &mockOptionsExecutor{}
	generateErr := errors.New("failed to execute generate")
	// error case
	testExecutor.On("executeOptions", getMockArguments()...).
		Return(exitWithError, generateErr).Once()

	assert := assert.New(t)

	options := PkiInitOption{}
	generateOn, _, _ := NewPkiInitOption(options)
	generateOn.(*PkiInitOption).executor = testExecutor
	exitCode, err := generateOn.ProcessOptions()
	assert.Equal(exitWithError.intValue(), exitCode)
	assert.Equal(generateErr, err)

	testExecutor.AssertExpectations(t)
}

func TestExecuteOption(t *testing.T) {
	testExecutor := &mockOptionsExecutor{}
	assert := assert.New(t)

	options := PkiInitOption{}
	anyOpt, _, _ := NewPkiInitOption(options)
	anyOpt.(*PkiInitOption).executor = testExecutor
	exitCode, err := anyOpt.executeOptions(mockOption())
	assert.Equal(normal, exitCode)
	assert.Nil(err)
}

func mockOption() func(*PkiInitOption) (exitCode, error) {
	return func(pkiInitOpton *PkiInitOption) (exitCode, error) {
		return normal, nil
	}
}

func getMockArguments() []interface{} {
	var ifc []interface{}
	// use reflection to find out how many bool type of options in PkiInitOption struct
	// and create a slice of mock argument interface instances
	elm := reflect.ValueOf(&PkiInitOption{}).Elem()
	for i := 0; i < elm.NumField(); i++ {
		field := elm.Field(i)
		switch field.Kind() {
		case reflect.Bool:
			ifc = append(ifc, mock.AnythingOfTypeArgument("func(*option.PkiInitOption) (option.exitCode, error)"))
		}
	}

	return ifc
}
