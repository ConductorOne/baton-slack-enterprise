
#line 1 "cgo-builtin-prolog"
#include <stddef.h>

/* Define intgo when compiling with GCC.  */
typedef ptrdiff_t intgo;

#define GO_CGO_GOSTRING_TYPEDEF
typedef struct { const char *p; intgo n; } _GoString_;
typedef struct { char *p; intgo n; intgo c; } _GoBytes_;
_GoString_ GoString(char *p);
_GoString_ GoStringN(char *p, int l);
_GoBytes_ GoBytes(void *p, int n);
char *CString(_GoString_);
void *CBytes(_GoBytes_);
void *_CMalloc(size_t);

__attribute__ ((unused))
static size_t _GoStringLen(_GoString_ s) { return (size_t)s.n; }

__attribute__ ((unused))
static const char *_GoStringPtr(_GoString_ s) { return s.p; }

#line 3 "/data/squire/src/worktrees/CXH-2390-cf-2610289sdi/CXH-2390-cf-2610289sdi.12.1/vendor/github.com/DataDog/zstd/zstd_stream.go"

#include "zstd.h"

typedef struct compressStream2_result_s {
	size_t return_code;
	size_t bytes_consumed;
	size_t bytes_written;
} compressStream2_result;

static void ZSTD_compressStream2_wrapper(compressStream2_result* result, ZSTD_CCtx* ctx,
		void* dst, size_t maxDstSize, const void* src, size_t srcSize) {
	ZSTD_outBuffer outBuffer = { dst, maxDstSize, 0 };
	ZSTD_inBuffer inBuffer = { src, srcSize, 0 };
	size_t retCode = ZSTD_compressStream2(ctx, &outBuffer, &inBuffer, ZSTD_e_continue);

	result->return_code = retCode;
	result->bytes_consumed = inBuffer.pos;
	result->bytes_written = outBuffer.pos;
}

static void ZSTD_compressStream2_flush(compressStream2_result* result, ZSTD_CCtx* ctx,
		void* dst, size_t maxDstSize, const void* src, size_t srcSize) {
	ZSTD_outBuffer outBuffer = { dst, maxDstSize, 0 };
	ZSTD_inBuffer inBuffer = { src, srcSize, 0 };
	size_t retCode = ZSTD_compressStream2(ctx, &outBuffer, &inBuffer, ZSTD_e_flush);

	result->return_code = retCode;
	result->bytes_consumed = inBuffer.pos;
	result->bytes_written = outBuffer.pos;
}

static void ZSTD_compressStream2_finish(compressStream2_result* result, ZSTD_CCtx* ctx,
		void* dst, size_t maxDstSize, const void* src, size_t srcSize) {
	ZSTD_outBuffer outBuffer = { dst, maxDstSize, 0 };
	ZSTD_inBuffer inBuffer = { src, srcSize, 0 };
	size_t retCode = ZSTD_compressStream2(ctx, &outBuffer, &inBuffer, ZSTD_e_end);

	result->return_code = retCode;
	result->bytes_consumed = inBuffer.pos;
	result->bytes_written = outBuffer.pos;
}

// decompressStream2_result is the same as compressStream2_result, but keep 2 separate struct for easier changes
typedef struct decompressStream2_result_s {
	size_t return_code;
	size_t bytes_consumed;
	size_t bytes_written;
} decompressStream2_result;

static void ZSTD_decompressStream_wrapper(decompressStream2_result* result, ZSTD_DCtx* ctx,
		void* dst, size_t maxDstSize, const void* src, size_t srcSize) {
	ZSTD_outBuffer outBuffer = { dst, maxDstSize, 0 };
	ZSTD_inBuffer inBuffer = { src, srcSize, 0 };
	size_t retCode = ZSTD_decompressStream(ctx, &outBuffer, &inBuffer);

	result->return_code = retCode;
	result->bytes_consumed = inBuffer.pos;
	result->bytes_written = outBuffer.pos;
}

#line 1 "cgo-generated-wrapper"


#line 1 "cgo-gcc-prolog"
/*
  If x and y are not equal, the type will be invalid
  (have a negative array count) and an inscrutable error will come
  out of the compiler and hopefully mention "name".
*/
#define __cgo_compile_assert_eq(x, y, name) typedef char name[(x-y)*(x-y)*-2UL+1UL];

/* Check at compile time that the sizes we use match our expectations. */
#define __cgo_size_assert(t, n) __cgo_compile_assert_eq(sizeof(t), (size_t)n, _cgo_sizeof_##t##_is_not_##n)

__cgo_size_assert(char, 1)
__cgo_size_assert(short, 2)
__cgo_size_assert(int, 4)
typedef long long __cgo_long_long;
__cgo_size_assert(__cgo_long_long, 8)
__cgo_size_assert(float, 4)
__cgo_size_assert(double, 8)

extern char* _cgo_topofstack(void);

/*
  We use packed structs, but they are always aligned.
  The pragmas and address-of-packed-member are only recognized as warning
  groups in clang 4.0+, so ignore unknown pragmas first.
*/
#pragma GCC diagnostic ignored "-Wunknown-pragmas"
#pragma GCC diagnostic ignored "-Wpragmas"
#pragma GCC diagnostic ignored "-Waddress-of-packed-member"
#pragma GCC diagnostic ignored "-Wunknown-warning-option"
#pragma GCC diagnostic ignored "-Wunaligned-access"

