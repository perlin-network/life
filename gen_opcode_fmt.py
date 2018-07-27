import sys

code = sys.stdin.read().split("\n")
started = False

for line in code:
    line = line.strip()
    if line == "":
        continue
    if line.endswith("= iota"):
        started = True
        print("package opcodes\n")
        print("func (op Opcode) String() string {")
        print("\tswitch op {")
        line = line.split(" ")[0].strip()
    if started:
        if line == ")":
            print("\t}")
            print("\treturn \"Unknown\"")
            print("}")
            break
        print("\tcase {}: return \"{}\"".format(line, line))
