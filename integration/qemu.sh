#!/bin/bash

#
# A simple script that takes stprov for a trial run with qemu.  Before running,
# you may need to install a few dependencies.  See details in ../gitlab-ci.yml.
#
# Usage: ./qemu.sh
#

set -eu
trap clean_up EXIT

# Change directory to where script is located.
cd "$(dirname $0)"
# Use local directory for built go tools.
export GOBIN="$(pwd)"/bin

rm -f qemu.pid

function clean_up() {
    # QEMU removes the pid file before exiting. There is a race where
    # we might read the pid file and attempt to kill the process too
    # late. Due to the short period, it's extremely unlikely that the
    # pid has already been reused for a different process.
    local qemu_pid
    qemu_pid=$(cat qemu.pid 2>/dev/null) || return 0
    kill "${qemu_pid}"
}

mkdir -p build saved

###
# Build
###

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

./bin/u-root\
  -o build/stprov.cpio\
  -uroot-source=build/u-root\
  -uinitcmd="/bin/sh /bin/uinitcmd.sh"\
  -files bin/stprov:bin/stprov\
  -files uinitcmd.sh:bin/uinitcmd.sh\
  build/u-root/cmds/core/{init,elvish,shutdown}

echo "PASS: build"

###
# Setup EFI-NVRAM stuff.  Magic, if you understand the choises please docdoc here.
#
# From:
# https://git.glasklar.is/system-transparency/core/system-transparency/-/blob/main/tasks/qemu.yml?ref_type=heads#L5-19
###
ovmf_code=""
for str in "OVMF" "edk2/ovmf" "edk2-ovmf/x64"; do
  file=/usr/share/"$str"/OVMF_CODE.fd
  if [[ -f "$file" ]]; then
    ovmf_code="$file"
    cp /usr/share/"$str"/OVMF_VARS.fd saved/OVMF_VARS.fd
    break
  fi
done

if [[ -z "$ovmf_code" ]]; then
  echo "FATAL: failed to locate OVMF_CODE.fd" 2>&1
  exit 1
fi

echo "PASS: copy OVMF files"

###
# Run with qemu
###
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

###
# Run tests
###
function reach_stage() {
  abort_in_num_seconds=$1
  while :; do
    if [[ $abort_in_num_seconds == 0 ]]; then
      echo "FAIL: reach $2" >&2
      exit 1
    fi

    if [[ ! -z $(grep "$2" saved/qemu.log) ]]; then
      echo "PASS: reach $2" >&2
      break
    fi

    sleep 1
    abort_in_num_seconds=$(( $abort_in_num_seconds - 1 ))
  done
}

reach_stage 10 "stage:boot"
reach_stage 60 "stage:network"
./bin/stprov local run --ip 127.0.0.1 -p 2009 --otp sikritpassword | tee saved/stprov.log
reach_stage 3 "stage:shutdown"

got=$(grep hostname saved/stprov.log | cut -d'=' -f2)
if [[ "$got" != "example.org" ]] then
  echo "FAIL: wrong hostname in stprov.log ($got)" >&2
  exit 1
fi
echo "PASS: stprov.log hostname"

fingerprint=$(grep fingerprint saved/stprov.log | cut -d'=' -f2)
virt-fw-vars -i saved/OVMF_VARS.fd --output-json saved/efivars.json

got=$(cat saved/efivars.json | jq -r '.variables[] | select(.name == "STHostName") | .data' | tr a-f A-F | basenc --base16 -d)
if [[ "$got" != "example.org" ]]; then
  echo "FAIL: wrong hostname in EFI NVRAM ($got)" >&2
  exit 1
fi
echo "PASS: EFI-NVRAM hostname"

cat saved/efivars.json | jq -r '.variables[] | select(.name == "STHostKey") | .data' | tr a-f A-F | basenc --base16 -d > saved/hostkey
chmod 600 saved/hostkey
got=$(ssh-keygen -lf saved/hostkey | cut -d' ' -f2)
if [[ "$got" != "$fingerprint" ]]; then
  echo "FAIL: wrong fingerprint for key in EFI NVRAM ($got)" >&2
  exit 1
fi
echo "PASS: EFI-NVRAM hostkey"

cat saved/efivars.json | jq -r '.variables[] | select(.name == "STHostConfig") | .data' | tr a-f A-F | basenc --base16 -d | jq > saved/hostcfg.json
got=$(cat saved/hostcfg.json | jq '.ospkg_pointer')
if [[ "$got" != "\"https://example.org/ospkg.json\"" ]]; then
  echo "FAIL: wrong URL in EFI NVRAM host config ($got)" >&2
  exit 1
fi
echo "PASS: EFI-NVRAM host config (URL-check only)"
