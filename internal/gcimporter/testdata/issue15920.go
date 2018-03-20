// Copyright 2016 The Go Authors. All rights reserved.
// Modified work copyright 2018 Alex Browne. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p

// The underlying type of Error is the underlying type of error.
// Make sure we can import this again without problems.
type Error error

func F() Error { return nil }