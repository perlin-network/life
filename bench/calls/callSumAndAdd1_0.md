https://wasdk.github.io/wasmcodeexplorer/

		case opcodes.GetLocal:
			//1, 149
			id := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4])) //2
			val := frame.Locals[id]
			frame.IP += 4
			frame.Regs[valueID] = val
			
		case opcodes.I32Const:
			// 2, 3
			val := LE.Uint32(frame.Code[frame.IP : frame.IP+4]) //1
			frame.IP += 4
			frame.Regs[valueID] = int64(val)			

		case opcodes.I32GeS:
			// 1, 31
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a >= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}			
			
		case opcodes.JmpIf:
			target := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			cond := int(LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8]))
			yieldedReg := int(LE.Uint32(frame.Code[frame.IP+8 : frame.IP+12]))
			frame.IP += 12
			if frame.Regs[cond] != 0 {
				vm.Yielded = frame.Regs[yieldedReg]
				frame.IP = target
			}
			
		case opcodes.Jmp:
			target := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			vm.Yielded = frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP = target			
			
		case opcodes.GetLocal:
			id := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			val := frame.Locals[id]
			frame.IP += 4
			frame.Regs[valueID] = val			
			
		case opcodes.ReturnValue:
			val := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.Destroy(vm)
			vm.CurrentFrame--
			if vm.CurrentFrame == -1 {
				vm.Exited = true
				vm.ReturnValue = val
				return
			}

			frame = vm.GetCurrentFrame()
			frame.Regs[frame.ReturnReg] = val			



1.0:{valueID: 1, opcode: opcodes.GetLocal, v1: 2, v2: 2}
1.9:{valueID: 2, opcode: opcodes.I32Const, v1: 1, v2: 1}
1.18:{valueID: 1, opcode: opcodes.I32GeS, v1: 1, v2: 2}
1.31:{valueID: 0, opcode: opcodes.JmpIf, v1: 61, v2: 1}
1.61:{valueID: 1, opcode: opcodes.GetLocal, v1: 0, v2: 2}
1.70:{valueID: 2, opcode: opcodes.GetLocal, v1: 1, v2: 1}
1.79:{valueID: 1, opcode: opcodes.Call, v1: 0, v2: 2}
0.0:{valueID: 1, opcode: opcodes.InvokeImport, v1: 0, v2: 0}
2020/08/04 15:23:01 Resolver called
0.9:{valueID: 0, opcode: opcodes.ReturnValue, v1: 1, v2: 0}
1.100:{valueID: 2, opcode: opcodes.I32Const, v1: 1, v2: 1}
1.109:{valueID: 1, opcode: opcodes.I32Add, v1: 1, v2: 2}
1.122:{valueID: 0, opcode: opcodes.SetLocal, v1: 0, v2: 1}
1.135:{valueID: 1, opcode: opcodes.GetLocal, v1: 2, v2: 2}
1.144:{valueID: 2, opcode: opcodes.I32Const, v1: 4294967295, v2: 1}
1.153:{valueID: 1, opcode: opcodes.I32Add, v1: 1, v2: 2}
1.166:{valueID: 0, opcode: opcodes.SetLocal, v1: 2, v2: 1}
1.179:{valueID: 0, opcode: opcodes.JmpIf, v1: 61, v2: 1}
1.61:{valueID: 1, opcode: opcodes.GetLocal, v1: 0, v2: 2}
1.70:{valueID: 2, opcode: opcodes.GetLocal, v1: 1, v2: 1}
1.79:{valueID: 1, opcode: opcodes.Call, v1: 0, v2: 2}
0.0:{valueID: 1, opcode: opcodes.InvokeImport, v1: 0, v2: 0}
0.9:{valueID: 0, opcode: opcodes.ReturnValue, v1: 1, v2: 0}
1.100:{valueID: 2, opcode: opcodes.I32Const, v1: 1, v2: 1}
1.109:{valueID: 1, opcode: opcodes.I32Add, v1: 1, v2: 2}
1.122:{valueID: 0, opcode: opcodes.SetLocal, v1: 0, v2: 1}
1.135:{valueID: 1, opcode: opcodes.GetLocal, v1: 2, v2: 2}
1.144:{valueID: 2, opcode: opcodes.I32Const, v1: 4294967295, v2: 1}
1.153:{valueID: 1, opcode: opcodes.I32Add, v1: 1, v2: 2}
1.166:{valueID: 0, opcode: opcodes.SetLocal, v1: 2, v2: 1}
1.179:{valueID: 0, opcode: opcodes.JmpIf, v1: 61, v2: 1}
1.196:{valueID: 1, opcode: opcodes.GetLocal, v1: 0, v2: 0}
1.205:{valueID: 0, opcode: opcodes.ReturnValue, v1: 1, v2: 0}			