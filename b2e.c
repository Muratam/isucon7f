#include <stdio.h>
#include <math.h>
#include <quadmath.h>
void printf128(__float128 f){
  char c[80];
  quadmath_snprintf(c,80,"%.40Qf",f);
  printf("%s\n",c);
}

/*
typedef unsigned long long UINT64;
void myb2e(UINT64 w1,UINT64 w2,UINT64 bef,UINT64 *o_keta,UINT64 *o_res){
  long double
    fw1 = w1,
    fw2 = w2,
    log10ed = (64 * bef) * logl(2.0L)/ logl(10.0L)
              + log10l(18446744073709551616.0L * fw1 + fw2),
    res = expl(fmodl(log10ed,1.0L) * logl(10.0L)) * 100000000000000.0L;
  *o_keta = (UINT64)log10ed -14;
  *o_res = res + (*o_keta > 10 ? -0.01L : 0.01L);
}
*/

typedef unsigned long long UINT64;
void myb2e(UINT64 w1,UINT64 w2,UINT64 bef,UINT64 *o_keta,UINT64 *o_res){

  __float128
    fw1 = w1,
    fw2 = w2,
    log10ed = (64 * bef) * M_LN2q / M_LN10q
              + log10q(18446744073709551616.0Q * fw1 + fw2),
    res = expq((log10ed - floorq(log10ed)) * M_LN10q) * 100000000000000.0Q;
  *o_keta = (UINT64)log10ed - 14;
  *o_res = res + 0.000000000000001Q;
}

