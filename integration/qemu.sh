#!/bin/bash

#
# A simple script that takes stprov for a trial run with qemu.  Before running,
# you may need to install a few dependencies.  See details in ../gitlab-ci.yml.
#
# Usage: ./qemu.sh
#

set -eu
trap clean_up EXIT

BUILD=$(realpath "$(dirname "$0")"/build)
SAVED=$(realpath "$(dirname "$0")"/saved)

qemu_pid=""
function clean_up() {
  set +e
  ps -p $qemu_pid > /dev/null && kill $qemu_pid
}

mkdir -p "$BUILD" "$SAVED"

###
# Build
###
GOPATH="$BUILD"/go go install ../cmd/stprov
[[ -d "$BUILD"/u-root ]] ||
  git clone --depth 1 https://github.com/u-root/u-root "$BUILD"/u-root &&
  pushd "$BUILD"/u-root && GOPATH="$BUILD"/go go install && popd

url="https://git.glasklar.is/system-transparency/core/system-transparency/-/raw/main/contrib/linuxboot.vmlinuz"
[[ -f "$BUILD"/kernel.vmlinuz ]] || curl -L "$url" -o "$BUILD"/kernel.vmlinuz

"$BUILD"/go/bin/u-root\
  -o "$BUILD"/stprov.cpio\
  -uroot-source="$BUILD"/u-root\
  -uinitcmd="/bin/sh /bin/uinitcmd.sh"\
  -files "$BUILD"/go/bin/stprov:bin/stprov\
  -files uinitcmd.sh:bin/uinitcmd.sh\
  "$BUILD"/u-root/cmds/core/{init,elvish}

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
    cp /usr/share/"$str"/OVMF_VARS.fd "$SAVED"/OVMF_VARS.fd
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
qemu-system-x86_64 -nographic\
  -m 512M -M q35 -rtc base=localtime\
  -net user,hostfwd=tcp::2009-:2009 -net nic\
  -object rng-random,filename=/dev/urandom,id=rng0\
  -device virtio-rng-pci,rng=rng0\
  -drive if=pflash,format=raw,readonly=on,file="$ovmf_code"\
  -drive if=pflash,format=raw,file="$SAVED"/OVMF_VARS.fd\
  -kernel "$BUILD"/kernel.vmlinuz\
  -initrd "$BUILD"/stprov.cpio\
  -append "console=ttyS0" >"$SAVED"/qemu.log &
qemu_pid=$!

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

    if [[ ! -z $(grep "$2" "$SAVED"/qemu.log) ]]; then
      echo "PASS: reach $2" >&2
      break
    fi

    sleep 1
    abort_in_num_seconds=$(( $abort_in_num_seconds - 1 ))
  done
}

reach_stage 10 "stage:boot"
reach_stage 60 "stage:network"
"$BUILD"/go/bin/stprov local run --ip 127.0.0.1 --otp sikritpassword | tee "$SAVED"/stprov.log
reach_stage 3 "stage:shutdown"

hostname=$(grep hostname "$SAVED"/stprov.log | cut -d'=' -f2)
fingerprint=$(grep fingerprint "$SAVED"/stprov.log | cut -d'=' -f2)
virt-fw-vars -i "$SAVED"/OVMF_VARS.fd --output-json "$SAVED"/efivars.json

got=$(cat "$SAVED"/efivars.json | jq -r '.variables[] | select(.name == "STHostName") | .data' | base16 -d)
if [[ "$got" != "$hostname" ]]; then
  echo "FAIL: wrong hostname in EFI NVRAM ($got)" >&2
  exit 1
fi
echo "PASS: EFI-NVRAM hostname"

cat "$SAVED"/efivars.json | jq -r '.variables[] | select(.name == "STHostKey") | .data' | base16 -d > "$SAVED"/hostkey
chmod 600 "$SAVED"/hostkey
got=$(ssh-keygen -lf "$SAVED"/hostkey | cut -d' ' -f2)
if [[ "$got" != "$fingerprint" ]]; then
  echo "FAIL: wrong fingerprint for key in EFI NVRAM ($got)" >&2
  exit 1
fi
echo "PASS: EFI-NVRAM hostkey"

cat "$SAVED"/efivars.json | jq -r '.variables[] | select(.name == "STHostConfig") | .data' | base16 -d | jq > "$SAVED"/hostcfg.json
got=$(cat "$SAVED"/hostcfg.json | jq '.ospkg_pointer')
if [[ "$got" != "\"https://example.org/ospkg.json\"" ]]; then
  echo "FAIL: wrong URL in EFI NVRAM host config ($got)" >&2
  exit 1
fi
echo "PASS: EFI-NVRAM host config (URL-check only)"
