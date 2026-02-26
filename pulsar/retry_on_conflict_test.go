// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pulsar

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
)

func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "409 conflict error",
			err:      rest.Error{Code: http.StatusConflict, Reason: "Concurrent modification"},
			expected: true,
		},
		{
			name:     "404 not found error",
			err:      rest.Error{Code: http.StatusNotFound, Reason: "Not Found"},
			expected: false,
		},
		{
			name:     "500 internal server error",
			err:      rest.Error{Code: http.StatusInternalServerError, Reason: "Internal Server Error"},
			expected: false,
		},
		{
			name:     "non-rest error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
		{
			name:     "wrapped 409 conflict error",
			err:      fmt.Errorf("wrapper: %w", rest.Error{Code: http.StatusConflict, Reason: "Concurrent modification"}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConflictError(tt.err)
			if result != tt.expected {
				t.Errorf("isConflictError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryOnConflict_SuccessOnFirstAttempt(t *testing.T) {
	var callCount int32
	err := retryOnConflict(func() error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryOnConflict_RetriesOnConflict(t *testing.T) {
	var callCount int32
	err := retryOnConflict(func() error {
		count := atomic.AddInt32(&callCount, 1)
		if count < 3 {
			return rest.Error{Code: http.StatusConflict, Reason: "Concurrent modification"}
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if atomic.LoadInt32(&callCount) < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
}

func TestRetryOnConflict_NoRetryOnOtherErrors(t *testing.T) {
	var callCount int32
	expectedErr := rest.Error{Code: http.StatusNotFound, Reason: "Not Found"}

	err := retryOnConflict(func() error {
		atomic.AddInt32(&callCount, 1)
		return expectedErr
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestRetryOnConflict_NoRetryOnNonRestErrors(t *testing.T) {
	var callCount int32
	expectedErr := fmt.Errorf("some non-rest error")

	err := retryOnConflict(func() error {
		atomic.AddInt32(&callCount, 1)
		return expectedErr
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}
