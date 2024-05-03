#!/bin/bash

#
# Script that creates an ISO with stprov configured for easy use in Glasklar's
# test lab.  It is assumed that qemu.sh has run successfully before this script.
#
#   Usage: SINGLE_TEST=0 ./qemu.sh && ./supermicro-x11scl.sh
#

set -eu
cd "$(dirname "$0")"

if [[ ! -x cache/bin/u-root ]]; then
    echo "FAIL: run qemu.sh before using this build script" >&2
    exit 1
fi

if ! command -v stmgr >/dev/null 2>&1; then
    echo "FAIL: stmgr is not installed" >&2
    exit 1
fi

# Build stprov with default values that make sense for the supermicro test
# server in Glasklar's lab as well as the published stimages.
make -C ../\
    DEFAULT_ALLOWED_NETWORKS=91.223.231.1/24\
    DEFAULT_TEMPLATE_URL=https://st.glasklar.is/st/qa/qa-debian-bookworm-amd64.json\
    DEFAULT_DOMAIN=st.glasklar.is\
    DEFAULT_DNS=9.9.9.9,149.112.112.112\
    DEFAULT_BONDING_MODE=802.3ad
mv ../stprov cache/bin/

# It appears that u-root's init doesn't mount EFI variables correctly when using
# modules.  So, we will need to mount it on our own after u-root's init exited.
echo "#!/bin/elvish" > build/uinitcmd.sh
echo "mount -t efivarfs none /sys/firmware/efi/efivars" >> build/uinitcmd.sh

./cache/bin/u-root\
    -o build/stprov.cpio\
    -uroot-source=cache/u-root\
    -uinitcmd="/bin/sh /bin/uinitcmd.sh"\
    -files build/uinitcmd.sh:bin/uinitcmd.sh\
    -files build/1-modules.conf:lib/modules-load.d/1-modules.conf\
    -files build/modules/usr/lib/modules:lib/modules\
    -files cache/bin/stprov:bin/stprov\
    -files isrgrootx1.pem:/etc/trust_policy/tls_roots.pem\
    cache/u-root/cmds/core/{init,elvish,shutdown,cat,cp,dd,echo,grep,hexdump,ls,mkdir,mv,ping,pwd,rm,wget,wc,ip,mount}

rm -f build/stprov.iso
gzip -f build/stprov.cpio
stmgr uki create -format iso\
    -kernel build/kernel.vmlinuz\
    -initramfs build/stprov.cpio.gz\
    -cmdline '-- -v'\
    -out build/stprov.iso

echo "INFO: created build/stprov.iso" >&2
