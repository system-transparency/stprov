#!/bin/bash

#
# A script that takes stprov for a trial run in qemu.  Before running, you may
# need to install a few system dependencies.  See details in ../gitlab-ci.yml.
#
# Usage: ./qemu.sh
#
# Environment variables that can be enabled (disabled by default):
#
#   INTERACTIVE=true    Build and launch QEMU for manual stprov tests
#   SINGLE_TEST=num     Run a single test, num is a zero-based number
#

set -eu
trap clean_up EXIT

cd "$(dirname "$0")" # Change directory to where script is located
GOBIN="$(pwd)"/bin   # Use local directory for built go tools
export GOBIN

INTERACTIVE=${INTERACTIVE:-false}
SINGLE_TEST=${SINGLE_TEST:-false}

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

function info() {
	echo "INFO: $*" >&2
}

function assert_hostcfg() {
	local test_num=$1; shift
	local key=$1; shift
	local want=$1; shift
	local got

	got=$(jq "$key" saved/hostcfg.json)
	[[ "$got" == "$want" ]] || die "test $test_num: host config: wrong $key: got $got, want $want"
}

function assert_headreq() {
	local test_num=$1; shift
	local token

	token="HEAD request on provisioning url gave content-length: "
	grep -q "$token" saved/qemu.log || die "test $test_num: HTTP HEAD provisioning URL"
}

function mock_operator() {
	local configure=$1; shift
	local run=$1; shift

	if [[ "$INTERACTIVE" == true ]]; then
		echo "#!/bin/elvish"
		return
	fi

	# Mock an operator that boots into the system, configures the network, and
	# then runs the client-server ping-pongs.  The exact stprov remote commands
	# are templated so that we can easily loop over several different options.
	# The printed messages help us figure out how it's going, see reach_stage.
	cat << EOF
#!/bin/elvish

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

		if grep -q "^$token" saved/qemu.log; then
			break
		fi

		sleep 1
		abort_in_num_seconds=$(( abort_in_num_seconds - 1 ))
	done
}

###
# Initial setup
###
PORT=2009
IP=10.0.3.200
MASK=25
GATEWAY=10.0.3.129
DNS=10.0.3.130
OSPKG_SRV=10.0.3.131
IFNAME=eth0
IFADDR=aa:bb:cc:dd:ee:ff
HOST=testonly
FULLHOST=$HOST.example.org
USER=stprov
PASSWORD=sikritpassword
RESOURCE=ospkg.json
URL=http://$USER:$PASSWORD@$OSPKG_SRV/$RESOURCE

mkdir -p build saved bin
make -C ../\
	DEFAULT_TEMPLATE_URL="http://user:password@$OSPKG_SRV/$RESOURCE"\
	DEFAULT_DOMAIN="$(cut -d'.' -f2- <<<"$FULLHOST")"\
	DEFAULT_USER="$USER"\
	DEFAULT_PASSWORD="$PASSWORD"\
	DEFAULT_DNS="$DNS"\
	DEFAULT_ALLOWED_NETWORKS="$GATEWAY/32"
mv ../stprov bin/
go install ./serve-http

version=$(git describe --tags --always)
[[ "$(./bin/stprov version)" == "$version" ]] || die "invalid stprov version"

# go work interacts badly with building u-root itself and with
# u-root's building of included commands. It can be enabled for
# compilation of stprov above by using the GOWORK environment
# variable, and then disabled for the rest of this script.
unset GOWORK
version=$(go list -m -f '{{.Version}}' github.com/u-root/u-root)
[[ -d build/u-root ]] ||
	git clone --depth 1 -b "$version" https://github.com/u-root/u-root build/u-root &&
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
local_run="./bin/stprov local run --ip 127.0.0.1 -p $PORT --otp $PASSWORD"
remote_run="stprov remote run -p $PORT --otp=$PASSWORD" # use compiled-in default set via Makefile
remote_configs=(
	# Static network configuration
	"static -I $IFNAME -d $DNS -i $IP/$MASK -H $FULLHOST -r $URL --gateway $GATEWAY"
	"static -m $IFADDR -d $DNS -i $IP/$MASK -H $FULLHOST -r $URL        -g $GATEWAY"
	"static -A         -d $DNS -i $IP/$MASK -h $HOST     -u $USER -p $PASSWORD"
	"static -i $IP/$MASK -h $HOST" # use compiled-in defaults set via Makefile
	# DHCP network configuration
	"dhcp --interface $IFNAME --dns $DNS --full-host $FULLHOST --url $URL"
	"dhcp --mac       $IFADDR --dns $DNS --full-host $FULLHOST --user $USER --pass $PASSWORD"
	"dhcp --host $HOST" # use compiled-in defaults set via Makefile
)

