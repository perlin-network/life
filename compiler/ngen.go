package compiler

import (
	"fmt"
	"strings"
)

const NGEN_FUNCTION_PREFIX = "wasm_function_"
const NGEN_LOCAL_PREFIX = "l"
const NGEN_VALUE_PREFIX = "v"
const NGEN_INS_LABEL_PREFIX = "ins"
const NGEN_ENV_API_PREFIX = "wenv_"
const NGEN_HEADER = `
//static const uint64_t UINT32_MASK = 0xffffffffull;
struct VirtualMachine;
typedef uint64_t (*ExternalFunction)(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params);
struct VirtualMachine {
	void (*throw_s)(struct VirtualMachine *vm, const char *s);
	ExternalFunction (*resolve_import)(struct VirtualMachine *vm, const char *module_name, const char *field_name);
	uint64_t mem_size;
	uint8_t *mem;
	void (*grow_memory)(struct VirtualMachine *vm, uint64_t inc_size);
	void *userdata;
};

#define V_uint32_t vu32
#define V_uint64_t vu64
#define V_int32_t vi32
#define V_int64_t vi64
#define V_float vf32
#define V_double vf64

union Value {
	uint32_t vu32;
	uint64_t vu64;
	int32_t vi32;
	int64_t vi64;
	float vf32;
	double vf64;
};
static uint8_t * __attribute__((always_inline)) mem_translate(struct VirtualMachine *vm, union Value start, uint32_t offset, uint32_t size) {
	start.vu32 += offset;
	#ifndef POLYMERASE_NO_MEM_BOUND_CHECK
	if(start.vu32 + size < start.vu32 || start.vu32 + size > vm->mem_size) vm->throw_s(vm, "memory access out of bounds");
	#endif
	return &vm->mem[start.vu32];
}
static uint64_t __attribute__((always_inline)) clz32(uint32_t x) {
	return __builtin_clz(x);
}
static uint64_t __attribute__((always_inline)) ctz32(uint32_t x) {
	return __builtin_ctz(x);
}
static uint64_t __attribute__((always_inline)) clz64(uint64_t x) {
	return __builtin_clzll(x);
}
static uint64_t __attribute__((always_inline)) ctz64(uint64_t x) {
	return __builtin_ctzll(x);
}
static uint64_t __attribute__((always_inline)) rotl32( uint32_t x, uint32_t r )
{
  return (x << r) | (x >> (32 - r % 32));
}
static uint64_t __attribute__((always_inline)) rotl64( uint64_t x, uint64_t r )
{
  return (x << r) | (x >> (64 - r % 64));
}
static uint64_t __attribute__((always_inline)) rotr32( uint32_t x, uint32_t r )
{
  return (x >> r) | (x << (32 - r % 32));
}
static uint64_t __attribute__((always_inline)) rotr64( uint64_t x, uint64_t r )
{
  return (x >> r) | (x << (64 - r % 64));
}
`
const NGEN_FP_HEADER = `
#include <math.h>

static float __attribute__((always_inline)) fmin32(float a, float b) {
	if(isnan(a) || isnan(b)) return NAN;
	return fminf(a, b);
}

static double __attribute__((always_inline)) fmin64(double a, double b) {
	if(isnan(a) || isnan(b)) return NAN;
	return fmin(a, b);
}

static float __attribute__((always_inline)) fmax32(float a, float b) {
	if(isnan(a) || isnan(b)) return NAN;
	return fmaxf(a, b);
}

static double __attribute__((always_inline)) fmax64(double a, double b) {
	if(isnan(a) || isnan(b)) return NAN;
	return fmax(a, b);
}

static float __attribute__((always_inline)) fneg32(float x) {
	return -x;
}

static double __attribute__((always_inline)) fneg64(double x) {
	return -x;
}

#define fsqrt32 sqrtf
#define fsqrt64 sqrt
#define fceil32 ceilf
#define fceil64 ceil
#define ffloor32 floorf
#define ffloor64 floor
#define ftrunc32 truncf
#define ftrunc64 trunc
#define fnearest32 roundf
#define fnearest64 round
#define fabs32 fabsf
#define fabs64 fabs
#define fcopysign32 copysignf
#define fcopysign64 copysign
`

