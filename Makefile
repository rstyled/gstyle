QUALITY_TMP_DIR ?= /tmp

.PHONY: check complexity cognitive gstyle-check fmt lint quality-baseline quality-gate test test-race

check:
	go test -buildvcs=false ./...
	$(MAKE) complexity

complexity:
	go run -buildvcs=false github.com/fzipp/gocyclo/cmd/gocyclo -over 6 -ignore '_test.go' .

cognitive:
	go run -buildvcs=false github.com/uudashr/gocognit/cmd/gocognit -over 15 -ignore '_test.go' .

# Dogfood gate: gstyle's two dimensions (tile area = cyclomatic, color =
# cognitive) must each stay under 15 on non-test code. Kept separate from
# lint so the TUI's own health is its own CI signal.
gstyle-check:
	go run -buildvcs=false github.com/fzipp/gocyclo/cmd/gocyclo -over 15 -ignore '_test.go' .
	go run -buildvcs=false github.com/uudashr/gocognit/cmd/gocognit -over 15 -ignore '_test.go' .

# Quality gate: gstyle's guardrails (fn_params, mutated targets, side-effect
# density, ...) as excess rates per 1,000 functions, so the gate scales with
# the codebase instead of pinning absolute finding counts. `check` exits 1
# when it reports findings, which is expected; only exit 2 (analysis
# failure) fails the run.
quality-gate:
	go build -buildvcs=false -o "$(QUALITY_TMP_DIR)/gstyle-check-bin" ./cmd/gstyle
	"$(QUALITY_TMP_DIR)/gstyle-check-bin" check --format json . > "$(QUALITY_TMP_DIR)/gstyle-facts.json" || test $$? -eq 1
	go run ./application/qualitygate --baseline quality-baseline.json --facts "$(QUALITY_TMP_DIR)/gstyle-facts.json"

# Regenerate the checked-in baseline after a change legitimately moves the
# numbers (usually down).
quality-baseline:
	go build -buildvcs=false -o "$(QUALITY_TMP_DIR)/gstyle-check-bin" ./cmd/gstyle
	"$(QUALITY_TMP_DIR)/gstyle-check-bin" check --format json . > "$(QUALITY_TMP_DIR)/gstyle-facts.json" || test $$? -eq 1
	go run ./application/qualitygate --write quality-baseline.json --facts "$(QUALITY_TMP_DIR)/gstyle-facts.json"

fmt:
	go fmt ./...

lint:
	test -z "$$(gofmt -l .)"
	go vet -buildvcs=false ./...
	$(MAKE) complexity

test:
	go test -buildvcs=false -count=1 ./...

test-race:
	go test -buildvcs=false -race -count=1 ./...

.PHONY: verify-output

# Lean proof of the output decision and stable site selection.
verify-output:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --output

.PHONY: verify-length

# Actual Go -> Goose -> Lean/Iris, including source binding and negative controls.
verify-length:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py

.PHONY: verify-pdg

# Reuse the pinned Lean project; verify actual PDG query records and decisions.
verify-pdg:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --pdg

.PHONY: verify-negative-index

verify-negative-index:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --interval

.PHONY: verify-runtime-value
verify-runtime-value:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --runtime-value

.PHONY: verify-state-map
verify-state-map:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --state-map

.PHONY: verify-numeric-transfer
verify-numeric-transfer:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --numeric-transfer

.PHONY: verify-signed-step
verify-signed-step:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --signed-step

.PHONY: verify-assignment
verify-assignment:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --assignment

.PHONY: verify-branch
verify-branch:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --branch

.PHONY: verify-pdg-text
verify-pdg-text:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --pdg-text

.PHONY: verify-operator

verify-operator:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --operator

.PHONY: verify-pdg-list

verify-pdg-list:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --pdg-list

.PHONY: verify-pdg-attributes

verify-pdg-attributes:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --pdg-attributes

.PHONY: verify-runtime-facts

verify-runtime-facts:
	python3 -B -m unittest discover -s verification/length -p 'test_*.py'
	python3 -B verification/length/verify.py --runtime-facts
