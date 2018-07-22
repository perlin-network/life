package codegen

import (
	"fmt"
	"strings"
	"github.com/perlin-network/life/compiler"
)

const ValueNamePrefix = "v"
const FunctionNamePrefix = "LFunc_"
const LabelPrefix = "LL"

type Instr struct {
	Target int
	Body string
}

type FunctionIR struct {
	ID int
	NumParams int
	NumLocals int
	HasReturn bool
	InstructionSets [][]Instr
	NextTempValueID int
	NextTempLabelID int
}

func (f *FunctionIR) Serialize() string {
	argList := make([]string, f.NumParams)
	for i := 0; i < f.NumParams; i++ {
		argList[i] = "i64"
	}

	ret := fmt.Sprintf(
		"define i64 @%s%d(%s) local_unnamed_addr {\n",
		FunctionNamePrefix,
		f.ID,
		strings.Join(argList, ", "),
	)

	ret += "%yielded = alloca i64\n"
	ret += fmt.Sprintf("%%locals = alloca i64, i64 %d\n", f.NumParams + f.NumLocals)
	for i := 0; i < f.NumParams; i++ {
		ret += fmt.Sprintf("%%li%d = getelementptr inbounds i64, i64* %%locals, i64 0, i64 %d\n", i, i)
		ret += fmt.Sprintf("store i64 %%%d, i64* %%li%d\n", i, i)
	}

	for i, set := range f.InstructionSets {
		ret += fmt.Sprintf("%s%d:\n", LabelPrefix, i)
		for _, ins := range set {
			if ins.Target != 0 {
				if ins.Target < 0 {
					panic("ins.Target cannot be less than 0")
				} else {
					ret += fmt.Sprintf("%%%s%d = ", ValueNamePrefix, ins.Target)
				}
			}
			ret += ins.Body
			ret += "\n"
		}
	}

	ret += "}"
	return ret
}

func (f *FunctionIR) RefTempValue() string {
	f.NextTempValueID++
	return fmt.Sprintf("%%tv%d", f.NextTempValueID)
}

func (f *FunctionIR) RefTempLabel() string {
	f.NextTempLabelID++
	return fmt.Sprintf("tl%d", f.NextTempLabelID)
}

func NewInstr(target compiler.TyValueID, body string, args... interface{}) Instr {
	return Instr {
		Target: int(target),
		Body: fmt.Sprintf(body, args...),
	}
}

func NewBr(insID int) Instr {
	return Instr {
		Body: fmt.Sprintf("br label %%%s%d", LabelPrefix, insID),
	}
}

func NewCondBr(cond string, insID1, insID2 int) Instr {
	return Instr {
		Body: fmt.Sprintf("br i1 %s, label %%%s%d, label %%%s%d", cond, LabelPrefix, insID1, LabelPrefix, insID2),
	}
}

func RefValue(id compiler.TyValueID) string {
	return fmt.Sprintf("%%%s%d", ValueNamePrefix, int(id))
}

func RefParam(id int) string {
	return fmt.Sprintf("%%%d", id)
}
