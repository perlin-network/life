JSON_DIR:=tests/generated
WAST_DIR:=testsuite
NO_TEST:=tests/skipped/%.no-test
TEST_RUNNER:=target/test_runner

# Run all test fixtures
.PHONY: test
test: build testsuite test_runner wast $(patsubst ${JSON_DIR}/%.json, ${NO_TEST}, $(wildcard ${JSON_DIR}/*.json))

.PHONY: build
build:
	go build .

target/test_runner:
	go build -o ${TEST_RUNNER} ./spec/test_runner

# Pseudo target to run each test fixture
${NO_TEST}: ${JSON_DIR}/%.json
	${TEST_RUNNER} $<

# Get latest test suite as a git submodule
.PHONY: testsuite
testsuite:
	git submodule update --init testsuite

# Build all JSON specs
.PHONY: wast
wast: $(patsubst ${WAST_DIR}/%.wast, ${JSON_DIR}/%.json, $(wildcard ${WAST_DIR}/*.wast))

# Build JSON specs - requires the wast2json binary to be on PATH, from wabt (https://github.com/WebAssembly/wabt)
${JSON_DIR}/%.json: testsuite/%.wast
	wast2json $< -o $@