#include <errno.h>
#include <string.h>


#define CGO_NO_SANITIZE_THREAD
#define _cgo_tsan_acquire()
#define _cgo_tsan_release()


#define _cgo_msan_write(addr, sz)

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_CCtx_loadDictionary(void *v)
{
	struct {
		ZSTD_CCtx* p0;
		const void* p1;
		size_t p2;
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_CCtx_loadDictionary(_cgo_a->p0, _cgo_a->p1, _cgo_a->p2);
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_CCtx_setParameter(void *v)
{
	struct {
		ZSTD_CCtx* p0;
		ZSTD_cParameter p1;
		int p2;
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_CCtx_setParameter(_cgo_a->p0, _cgo_a->p1, _cgo_a->p2);
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_DCtx_loadDictionary(void *v)
{
	struct {
		ZSTD_DCtx* p0;
		const void* p1;
		size_t p2;
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_DCtx_loadDictionary(_cgo_a->p0, _cgo_a->p1, _cgo_a->p2);
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_DCtx_reset(void *v)
{
	struct {
		ZSTD_DCtx* p0;
		ZSTD_ResetDirective p1;
		char __pad12[4];
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_DCtx_reset(_cgo_a->p0, _cgo_a->p1);
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_DStreamInSize(void *v)
{
	struct {
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_DStreamInSize();
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_DStreamOutSize(void *v)
{
	struct {
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_DStreamOutSize();
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_compressStream2_finish(void *v)
{
	struct {
		compressStream2_result* p0;
		ZSTD_CCtx* p1;
		void* p2;
		size_t p3;
		const void* p4;
		size_t p5;
	} __attribute__((__packed__)) *_cgo_a = v;
	_cgo_tsan_acquire();
	ZSTD_compressStream2_finish(_cgo_a->p0, _cgo_a->p1, _cgo_a->p2, _cgo_a->p3, _cgo_a->p4, _cgo_a->p5);
	_cgo_tsan_release();
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_compressStream2_flush(void *v)
{
	struct {
		compressStream2_result* p0;
		ZSTD_CCtx* p1;
		void* p2;
		size_t p3;
		const void* p4;
		size_t p5;
	} __attribute__((__packed__)) *_cgo_a = v;
	_cgo_tsan_acquire();
	ZSTD_compressStream2_flush(_cgo_a->p0, _cgo_a->p1, _cgo_a->p2, _cgo_a->p3, _cgo_a->p4, _cgo_a->p5);
	_cgo_tsan_release();
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_compressStream2_wrapper(void *v)
{
	struct {
		compressStream2_result* p0;
		ZSTD_CCtx* p1;
		void* p2;
		size_t p3;
		const void* p4;
		size_t p5;
	} __attribute__((__packed__)) *_cgo_a = v;
	_cgo_tsan_acquire();
	ZSTD_compressStream2_wrapper(_cgo_a->p0, _cgo_a->p1, _cgo_a->p2, _cgo_a->p3, _cgo_a->p4, _cgo_a->p5);
	_cgo_tsan_release();
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_createCStream(void *v)
{
	struct {
		ZSTD_CStream* r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = (__typeof__(_cgo_a->r)) ZSTD_createCStream();
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_createDStream(void *v)
{
	struct {
		ZSTD_DStream* r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = (__typeof__(_cgo_a->r)) ZSTD_createDStream();
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_decompressStream_wrapper(void *v)
{
	struct {
		decompressStream2_result* p0;
		ZSTD_DCtx* p1;
		void* p2;
		size_t p3;
		const void* p4;
		size_t p5;
	} __attribute__((__packed__)) *_cgo_a = v;
	_cgo_tsan_acquire();
	ZSTD_decompressStream_wrapper(_cgo_a->p0, _cgo_a->p1, _cgo_a->p2, _cgo_a->p3, _cgo_a->p4, _cgo_a->p5);
	_cgo_tsan_release();
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_freeCStream(void *v)
{
	struct {
		ZSTD_CStream* p0;
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_freeCStream(_cgo_a->p0);
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_freeDStream(void *v)
{
	struct {
		ZSTD_DStream* p0;
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_freeDStream(_cgo_a->p0);
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

CGO_NO_SANITIZE_THREAD
void
_cgo_833553befd65_Cfunc_ZSTD_initDStream(void *v)
{
	struct {
		ZSTD_DStream* p0;
		size_t r;
	} __attribute__((__packed__)) *_cgo_a = v;
	char *_cgo_stktop = _cgo_topofstack();
	__typeof__(_cgo_a->r) _cgo_r;
	_cgo_tsan_acquire();
	_cgo_r = ZSTD_initDStream(_cgo_a->p0);
	_cgo_tsan_release();
	_cgo_a = (void*)((char*)_cgo_a + (_cgo_topofstack() - _cgo_stktop));
	_cgo_a->r = _cgo_r;
	_cgo_msan_write(&_cgo_a->r, sizeof(_cgo_a->r));
}

