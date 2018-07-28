package exec

import (
	"bytes"
	"encoding/binary"
	//"fmt"
	"io"
	"github.com/go-interpreter/wagon/wasm/leb128"
	ops "github.com/go-interpreter/wagon/wasm/operators"
)

func readU32(r io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

func readU64(r io.Reader) (uint64, error) {
	var buf [8]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}


// Executes an init expr.
func execInitExpr(expr []byte, globals []int64) int64 {
	var stack []int64
	r := bytes.NewReader(expr)

	for {
		b, err := r.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		switch b {
		case ops.I32Const:
			i, err := leb128.ReadVarint32(r)
			if err != nil {
				panic(err)
			}
			stack = append(stack, int64(i))
		case ops.I64Const:
			i, err := leb128.ReadVarint64(r)
			if err != nil {
				panic(err)
			}
			stack = append(stack, int64(i))
		case ops.F32Const:
			i, err := readU32(r)
			if err != nil {
				panic(err)
			}
			stack = append(stack, int64(i))
		case ops.F64Const:
			i, err := readU64(r)
			if err != nil {
				panic(err)
			}
			stack = append(stack, int64(i))
		case ops.GetGlobal:
			index, err := leb128.ReadVarUint32(r)
			if err != nil {
				panic(err)
			}
			stack = append(stack, globals[int(index)])
		case ops.End:
			break
		default:
			panic("invalid opcode in init expr")
		}
	}

	return stack[len(stack) - 1]
}
