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
const NGEN_UINT32_MASK = "0xffffffffull"
const NGEN_HEADER = `
struct VirtualMachine;
typedef uint64_t (*ExternalFunction)(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params);
struct VirtualMachine {
	void (*throw_s)(struct VirtualMachine *vm, const char *s);
	ExternalFunction (*resolve_import)(struct VirtualMachine *vm, const char *module_name, const char *field_name);
};
`

func bSprintf(builder *strings.Builder, format string, args ...interface{}) {
	builder.WriteString(fmt.Sprintf(format, args...))
}

func writeUnOp_Eqz(b *strings.Builder, ins Instr, ty string) {
	bSprintf(b,
		"%s%d = ((* (%s*) &%s%d) == 0);",
		NGEN_VALUE_PREFIX, ins.Target,
		ty, NGEN_VALUE_PREFIX, ins.Values[0],
	)
}

func writeUnOp_Fcall(b *strings.Builder, ins Instr, f string, ty string) {
	bSprintf(b,
		"%s%d = %s%s(* (%s*) &%s%d);",
		NGEN_VALUE_PREFIX, ins.Target,
		NGEN_FUNCTION_PREFIX, f,
		ty, NGEN_VALUE_PREFIX, ins.Values[0],
	)
}

func writeBinOp_Shift(b *strings.Builder, ins Instr, op string, ty string, rounding uint64) {
	bSprintf(b,
		"%s%d = ((* (%s*) &%s%d) %s ((* (%s*) &%s%d) %% %d));",
		NGEN_VALUE_PREFIX, ins.Target,
		ty, NGEN_VALUE_PREFIX, ins.Values[0],
		op,
		ty, NGEN_VALUE_PREFIX, ins.Values[1],
		rounding,
	)
}

func writeBinOp_Fcall(b *strings.Builder, ins Instr, f string, ty string) {
	bSprintf(b,
		"* (%s*) &%s%d = %s%s(* (%s*) &%s%d, * (%s*) &%s%d);",
		ty,
		NGEN_VALUE_PREFIX, ins.Target,
		NGEN_FUNCTION_PREFIX, f,
		ty, NGEN_VALUE_PREFIX, ins.Values[0],
		ty, NGEN_VALUE_PREFIX, ins.Values[1],
	)
}

func writeBinOp_ConstRv(b *strings.Builder, ins Instr, op string, ty string, rv string) {
	bSprintf(b,
		"* (%s*) &%s%d = ((* (%s*) &%s%d) %s (%s));",
		ty,
		NGEN_VALUE_PREFIX, ins.Target,
		ty, NGEN_VALUE_PREFIX, ins.Values[0],
		op,
		rv,
	)
}

func writeBinOp2(b *strings.Builder, ins Instr, op string, ty string, retTy string) {
	bSprintf(b,
		"* (%s*) &%s%d = ((* (%s*) &%s%d) %s (* (%s*) &%s%d));",
		retTy,
		NGEN_VALUE_PREFIX, ins.Target,
		ty, NGEN_VALUE_PREFIX, ins.Values[0],
		op,
		ty, NGEN_VALUE_PREFIX, ins.Values[1],
	)
}

func writeBinOp(b *strings.Builder, ins Instr, op string, ty string) {
	writeBinOp2(b, ins, op, ty, ty)
}

