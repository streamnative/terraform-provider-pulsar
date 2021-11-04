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

package bytesize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormBytes(t *testing.T) {
	testcases := make(map[uint64]uint64)
	testcases[1] = 1 * 1024 * 1024
	testcases[2] = 2 * 1024 * 1024
	testcases[10] = 10 * 1024 * 1024
	testcases[20] = 20 * 1024 * 1024

	for mb, bytes := range testcases {
		b := FormBytes(bytes)
		assert.Equal(t, bytes, b.ToBytes())
		assert.Equal(t, mb, b.ToMegaBytes())
	}
}

func TestFormMegaBytes(t *testing.T) {
	testcases := make(map[uint64]uint64)
	testcases[1] = 1 * 1024 * 1024
	testcases[2] = 2 * 1024 * 1024
	testcases[10] = 10 * 1024 * 1024
	testcases[20] = 20 * 1024 * 1024

	for mb, bytes := range testcases {
		b := FormMegaBytes(mb)
		assert.Equal(t, bytes, b.ToBytes())
		assert.Equal(t, mb, b.ToMegaBytes())
	}
}
