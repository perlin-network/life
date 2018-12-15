with open("opcodes.go") as f:
    data = f.read()
    lines = data.split("const (")[1].split(")")[0].strip().split("\n")
    id = 0

    out = "#[repr(u8)]\n#[derive(Copy, Clone, Eq, PartialEq)]\npub enum Opcode {\n"
    for i, line in enumerate(lines):
        line = line.strip().split(" ")[0]
        if len(line) > 0:
            out += "    {0} = {1},\n".format(line, id)
            id += 1
    out += "}\n"
    with open("opcodes.rs", "w") as outFile:
        outFile.write(out)