func (c *SSAFunctionCompiler) NGen(selfID uint64, numParams uint64, numLocals uint64) string {
	builder := &strings.Builder{}

	bSprintf(builder, "uint64_t %s%d(struct VirtualMachine *vm", NGEN_FUNCTION_PREFIX, selfID)

	for i := uint64(0); i < numParams; i++ {
		bSprintf(builder, ",uint64_t %s%d", NGEN_LOCAL_PREFIX, i)
	}
	builder.WriteString(") {\n")

	builder.WriteString("uint64_t phi = 0;\n")

	for i := uint64(0); i < numLocals; i++ {
		bSprintf(builder, "uint64_t %s%d = 0;\n", NGEN_LOCAL_PREFIX, i+numParams)
	}

	body := &strings.Builder{}
	valueIDs := make(map[TyValueID]struct{})

	for i, ins := range c.Code {
		valueIDs[ins.Target] = struct{}{}

		bSprintf(body, "\n%s%d:\n", NGEN_INS_LABEL_PREFIX, i)
		switch ins.Op {
		case "unreachable":
			bSprintf(body, "%strap_unreachable();", NGEN_ENV_API_PREFIX)
		case "return":
			if len(ins.Values) == 0 {
				body.WriteString("return 0;")
			} else {
				bSprintf(body, "return %s%d;", NGEN_VALUE_PREFIX, ins.Values[0])
			}
		case "get_local":
			bSprintf(body,
				"%s%d = %s%d;",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_LOCAL_PREFIX, ins.Immediates[0],
			)
		case "set_local":
			bSprintf(body,
				"%s%d = %s%d;",
				NGEN_LOCAL_PREFIX, ins.Immediates[0],
				NGEN_VALUE_PREFIX, ins.Values[0],
			)
		case "get_global":
			bSprintf(body,
				"%s%d = %sget_global(%d);",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_ENV_API_PREFIX, ins.Immediates[0],
			)
		case "set_global":
			bSprintf(body,
				"%s%d = %sset_global(%d, %s%d);",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_ENV_API_PREFIX, ins.Immediates[0],
				NGEN_VALUE_PREFIX, ins.Values[0],
			)
		case "call":
			bSprintf(body,
				"%s%d = %s%d(vm",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_FUNCTION_PREFIX, ins.Immediates[0],
			)
			for _, v := range ins.Values {
				bSprintf(body, ",%s%d", NGEN_VALUE_PREFIX, v)
			}
			body.WriteString(");")
		case "call_indirect":
			bSprintf(body,
				"%s%d = ((uint64_t (*)(struct VirtualMachine *",
				NGEN_VALUE_PREFIX, ins.Target,
			)
			for range ins.Values[:len(ins.Values)-1] {
				bSprintf(body, ",uint64_t")
			}
			bSprintf(body,
				")) %sresolve_indirect(vm, %s%d, %d)) (vm",
				NGEN_ENV_API_PREFIX,
				NGEN_VALUE_PREFIX, ins.Values[len(ins.Values)-1],
				len(ins.Values)-1,
			)
			for _, v := range ins.Values[:len(ins.Values)-1] {
				bSprintf(body, ",%s%d", NGEN_VALUE_PREFIX, v)
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
				"if(%s%d & 0xffffffffull) { phi = %s%d; goto %s%d; }",
				NGEN_VALUE_PREFIX, ins.Values[0],
				NGEN_VALUE_PREFIX, ins.Values[1],
				NGEN_INS_LABEL_PREFIX, ins.Immediates[0],
			)
		case "jmp_either":
			bSprintf(body,
				"phi = %s%d; if(%s%d & 0xffffffffull) { goto %s%d; } else { goto %s%d; }",
				NGEN_VALUE_PREFIX, ins.Values[0],
				NGEN_VALUE_PREFIX, ins.Values[1],
				NGEN_INS_LABEL_PREFIX, ins.Immediates[0],
				NGEN_INS_LABEL_PREFIX, ins.Immediates[1],
			)
		case "jmp_table":
			bSprintf(body, "phi = %s%d;\n", NGEN_VALUE_PREFIX, ins.Values[0])
			bSprintf(body, "switch(%s%d & 0xffffffffull) {\n", NGEN_VALUE_PREFIX, ins.Values[1])
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
				"%s%d = (%s%d & %s) ? %s%d : %s%d;",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_VALUE_PREFIX, ins.Values[2],
				NGEN_UINT32_MASK,
				NGEN_VALUE_PREFIX, ins.Values[0],
				NGEN_VALUE_PREFIX, ins.Values[1],
			)
		case "i32.const", "f32.const":
			bSprintf(body,
				"%s%d = (int32_t) (%d);",
				NGEN_VALUE_PREFIX, ins.Target,
				int32(ins.Immediates[0]),
			)
		case "i32.add":
			writeBinOp(body, ins, "+", "uint32_t")
		case "i32.sub":
			writeBinOp(body, ins, "-", "uint32_t")
		case "i32.mul":
			writeBinOp(body, ins, "*", "uint32_t")
		case "i32.div_s":
			writeBinOp(body, ins, "/", "int32_t")
		case "i32.div_u":
			writeBinOp(body, ins, "/", "uint32_t")
		case "i32.rem_s":
			writeBinOp(body, ins, "%", "int32_t")
		case "i32.rem_u":
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
			writeBinOp_Fcall(body, ins, "rotl32", "uint32_t")
		case "i32.rotr":
			writeBinOp_Fcall(body, ins, "rotr32", "uint32_t")
		case "i32.clz":
			writeUnOp_Fcall(body, ins, "clz32", "uint32_t")
		case "i32.ctz":
			writeUnOp_Fcall(body, ins, "ctz32", "uint32_t")
		case "i32.popcnt":
			writeUnOp_Fcall(body, ins, "popcnt32", "uint32_t")
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
				"%s%d = (int64_t) (%d);",
				NGEN_VALUE_PREFIX, ins.Target,
				int64(ins.Immediates[0]),
			)
		case "i64.add":
			writeBinOp(body, ins, "+", "uint64_t")
		case "i64.sub":
			writeBinOp(body, ins, "-", "uint64_t")
		case "i64.mul":
			writeBinOp(body, ins, "*", "uint64_t")
		case "i64.div_s":
			writeBinOp(body, ins, "/", "int64_t")
		case "i64.div_u":
			writeBinOp(body, ins, "/", "uint64_t")
		case "i64.rem_s":
			writeBinOp(body, ins, "%", "int64_t")
		case "i64.rem_u":
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
			writeBinOp_Fcall(body, ins, "rotl64", "uint64_t")
		case "i64.rotr":
			writeBinOp_Fcall(body, ins, "rotr64", "uint64_t")
		case "i64.clz":
			writeUnOp_Fcall(body, ins, "clz64", "uint64_t")
		case "i64.ctz":
			writeUnOp_Fcall(body, ins, "ctz64", "uint64_t")
		case "i64.popcnt":
			writeUnOp_Fcall(body, ins, "popcnt64", "uint64_t")
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
			writeUnOp_Fcall(body, ins, "fsqrt32", "float")
		case "f32.min":
			writeBinOp_Fcall(body, ins, "fmin32", "float")
		case "f32.max":
			writeBinOp_Fcall(body, ins, "fmax32", "float")
		case "f32.ceil":
			writeUnOp_Fcall(body, ins, "fceil32", "float")
		case "f32.floor":
			writeUnOp_Fcall(body, ins, "ffloor32", "float")
		case "f32.trunc":
			writeUnOp_Fcall(body, ins, "ftrunc32", "float")
		case "f32.nearest":
			writeUnOp_Fcall(body, ins, "fnearest32", "float")
		case "f32.abs":
			writeUnOp_Fcall(body, ins, "fabs32", "float")
		case "f32.neg":
			writeUnOp_Fcall(body, ins, "fneg32", "float")
		case "f32.copysign":
			writeBinOp_Fcall(body, ins, "fcopysign32", "float")
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
			writeUnOp_Fcall(body, ins, "fsqrt64", "double")
		case "f64.min":
			writeBinOp_Fcall(body, ins, "fmin64", "double")
		case "f64.max":
			writeBinOp_Fcall(body, ins, "fmax64", "double")
		case "f64.ceil":
			writeUnOp_Fcall(body, ins, "fceil64", "double")
		case "f64.floor":
			writeUnOp_Fcall(body, ins, "ffloor64", "double")
		case "f64.trunc":
			writeUnOp_Fcall(body, ins, "ftrunc64", "double")
		case "f64.nearest":
			writeUnOp_Fcall(body, ins, "fnearest64", "double")
		case "f64.abs":
			writeUnOp_Fcall(body, ins, "fabs64", "double")
		case "f64.neg":
			writeUnOp_Fcall(body, ins, "fneg64", "double")
		case "f64.copysign":
			writeBinOp_Fcall(body, ins, "fcopysign64", "double")
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
			writeBinOp_ConstRv(body, ins, "&", "uint64_t", "0xffffffffull")
		case "i64.extend_s/i32":
			writeBinOp_ConstRv(body, ins, "&", "uint64_t", "0xffffffffull")
			bSprintf(body,
				"\nif((%s%d >> 31) & 1) %s%d |= 0xffffffff00000000ull;",
				NGEN_VALUE_PREFIX, ins.Target,
				NGEN_VALUE_PREFIX, ins.Target,
			)
		case "i32.wrap/i64":
			writeBinOp_ConstRv(body, ins, "&", "uint64_t", "0xffffffffull")

		default:
			panic(ins.Op)
		}
		body.WriteByte('\n')
	}

	body.WriteString("\nreturn 0;\n")

	for id, _ := range valueIDs {
		bSprintf(builder, "uint64_t %s%d = 0;\n", NGEN_VALUE_PREFIX, id)
	}

	builder.WriteString(body.String())
	builder.WriteString("}\n")

	return builder.String()
}
