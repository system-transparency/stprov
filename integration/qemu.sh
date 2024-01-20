#!/bin/bash

#
# A simple script that takes stprov for a trial run with qemu.  Before running,
# you may need to install a few dependencies.  See details in ../gitlab-ci.yml.
#
# Usage: ./qemu.sh
#

set -eu
trap clean_up EXIT

cd "$(dirname "$0")" # Change directory to where script is located
GOBIN="$(pwd)"/bin   # Use local directory for built go tools
export GOBIN

rm -f qemu.pid

function clean_up() {
    local qemu_pid
    qemu_pid=$(cat qemu.pid 2>/dev/null) || return 0

    # QEMU removes the pid file before exiting. There is a race where
    # we might read the pid file and attempt to kill the process too
    # late. Due to the short period, it's extremely unlikely that the
    # pid has already been reused for a different process.
    kill "${qemu_pid}"
}

function die()  {
	echo "FAIL: $*" >&2
	exit 1
}

function pass() {
	echo "PASS: $*" >&2
}

function assert_hostcfg() {
	local test_num=$1; shift
	local key=$1; shift
	local want=$1; shift
	local got

	got=$(jq "$key" saved/hostcfg.json)
	[[ "$got" == "$want" ]] || die "test $test_num: host config: wrong $key: got $got, want $want"
}

function mock_operator() {
	local configure=$1; shift
	local run=$1; shift

	# Mock an operator that boots into the system, configures the network, and
	# then runs the client-server ping-pongs.  The exact stprov remote commands
	# are templated so that we can easily loop over several different options.
	# The printed messages help us figure out how it's going, see reach_stage.
	cat << EOF
#!/bin/sh

printf "stage:boot\n"
$configure

printf "stage:network\n"
printf "\n" | $run
printf "\n"

printf "stage:shutdown\n"
shutdown
EOF
}

function reach_stage() {
	local test_num=$1; shift
	local abort_in_num_seconds=$1; shift
	local token=$1; shift

	while :; do
		if [[ $abort_in_num_seconds == 0 ]]; then
			die "test $test_num: reach $token"
		fi

		if grep -q "$token" saved/qemu.log; then
			pass "test $test_num: reach $token"
			break
		fi

		sleep 1
		abort_in_num_seconds=$(( abort_in_num_seconds - 1 ))
	done
}

###
# Initial setup
###
mkdir -p build saved
go install ../cmd/stprov

# go work interacts badly with building u-root itself and with
# u-root's building of included commands. It can be enabled for
# compilation of stprov above by using the GOWORK environment
# variable, and then disabled for the rest of this script.
unset GOWORK
[[ -d build/u-root ]] ||
	git clone --depth 1 https://github.com/u-root/u-root build/u-root &&
	(cd build/u-root && go install)

url="https://git.glasklar.is/system-transparency/core/system-transparency/-/raw/main/contrib/linuxboot.vmlinuz"
[[ -f build/kernel.vmlinuz ]] || curl -L "$url" -o build/kernel.vmlinuz

# Setup EFI-NVRAM stuff.  Magic, if you understand the choises please docdoc here.
#
# From:
# https://git.glasklar.is/system-transparency/core/system-transparency/-/blob/main/tasks/qemu.yml?ref_type=heads#L5-19
ovmf_code=""
for str in "OVMF" "edk2/ovmf" "edk2-ovmf/x64"; do
	file=/usr/share/"$str"/OVMF_CODE.fd
	if [[ -f "$file" ]]; then
		ovmf_code="$file"
		cp /usr/share/"$str"/OVMF_VARS.fd saved/OVMF_VARS.fd
		break
	fi
done

[[ -n "$ovmf_code" ]] || die "unable to locate OVMF_CODE.fd"

###
# Run tests
###
URL=https://example.org/ospkg.json
FULLHOST=example.org

local_run="./bin/stprov local run --ip 127.0.0.1 -p 2009 --otp sikritpassword"
remote_run="stprov remote run -p 2009 --allow=0.0.0.0/0 --otp=sikritpassword"
remote_configs=(
	"stprov remote static -A --ip=10.0.2.15/24 --full-host=$FULLHOST --url=https://example.org/ospkg.json"
)

for i in "${!remote_configs[@]}"; do
	remote_cfg=${remote_configs[$i]}
	mock_operator "$remote_cfg" "$remote_run" > build/uinitcmd.sh

	./bin/u-root\
		-o build/stprov.cpio\
		-uroot-source=build/u-root\
		-uinitcmd="/bin/sh /bin/uinitcmd.sh"\
		-files bin/stprov:bin/stprov\
		-files build/uinitcmd.sh:bin/uinitcmd.sh\
		build/u-root/cmds/core/{init,elvish,shutdown}

	qemu-system-x86_64 -nographic -no-reboot -pidfile qemu.pid\
		-m 512M -M q35 -rtc base=localtime\
		-net user,hostfwd=tcp::2009-:2009 -net nic\
		-object rng-random,filename=/dev/urandom,id=rng0\
		-device virtio-rng-pci,rng=rng0\
		-drive if=pflash,format=raw,readonly=on,file="$ovmf_code"\
		-drive if=pflash,format=raw,file=saved/OVMF_VARS.fd\
		-kernel build/kernel.vmlinuz\
		-initrd build/stprov.cpio\
		-append "console=ttyS0" >saved/qemu.log &

	reach_stage "$i" 10 "stage:boot"
	reach_stage "$i" 60 "stage:network"
	$local_run | tee saved/stprov.log
	reach_stage "$i" 3 "stage:shutdown"

	virt-fw-vars -i saved/OVMF_VARS.fd --output-json saved/efivars.json
	jq -r '.variables[] | select(.name == "STHostConfig") | .data' saved/efivars.json | tr a-f A-F | basenc --base16 -d | jq > saved/hostcfg.json
	jq -r '.variables[] | select(.name == "STHostKey")    | .data' saved/efivars.json | tr a-f A-F | basenc --base16 -d > saved/hostkey
	jq -r '.variables[] | select(.name == "STHostName")   | .data' saved/efivars.json | tr a-f A-F | basenc --base16 -d > saved/hostname

	#
	# Check hostname
	#
	got=$(grep hostname saved/stprov.log | cut -d'=' -f2)
	[[ "$got" == "$FULLHOST" ]] || die "test $i: stprov local hostname: got $got, want $FULLHOST"

	got=$(cat saved/hostname)
	[[ "$got" == "$FULLHOST" ]] || die "test $i: EFI NVRAM hostname: got $got, want $FULLHOST"

	pass "hostname"

	#
	# Check SSH key
	#
	chmod 600 saved/hostkey
	fingerprint=$(ssh-keygen -lf saved/hostkey | cut -d' ' -f2)

	got=$(grep fingerprint saved/stprov.log | cut -d'=' -f2)
	[[ "$got" == "$fingerprint" ]] || die "test $i: SSH key fingerprint: got $got, want $fingerprint"

	pass "SSH key"

	#
	# Check host configuration
	#
	assert_hostcfg "$i" ".ospkg_pointer" "\"$URL\""

	pass "host configuration (URL-check only)"
done
