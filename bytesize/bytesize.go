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

const MB = 1 << (10 * 2)

type byteSize struct {
	value uint64
}

func newByteSize(n uint64) *byteSize {
	return &byteSize{value: n}
}

func FormBytes(n uint64) *byteSize {
	return newByteSize(n)
}

func FormMegaBytes(n uint64) *byteSize {
	return newByteSize(n * MB)
}

func (b *byteSize) ToBytes() uint64 {
	return b.value
}

func (b *byteSize) ToMegaBytes() uint64 {
	return b.value / MB
}