func bSprintf(builder *strings.Builder, format string, args ...interface{}) {
	builder.WriteString(fmt.Sprintf(format, args...))
}

func writeDivZeroRvCheck(b *strings.Builder, ins Instr) {
	bSprintf(b, "if(%s%d.vu64 == 0) vm->throw_s(vm, \"divide by zero\"); ", NGEN_VALUE_PREFIX, ins.Values[1]) // TODO: fix
}

func writeUnOp_Eqz(b *strings.Builder, ins Instr, ty string) {
	bSprintf(b,
		"%s%d.vu64 = (%s%d.V_%s == 0);",
		NGEN_VALUE_PREFIX, ins.Target,
		NGEN_VALUE_PREFIX, ins.Values[0], ty,
	)
}

func writeUnOp_Fcall(b *strings.Builder, ins Instr, f string, ty string, retTy string) {
	bSprintf(b,
		"%s%d.V_%s = %s(%s%d.V_%s);",
		NGEN_VALUE_PREFIX, ins.Target, retTy,
		f,
		NGEN_VALUE_PREFIX, ins.Values[0], ty,
	)
}

func writeBinOp_Shift(b *strings.Builder, ins Instr, op string, ty string, rounding uint64) {
	bSprintf(b,
		"%s%d.vu64 = (%s%d.V_%s) %s (%s%d.V_%s %% %d);",
		NGEN_VALUE_PREFIX, ins.Target,
		NGEN_VALUE_PREFIX, ins.Values[0], ty,
		op,
		NGEN_VALUE_PREFIX, ins.Values[1], ty,
		rounding,
	)
}

func writeBinOp_Fcall(b *strings.Builder, ins Instr, f string, ty string, retTy string) {
	bSprintf(b,
		"%s%d.V_%s = %s(%s%d.V_%s, %s%d.V_%s);",
		NGEN_VALUE_PREFIX, ins.Target, retTy,
		f,
		NGEN_VALUE_PREFIX, ins.Values[0], ty,
		NGEN_VALUE_PREFIX, ins.Values[1], ty,
	)
}

func writeBinOp_ConstRv(b *strings.Builder, ins Instr, op string, ty string, rv string) {
	bSprintf(b,
		"%s%d.V_%s = (%s%d.V_%s %s (%s));",
		NGEN_VALUE_PREFIX, ins.Target, ty,
		NGEN_VALUE_PREFIX, ins.Values[0], ty,
		op,
		rv,
	)
}

func writeBinOp2(b *strings.Builder, ins Instr, op string, ty string, retTy string) {
	bSprintf(b,
		"%s%d.V_%s = (%s%d.V_%s %s %s%d.V_%s);",
		NGEN_VALUE_PREFIX, ins.Target, retTy,
		NGEN_VALUE_PREFIX, ins.Values[0], ty,
		op,
		NGEN_VALUE_PREFIX, ins.Values[1], ty,
	)
}

func writeBinOp(b *strings.Builder, ins Instr, op string, ty string) {
	writeBinOp2(b, ins, op, ty, ty)
}

func writeMemLoad(b *strings.Builder, ins Instr, ty string) {
	bSprintf(b,
		"%s%d.vi64 = * (%s *) mem_translate(vm, %s%d, %du, sizeof(%s));", // TODO: any missing conversions?
		NGEN_VALUE_PREFIX, ins.Target,
		ty,
		NGEN_VALUE_PREFIX, ins.Values[0],
		uint64(ins.Immediates[1]),
		ty,
	)
}

func writeMemStore(b *strings.Builder, ins Instr, ty string) {
	bSprintf(b,
		"* (%s *) mem_translate(vm, %s%d, %du, sizeof(%s)) = %s%d.vu64;",
		ty,
		NGEN_VALUE_PREFIX, ins.Values[0],
		uint64(ins.Immediates[1]),
		ty,
		NGEN_VALUE_PREFIX, ins.Values[1],
	)
}

