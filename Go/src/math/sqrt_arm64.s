// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

// func archSqrt(x float64) float64
TEXT ·archSqrt(SB),NOSPLIT,$0
	FMOVD	x+0(FP), F0
	FSQRTD	F0, F0
	FMOVD	F0, ret+8(FP)
	RET