printf '{"desc": "dummy ospkg"}' > "saved/$RESOURCE"
for i in "${!remote_configs[@]}"; do
	if [[ "$SINGLE_TEST" != false ]] && [[ "$SINGLE_TEST" != "$i" ]]; then
		continue # not the single test being requested by the user
	fi

	info "running test $i"

	remote_cfg="stprov remote ${remote_configs[$i]}"
	mock_operator "$remote_cfg" "$remote_run" > build/uinitcmd.sh

	./bin/u-root\
		-o build/stprov.cpio\
		-uroot-source=build/u-root\
		-uinitcmd="/bin/sh /bin/uinitcmd.sh"\
		-files bin/stprov:bin/stprov\
		-files build/uinitcmd.sh:bin/uinitcmd.sh\
		build/u-root/cmds/core/{init,elvish,shutdown,cat,cp,dd,echo,grep,hexdump,ls,mkdir,mv,ping,pwd,rm,wget,wc}

	# Documentation to understand qemu user networking and these options:
	# - https://wiki.qemu.org/Documentation/Networking#User_Networking_(SLIRP)
	# - https://www.qemu.org/docs/master/system/invocation.html#hxtool-5
	#
	# Be aware: our use of the guestfwd option appears broken in QEMU version
	# 7.2.7.  If you encounter the same issue, try QEMU version 8.1.1 instead.
	nic_opts="type=user"               # qemu user networking
	nic_opts="$nic_opts,net=$IP/$MASK" # guest NAT network
	nic_opts="$nic_opts,host=$GATEWAY" # guest gateway
	nic_opts="$nic_opts,dns=$DNS"      # guest dns server
	nic_opts="$nic_opts,dhcpstart=$IP" # guest dhcp server assigns this ip first
	nic_opts="$nic_opts,id=$IFNAME"    # guest interface name
	nic_opts="$nic_opts,mac=$IFADDR"   # guest mac address
	nic_opts="$nic_opts,restrict=yes"  # guest is isolated inside its NAT network
	nic_opts="$nic_opts,hostfwd=tcp:127.0.0.1:$PORT-$IP:$PORT"
	nic_opts="$nic_opts,guestfwd=tcp:$OSPKG_SRV:80-cmd:./bin/serve-http -d saved"

	qemu_opts=(
		-nographic -no-reboot -m 512M -M q35
		-rtc base=localtime -pidfile qemu.pid
		-object "rng-random,filename=/dev/urandom,id=rng0"
		-device "virtio-rng-pci,rng=rng0"
		-nic "$nic_opts"
		-drive "if=pflash,format=raw,readonly=on,file=$ovmf_code"
		-drive "if=pflash,format=raw,file=saved/OVMF_VARS.fd"
		-kernel "build/kernel.vmlinuz"
		-initrd "build/stprov.cpio"
		-append "console=ttyS0"
	)

	if [[ "$INTERACTIVE" == true ]]; then
		qemu-system-x86_64 "${qemu_opts[@]}"
		exit 0
	fi

	qemu-system-x86_64 "${qemu_opts[@]}" >saved/qemu.log &

	reach_stage "$i" 10 "stage:boot"
	reach_stage "$i" 60 "stage:network"
	assert_headreq "$i"

	sleep 3 # unclear why local_rune sometimes fail without this
	$local_run | tee saved/stprov.log
	reach_stage "$i" 3 "stage:shutdown"

	virt-fw-vars -i saved/OVMF_VARS.fd --output-json saved/efivars.json
	jq -r '.variables[] | select(.name == "STHostConfig") | .data' saved/efivars.json | tr a-f A-F | basenc --base16 -d | jq > saved/hostcfg.json
	jq -r '.variables[] | select(.name == "STHostKey")    | .data' saved/efivars.json | tr a-f A-F | basenc --base16 -d > saved/hostkey
	jq -r '.variables[] | select(.name == "STHostName")   | .data' saved/efivars.json | tr a-f A-F | basenc --base16 -d > saved/hostname

	#
	# Check echo:ed IP address
	#
	got=$(grep ^ip saved/stprov.log | cut -d'=' -f2)
	[[ "$got" == "127.0.0.1" ]] || die "test $i: stprov local ip: got $got, want 127.0.0.1"

	#
	# Check hostname
	#
	got=$(grep ^hostname saved/stprov.log | cut -d'=' -f2)
	[[ "$got" == "$FULLHOST" ]] || die "test $i: stprov local hostname: got $got, want $FULLHOST"

	got=$(cat saved/hostname)
	[[ "$got" == "$FULLHOST" ]] || die "test $i: EFI NVRAM hostname: got $got, want $FULLHOST"

	#
	# Check SSH key
	#
	chmod 600 saved/hostkey
	fingerprint=$(ssh-keygen -lf saved/hostkey | cut -d' ' -f2)

	got=$(grep ^fingerprint saved/stprov.log | cut -d'=' -f2)
	[[ "$got" == "$fingerprint" ]] || die "test $i: SSH key fingerprint: got $got, want $fingerprint"

	#
	# Check host configuration
	#
	mode=$(echo "$remote_cfg" | cut -d' ' -f3)
	want_ip=null
	want_gw=null
	if [[ "$mode" == static ]]; then
		want_ip="\"$IP/$MASK\"" # quote here so null can be unquoted
		want_gw="\"$GATEWAY\""  # quote here so null can be unquoted
	fi

	assert_hostcfg "$i" ".network_mode"                         "\"$mode\""
	assert_hostcfg "$i" ".host_ip"                                "$want_ip"
	assert_hostcfg "$i" ".gateway"                                "$want_gw"
	assert_hostcfg "$i" ".dns[0]"                               "\"$DNS\""
	assert_hostcfg "$i" ".network_interfaces[0].interface_name" "\"$IFNAME\""
	assert_hostcfg "$i" ".network_interfaces[0].mac_address"    "\"$IFADDR\""
	assert_hostcfg "$i" ".ospkg_pointer"                        "\"$URL\""
	assert_hostcfg "$i" ".identity"                             "\"bar\"" # FIXME
	assert_hostcfg "$i" ".authentication"                       "\"foo\"" # FIXME
	assert_hostcfg "$i" ".bonding_mode"                         null # only tested manually
	assert_hostcfg "$i" ".bonding_name"                         null # only tested manually
done
