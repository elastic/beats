// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Run 'go generate' to create mocks that are used in tests.
//go:generate go run go.uber.org/mock/mockgen -source=logout.go -destination=mock_logout.go -package client -mock_names=Logouter=MockLogouter

func TestLogout(t *testing.T) {
	tests := []struct {
		name                 string
		mockClient           func(*MockLogouter)
		expectedErrorMessage string
	}{
		{
			name: "Logout success",
			mockClient: func(clientMock *MockLogouter) {
				clientMock.EXPECT().Logout(gomock.Any()).Return(nil)
			},
			expectedErrorMessage: "",
		},
		{
			name: "Logout fails once",
			mockClient: func(clientMock *MockLogouter) {
				clientMock.EXPECT().
					Logout(gomock.Any()).
					Return(errors.New("logout failed")).
					Return(nil)

			},
			expectedErrorMessage: "",
		},
		{
			name: "Logout fails 4 times (original try + 3 retries)",
			mockClient: func(clientMock *MockLogouter) {
				clientMock.EXPECT().
					Logout(gomock.Any()).
					Return(errors.New("logout failed")).
					Times(4)
			},
			expectedErrorMessage: "logout failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			clientMock := NewMockLogouter(ctrl)

			tt.mockClient(clientMock)

			ctx := context.Background()
			err := Logout(ctx, clientMock)

			if tt.expectedErrorMessage == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err, tt.expectedErrorMessage)
			}
		})
	}
}
