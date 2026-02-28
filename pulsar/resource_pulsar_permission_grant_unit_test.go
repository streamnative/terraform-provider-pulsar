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
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetPermissionLock_SameKeyReturnsSamePointer verifies that repeated calls
// with the same key return the identical *sync.Mutex.
func TestGetPermissionLock_SameKeyReturnsSamePointer(t *testing.T) {
	key := t.Name()
	mu1 := getPermissionLock(key)
	mu2 := getPermissionLock(key)
	assert.Same(t, mu1, mu2)
}

// TestGetPermissionLock_DifferentKeysReturnDifferentPointers verifies that
// distinct namespace/topic strings yield independent mutexes.
func TestGetPermissionLock_DifferentKeysReturnDifferentPointers(t *testing.T) {
	mu1 := getPermissionLock(t.Name() + "/ns1")
	mu2 := getPermissionLock(t.Name() + "/ns2")
	assert.NotSame(t, mu1, mu2)
}

// TestGetPermissionLock_SerializesConcurrentAccess verifies that goroutines
// sharing the same key are mutually exclusive inside the critical section.
// Run with -race to additionally detect data races.
func TestGetPermissionLock_SerializesConcurrentAccess(t *testing.T) {
	const workers = 20
	key := t.Name()

	var (
		active     int64
		violations int64
		wg         sync.WaitGroup
	)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			mu := getPermissionLock(key)
			mu.Lock()
			defer mu.Unlock()

			if atomic.AddInt64(&active, 1) > 1 {
				atomic.AddInt64(&violations, 1)
			}
			time.Sleep(time.Millisecond) // hold lock briefly to maximise contention
			atomic.AddInt64(&active, -1)
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(0), violations,
		"at most one goroutine must hold the lock at a time")
}

func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "409 conflict",
			err:  rest.Error{Code: http.StatusConflict},
			want: true,
		},
		{
			name: "404 not found",
			err:  rest.Error{Code: http.StatusNotFound},
			want: false,
		},
		{
			name: "500 internal server error",
			err:  rest.Error{Code: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "wrapped 409",
			err:  fmt.Errorf("grant failed: %w", rest.Error{Code: http.StatusConflict}),
			want: true,
		},
		{
			name: "plain error with conflict text",
			err:  errors.New("concurrent modification"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isConflictError(tt.err))
		})
	}
}

func TestRetryOnConflict_SucceedsImmediately(t *testing.T) {
	calls := 0
	err := retryOnConflict(context.Background(), func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryOnConflict_NonConflictErrorNotRetried(t *testing.T) {
	calls := 0
	sentinel := errors.New("unexpected error")
	err := retryOnConflict(context.Background(), func() error {
		calls++
		return sentinel
	})
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, calls, "non-409 error must not trigger a retry")
}

// TestRetryOnConflict_ConflictErrorIsRetried simulates one 409 followed by
// success and verifies the operation is called twice in total.
// Note: this test takes ~500ms due to the backoff initial interval.
func TestRetryOnConflict_ConflictErrorIsRetried(t *testing.T) {
	calls := 0
	err := retryOnConflict(context.Background(), func() error {
		calls++
		if calls == 1 {
			return rest.Error{Code: http.StatusConflict}
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "operation must be retried after a 409 response")
}

// TestRetryOnConflict_CancelledContextStopsRetry verifies that cancelling the
// context stops the retry loop promptly instead of waiting the full backoff.
func TestRetryOnConflict_CancelledContextStopsRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	cancel() // cancel immediately
	err := retryOnConflict(ctx, func() error {
		calls++
		return rest.Error{Code: http.StatusConflict}
	})
	// The first call executes, then the backoff detects the cancelled context and stops.
	assert.Equal(t, 1, calls, "retry loop must stop promptly on cancelled context")
	assert.Error(t, err)
}