func (c *SSAFunctionCompiler) NGen(selfID uint64, numParams uint64, numLocals uint64, numGlobals uint64) string {
	builder := &strings.Builder{}

	bSprintf(builder, "uint64_t %s%d(struct VirtualMachine *vm", NGEN_FUNCTION_PREFIX, selfID)

	for i := uint64(0); i < numParams; i++ {
		bSprintf(builder, ",uint64_t %s%d", NGEN_LOCAL_PREFIX, i)
	}
	builder.WriteString(") {\n")

	for i := uint64(0); i < numLocals; i++ {
		bSprintf(builder, "uint64_t %s%d = 0;\n", NGEN_LOCAL_PREFIX, i+numParams)
	}

	body := &strings.Builder{}
	valueIDs := make(map[TyValueID]struct{})

	for i, ins := range c.Code {
		valueIDs[ins.Target] = struct{}{}

		bSprintf(body, "%s%d: ", NGEN_INS_LABEL_PREFIX, i)
		switch ins.Op {
		case "unreachable":
			bSprintf(body, "vm->throw_s(vm, \"unreachable executed\");")
		case "return":
			if len(ins.Values) == 0 {
				body.WriteString("return 0;")
			} else {
				bSprintf(body, "return %s%d.vu64;", NGEN_VALUE_PREFIX, ins.Values[0])
			}
		case "get_local":
			bSprintf(body,
				"%s%d.vu64 = %s%d;",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_LOCAL_PREFIX, ins.Immediates[0],
			)
		case "set_local":
			bSprintf(body,
				"%s%d = %s%d.vu64;",
				NGEN_LOCAL_PREFIX, ins.Immediates[0],
				NGEN_VALUE_PREFIX, ins.Values[0],
			)
		case "get_global":
			if uint64(ins.Immediates[0]) >= numGlobals {
				panic("global index out of bounds")
			}
			bSprintf(body,
				"%s%d.vu64 = globals[%d];",
				NGEN_VALUE_PREFIX, ins.Target,
				uint64(ins.Immediates[0]),
			)
		case "set_global":
			if uint64(ins.Immediates[0]) >= numGlobals {
				panic("global index out of bounds")
			}
			bSprintf(body,
				"globals[%d] = %s%d.vu64;",
				uint64(ins.Immediates[0]),
				NGEN_VALUE_PREFIX, ins.Values[0],
			)
		case "call":
			bSprintf(body,
				"%s%d.vu64 = %s%d(vm",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_FUNCTION_PREFIX, ins.Immediates[0],
			)
			for _, v := range ins.Values {
				bSprintf(body, ",%s%d.vu64", NGEN_VALUE_PREFIX, v)
			}
			body.WriteString(");")
		case "call_indirect":
			bSprintf(body,
				"%s%d.vu64 = ((uint64_t (*)(struct VirtualMachine *",
				NGEN_VALUE_PREFIX, ins.Target,
			)
			for range ins.Values[:len(ins.Values)-1] {
				bSprintf(body, ",uint64_t")
			}
			bSprintf(body,
				")) %sresolve_indirect(vm, %s%d.vu32, %d)) (vm",
				NGEN_ENV_API_PREFIX,
				NGEN_VALUE_PREFIX, ins.Values[len(ins.Values)-1],
				len(ins.Values)-1,
			)
			for _, v := range ins.Values[:len(ins.Values)-1] {
				bSprintf(body, ",%s%d.vu64", NGEN_VALUE_PREFIX, v)
			}
			body.WriteString(");")
		case "jmp":
			bSprintf(body,
				"phi = %s%d; goto %s%d;",
				NGEN_VALUE_PREFIX, ins.Values[0],
				NGEN_INS_LABEL_PREFIX, ins.Immediates[0],
			)
		case "jmp_if":
			bSprintf(body,
				"if(%s%d.vu32) { phi = %s%d; goto %s%d; }",
				NGEN_VALUE_PREFIX, ins.Values[0],
				NGEN_VALUE_PREFIX, ins.Values[1],
				NGEN_INS_LABEL_PREFIX, ins.Immediates[0],
			)
		case "jmp_either":
			bSprintf(body,
				"phi = %s%d; if(%s%d.vu32) { goto %s%d; } else { goto %s%d; }",
				NGEN_VALUE_PREFIX, ins.Values[1],
				NGEN_VALUE_PREFIX, ins.Values[0],
				NGEN_INS_LABEL_PREFIX, ins.Immediates[0],
				NGEN_INS_LABEL_PREFIX, ins.Immediates[1],
			)
		case "jmp_table":
			bSprintf(body, "phi = %s%d;\n", NGEN_VALUE_PREFIX, ins.Values[1])
			bSprintf(body, "switch(%s%d.vu32) {\n", NGEN_VALUE_PREFIX, ins.Values[0])
			for i, v := range ins.Immediates {
				if i == len(ins.Immediates)-1 {
					bSprintf(body, "default: ")
				} else {
					bSprintf(body, "case %d: ", i)
				}
				bSprintf(body, "goto %s%d;\n", NGEN_INS_LABEL_PREFIX, v)
			}
			bSprintf(body, "}")

		case "phi":
			bSprintf(body,
				"%s%d = phi;",
				NGEN_VALUE_PREFIX, ins.Target,
			)
		case "select":
			bSprintf(body,
				"%s%d = %s%d.vu32 ? %s%d : %s%d;",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_VALUE_PREFIX, ins.Values[2],
				NGEN_VALUE_PREFIX, ins.Values[0],
				NGEN_VALUE_PREFIX, ins.Values[1],
			)
		case "i32.const", "f32.const":
			bSprintf(body,
				"%s%d.vu64 = (uint32_t) (%du);",
				NGEN_VALUE_PREFIX, ins.Target,
				uint32(ins.Immediates[0]),
			)
		case "i32.add":
			writeBinOp(body, ins, "+", "uint32_t")
		case "i32.sub":
			writeBinOp(body, ins, "-", "uint32_t")
		case "i32.mul":
			writeBinOp(body, ins, "*", "uint32_t")
		case "i32.div_s":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "/", "int32_t")
		case "i32.div_u":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "/", "uint32_t")
		case "i32.rem_s":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "%", "int32_t")
		case "i32.rem_u":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "%", "uint32_t")
		case "i32.and":
			writeBinOp(body, ins, "&", "uint32_t")
		case "i32.or":
			writeBinOp(body, ins, "|", "uint32_t")
		case "i32.xor":
			writeBinOp(body, ins, "^", "uint32_t")
		case "i32.shl":
			writeBinOp_Shift(body, ins, "<<", "uint32_t", 32)
		case "i32.shr_s":
			writeBinOp_Shift(body, ins, ">>", "int32_t", 32)
		case "i32.shr_u":
			writeBinOp_Shift(body, ins, ">>", "uint32_t", 32)
		case "i32.rotl":
			writeBinOp_Fcall(body, ins, "rotl32", "uint32_t", "uint64_t")
		case "i32.rotr":
			writeBinOp_Fcall(body, ins, "rotr32", "uint32_t", "uint64_t")
		case "i32.clz":
			writeUnOp_Fcall(body, ins, "clz32", "uint32_t", "uint64_t")
		case "i32.ctz":
			writeUnOp_Fcall(body, ins, "ctz32", "uint32_t", "uint64_t")
		case "i32.popcnt":
			writeUnOp_Fcall(body, ins, "popcnt32", "uint32_t", "uint64_t")
		case "i32.eqz":
			writeUnOp_Eqz(body, ins, "uint32_t")
		case "i32.eq":
			writeBinOp(body, ins, "==", "uint32_t")
		case "i32.ne":
			writeBinOp(body, ins, "!=", "uint32_t")
		case "i32.lt_s":
			writeBinOp(body, ins, "<", "int32_t")
		case "i32.lt_u":
			writeBinOp(body, ins, "<", "uint32_t")
		case "i32.le_s":
			writeBinOp(body, ins, "<=", "int32_t")
		case "i32.le_u":
			writeBinOp(body, ins, "<=", "uint32_t")
		case "i32.gt_s":
			writeBinOp(body, ins, ">", "int32_t")
		case "i32.gt_u":
			writeBinOp(body, ins, ">", "uint32_t")
		case "i32.ge_s":
			writeBinOp(body, ins, ">=", "int32_t")
		case "i32.ge_u":
			writeBinOp(body, ins, ">=", "uint32_t")
		case "i64.const", "f64.const":
			bSprintf(body,
				"%s%d.vu64 = (uint64_t) (%dull);",
				NGEN_VALUE_PREFIX, ins.Target,
				uint64(ins.Immediates[0]),
			)
		case "i64.add":
			writeBinOp(body, ins, "+", "uint64_t")
		case "i64.sub":
			writeBinOp(body, ins, "-", "uint64_t")
		case "i64.mul":
			writeBinOp(body, ins, "*", "uint64_t")
		case "i64.div_s":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "/", "int64_t")
		case "i64.div_u":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "/", "uint64_t")
		case "i64.rem_s":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "%", "int64_t")
		case "i64.rem_u":
			writeDivZeroRvCheck(body, ins)
			writeBinOp(body, ins, "%", "uint64_t")
		case "i64.and":
			writeBinOp(body, ins, "&", "uint64_t")
		case "i64.or":
			writeBinOp(body, ins, "|", "uint64_t")
		case "i64.xor":
			writeBinOp(body, ins, "^", "uint64_t")
		case "i64.shl":
			writeBinOp_Shift(body, ins, "<<", "uint64_t", 64)
		case "i64.shr_s":
			writeBinOp_Shift(body, ins, ">>", "int64_t", 64)
		case "i64.shr_u":
			writeBinOp_Shift(body, ins, ">>", "uint64_t", 64)
		case "i64.rotl":
			writeBinOp_Fcall(body, ins, "rotl64", "uint64_t", "uint64_t")
		case "i64.rotr":
			writeBinOp_Fcall(body, ins, "rotr64", "uint64_t", "uint64_t")
		case "i64.clz":
			writeUnOp_Fcall(body, ins, "clz64", "uint64_t", "uint64_t")
		case "i64.ctz":
			writeUnOp_Fcall(body, ins, "ctz64", "uint64_t", "uint64_t")
		case "i64.popcnt":
			writeUnOp_Fcall(body, ins, "popcnt64", "uint64_t", "uint64_t")
		case "i64.eqz":
			writeUnOp_Eqz(body, ins, "uint64_t")
		case "i64.eq":
			writeBinOp(body, ins, "==", "uint64_t")
		case "i64.ne":
			writeBinOp(body, ins, "!=", "uint64_t")
		case "i64.lt_s":
			writeBinOp(body, ins, "<", "int64_t")
		case "i64.lt_u":
			writeBinOp(body, ins, "<", "uint64_t")
		case "i64.le_s":
			writeBinOp(body, ins, "<=", "int64_t")
		case "i64.le_u":
			writeBinOp(body, ins, "<=", "uint64_t")
		case "i64.gt_s":
			writeBinOp(body, ins, ">", "int64_t")
		case "i64.gt_u":
			writeBinOp(body, ins, ">", "uint64_t")
		case "i64.ge_s":
			writeBinOp(body, ins, ">=", "int64_t")
		case "i64.ge_u":
			writeBinOp(body, ins, ">=", "uint64_t")
		case "f32.add":
			writeBinOp(body, ins, "+", "float")
		case "f32.sub":
			writeBinOp(body, ins, "-", "float")
		case "f32.mul":
			writeBinOp(body, ins, "*", "float")
		case "f32.div":
			writeBinOp(body, ins, "/", "float")
		case "f32.sqrt":
			writeUnOp_Fcall(body, ins, "fsqrt32", "float", "float")
		case "f32.min":
			writeBinOp_Fcall(body, ins, "fmin32", "float", "float")
		case "f32.max":
			writeBinOp_Fcall(body, ins, "fmax32", "float", "float")
		case "f32.ceil":
			writeUnOp_Fcall(body, ins, "fceil32", "float", "float")
		case "f32.floor":
			writeUnOp_Fcall(body, ins, "ffloor32", "float", "float")
		case "f32.trunc":
			writeUnOp_Fcall(body, ins, "ftrunc32", "float", "float")
		case "f32.nearest":
			writeUnOp_Fcall(body, ins, "fnearest32", "float", "float")
		case "f32.abs":
			writeUnOp_Fcall(body, ins, "fabs32", "float", "float")
		case "f32.neg":
			writeUnOp_Fcall(body, ins, "fneg32", "float", "float")
		case "f32.copysign":
			writeBinOp_Fcall(body, ins, "fcopysign32", "float", "float")
		case "f32.eq":
			writeBinOp2(body, ins, "==", "float", "uint64_t")
		case "f32.ne":
			writeBinOp2(body, ins, "!=", "float", "uint64_t")
		case "f32.lt":
			writeBinOp2(body, ins, "<", "float", "uint64_t")
		case "f32.le":
			writeBinOp2(body, ins, "<=", "float", "uint64_t")
		case "f32.gt":
			writeBinOp2(body, ins, ">", "float", "uint64_t")
		case "f32.ge":
			writeBinOp2(body, ins, ">=", "float", "uint64_t")

		case "f64.add":
			writeBinOp(body, ins, "+", "double")
		case "f64.sub":
			writeBinOp(body, ins, "-", "double")
		case "f64.mul":
			writeBinOp(body, ins, "*", "double")
		case "f64.div":
			writeBinOp(body, ins, "/", "double")
		case "f64.sqrt":
			writeUnOp_Fcall(body, ins, "fsqrt64", "double", "double")
		case "f64.min":
			writeBinOp_Fcall(body, ins, "fmin64", "double", "double")
		case "f64.max":
			writeBinOp_Fcall(body, ins, "fmax64", "double", "double")
		case "f64.ceil":
			writeUnOp_Fcall(body, ins, "fceil64", "double", "double")
		case "f64.floor":
			writeUnOp_Fcall(body, ins, "ffloor64", "double", "double")
		case "f64.trunc":
			writeUnOp_Fcall(body, ins, "ftrunc64", "double", "double")
		case "f64.nearest":
			writeUnOp_Fcall(body, ins, "fnearest64", "double", "double")
		case "f64.abs":
			writeUnOp_Fcall(body, ins, "fabs64", "double", "double")
		case "f64.neg":
			writeUnOp_Fcall(body, ins, "fneg64", "double", "double")
		case "f64.copysign":
			writeBinOp_Fcall(body, ins, "fcopysign64", "double", "double")
		case "f64.eq":
			writeBinOp2(body, ins, "==", "double", "uint64_t")
		case "f64.ne":
			writeBinOp2(body, ins, "!=", "double", "uint64_t")
		case "f64.lt":
			writeBinOp2(body, ins, "<", "double", "uint64_t")
		case "f64.le":
			writeBinOp2(body, ins, "<=", "double", "uint64_t")
		case "f64.gt":
			writeBinOp2(body, ins, ">", "double", "uint64_t")
		case "f64.ge":
			writeBinOp2(body, ins, ">=", "double", "uint64_t")

		case "i64.extend_u/i32":
			writeUnOp_Fcall(body, ins, "", "uint32_t", "uint64_t")
		case "i64.extend_s/i32":
			writeUnOp_Fcall(body, ins, "", "int32_t", "int64_t")

		case "i32.wrap/i64":
			writeUnOp_Fcall(body, ins, "", "uint32_t", "uint64_t")

		// TODO: These floating point operations need to be double-checked for correctness.

		case "i32.trunc_s/f32", "i64.trunc_s/f32", "i32.trunc_u/f32", "i64.trunc_u/f32":
			writeUnOp_Fcall(body, ins, "ftrunc32", "float", "int64_t")

		case "i32.trunc_s/f64", "i64.trunc_s/f64", "i32.trunc_u/f64", "i64.trunc_u/f64":
			writeUnOp_Fcall(body, ins, "ftrunc64", "double", "int64_t")

		case "f32.demote/f64":
			writeUnOp_Fcall(body, ins, "", "double", "float")

		case "f64.promote/f32":
			writeUnOp_Fcall(body, ins, "", "float", "double")

		case "f32.convert_s/i32":
			writeUnOp_Fcall(body, ins, "", "int32_t", "float")

		case "f32.convert_s/i64":
			writeUnOp_Fcall(body, ins, "", "int64_t", "float")

		case "f32.convert_u/i32":
			writeUnOp_Fcall(body, ins, "", "uint32_t", "float")

		case "f32.convert_u/i64":
			writeUnOp_Fcall(body, ins, "", "uint64_t", "float")

		case "f64.convert_s/i32":
			writeUnOp_Fcall(body, ins, "", "int32_t", "double")

		case "f64.convert_s/i64":
			writeUnOp_Fcall(body, ins, "", "int64_t", "double")

		case "f64.convert_u/i32":
			writeUnOp_Fcall(body, ins, "", "uint32_t", "double")

		case "f64.convert_u/i64":
			writeUnOp_Fcall(body, ins, "", "uint64_t", "double")

		case "i32.reinterpret/f32", "i64.reinterpret/f64", "f32.reinterpret/i32", "f64.reinterpret/i64":

		case "i32.load", "f32.load", "i64.load32_u":
			writeMemLoad(body, ins, "uint32_t")

		case "i32.load8_s", "i64.load8_s":
			writeMemLoad(body, ins, "int8_t")

		case "i32.load8_u", "i64.load8_u":
			writeMemLoad(body, ins, "uint8_t")

		case "i32.load16_s", "i64.load16_s":
			writeMemLoad(body, ins, "int16_t")

		case "i32.load16_u", "i64.load16_u":
			writeMemLoad(body, ins, "uint16_t")

		case "i64.load32_s":
			writeMemLoad(body, ins, "int32_t")

		case "i64.load", "f64.load":
			writeMemLoad(body, ins, "uint64_t")

		case "i32.store", "f32.store", "i64.store32":
			writeMemStore(body, ins, "uint32_t")

		case "i32.store8", "i64.store8":
			writeMemStore(body, ins, "uint8_t")

		case "i32.store16", "i64.store16":
			writeMemStore(body, ins, "uint16_t")

		case "i64.store", "f64.store":
			writeMemStore(body, ins, "uint64_t")

		case "current_memory":
			bSprintf(body,
				"%s%d.vu64 = vm->mem_size / 65536;",
				NGEN_VALUE_PREFIX, ins.Target,
			)

		case "grow_memory":
			bSprintf(body,
				"%s%d.vu64 = vm->mem_size / 65536; vm->grow_memory(vm, %s%d.vu32 * 65536);",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_VALUE_PREFIX, ins.Values[0],
			)

		case "add_gas":
			// TODO: Implement

		case "fp_disabled_error":
			bSprintf(body, "vm->throw_s(vm, \"floating point disabled\");")

		default:
			panic(ins.Op)
		}
		body.WriteByte('\n')
	}

	body.WriteString("\nreturn 0;\n")

	bSprintf(builder, "union Value phi")

	for id, _ := range valueIDs {
		bSprintf(builder, ",%s%d", NGEN_VALUE_PREFIX, id)
	}
	bSprintf(builder, ";\n")

	builder.WriteString(body.String())
	builder.WriteString("}\n")

	return builder.String()
}
