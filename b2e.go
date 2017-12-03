package main

/*
#cgo LDFLAGS: -lquadmath
#cgo CFLAGS: -O3
//#cgo LDFLAGS: -lm
#include <stdlib.h>
typedef unsigned long long UINT64;
extern void myb2e(UINT64 w1,UINT64 w2,UINT64 bef,UINT64* o_keta,UINT64* o_res);
*/
import "C"

func B2E(w1, w2, bef uint64) (keta, res uint64) {
	var o_keta C.UINT64
	var o_res C.UINT64
	C.myb2e(C.UINT64(w1), C.UINT64(w2), C.UINT64(bef), &o_keta, &o_res)
	return uint64(o_keta), uint64(o_res)
}
