#!/usr/bin/env bash
#
# End-to-end smoke test for the compiled usb-soundcard-mapper binary.
#
# It builds the binary, puts the fake lsusb/aplay/udevadm commands from
# testdata/fakebin on PATH, and drives the real executable through its
# command-line surface — asserting on exit codes and output. No physical USB
# audio hardware is required.
#
# Usage: scripts/e2e.sh
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

bin="$work/usb-soundcard-mapper"
fakebin="$work/fakebin"
rules="$work/rules.d"
mkdir -p "$fakebin" "$rules"

pass=0
fail=0

log()  { printf '  %s\n' "$*"; }
ok()   { printf '\033[32mPASS\033[0m %s\n' "$*"; pass=$((pass + 1)); }
bad()  { printf '\033[31mFAIL\033[0m %s\n' "$*"; fail=$((fail + 1)); }

echo "== building binary =="
go build -o "$bin" .

echo "== installing fake commands =="
for f in testdata/fakebin/*; do
	install -m 0755 "$f" "$fakebin/"
done

# Fakes take precedence, but keep the rest of PATH for bash/cat/grep/etc.
export PATH="$fakebin:$PATH"
export FAKE_DEV_DIR="$work/scenario"
mkdir -p "$FAKE_DEV_DIR"

# run_case <name> <expected_exit> <grep-pattern|-> -- <args...>
run_case() {
	local name="$1" want_rc="$2" pattern="$3"
	shift 3
	[[ "${1:-}" == "--" ]] && shift

	local out rc
	out="$("$bin" "$@" 2>&1)" && rc=0 || rc=$?

	if [[ "$rc" -ne "$want_rc" ]]; then
		bad "$name (exit $rc, want $want_rc)"
		log "$out"
		return
	fi
	if [[ "$pattern" != "-" ]] && ! grep -qE "$pattern" <<<"$out"; then
		bad "$name (output missing /$pattern/)"
		log "$out"
		return
	fi
	ok "$name"
}

echo "== running cases =="

# --help prints usage and exits 0 (flag package handles -h).
run_case "help/usage" 0 'Usage:' -- -h

# Invalid vendor ID is rejected by config validation before anything else.
run_case "invalid vendor id" 1 'invalid vendor ID' -- --vendor-id ZZZZ

# Invalid product ID is likewise rejected up front.
run_case "invalid product id" 1 'invalid product ID' -- --product-id NOPE

# Scenario: only a non-USB onboard card is present, so detection finds no USB
# sound cards and the tool reports that cleanly.
cat >"$FAKE_DEV_DIR/aplay_l.txt" <<-'EOF'
	**** List of PLAYBACK Hardware Devices ****
	card 0: PCH [HDA Intel PCH], device 0: ALC892 Analog [ALC892 Analog]
EOF

# --list reaches detection and reports no USB sound cards, exiting cleanly.
run_case "list (no cards)" 0 'No USB sound cards found' -- --list --retries 0

# Dry run needs no privileges and ends cleanly when no USB card is present.
run_case "dry-run non-interactive" 0 'No USB sound cards found' -- \
	--non-interactive --vendor-id 1234 --product-id 5678 --dry-run --retries 0

# A mistyped, non-positive command timeout must not brick a field deployment:
# validation clamps it to the default and the run still completes cleanly.
run_case "zero command-timeout is clamped" 0 'No USB sound cards found' -- \
	--list --command-timeout 0 --retries 0

# A negative retry count is clamped to zero rather than skipping execution.
run_case "negative retries is clamped" 0 'No USB sound cards found' -- \
	--list --retries=-1

# An unknown log level is normalized to info instead of aborting the run.
run_case "unknown log level is normalized" 0 'No USB sound cards found' -- \
	--list --log-level nonsense --retries 0

# When aplay advertises a USB card that is absent from /sys (card 99 exists in no
# real environment), the tool reports the inconsistency and exits non-zero.
cat >"$FAKE_DEV_DIR/aplay_l.txt" <<-'EOF'
	**** List of PLAYBACK Hardware Devices ****
	card 99: Device [USB Audio Device], device 0: USB Audio [USB Audio]
EOF
run_case "aplay/sysfs mismatch" 1 'device disconnected|failed to process' -- --list --retries 0

# With lsusb/aplay/udevadm absent from PATH, the tool reports the missing
# commands and exits non-zero. An empty directory as PATH makes this
# deterministic regardless of what the host has installed in /usr/bin.
emptybin="$work/emptybin"
mkdir -p "$emptybin"
out="$(PATH="$emptybin" "$bin" --list 2>&1)" && rc=0 || rc=$?
if [[ "$rc" -eq 1 ]] && grep -qE 'required commands not found' <<<"$out"; then
	ok "missing commands detected"
else
	bad "missing commands detected (exit $rc)"
	log "$out"
fi

echo
echo "== summary: $pass passed, $fail failed =="
[[ "$fail" -eq 0 ]]
