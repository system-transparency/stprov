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

# Build stprov with default values that make sense for the supermicro test
# server in Glasklar's lab as well as the published stimages.
make -C ../ \
    DEFAULT_TEMPLATE_URL=https://st.glasklar.is/st/qa/qa-debian-trixie-amd64.json \
    DEFAULT_DNS=9.9.9.9,149.112.112.112
mv ../stprov cache/bin/

# It appears that u-root's init doesn't mount EFI variables correctly when using
# modules.  So, we will need to mount it on our own after u-root's init exited.
{
    echo "#!/bin/sh"
    echo "mount -t efivarfs none /sys/firmware/efi/efivars"
    # Get our interactive shell running, working around
    # u-root regression, see https://github.com/u-root/u-root/issues/3645
    echo "/bin/sh"
} > build/uinitcmd.sh

# With "-go-build-tags goshliner" we avoid the buggy bubbline, and get
# at least arrow-up for editing and running a previous command
# (goshsmall does not have that).

# So we have to use goshsmall, with no line-editing whatsoever! (also
# no arrow-up for shell history). We can't use goshliner, because it's
# broken on 0.16.0 (released February 2026) due to issue in gosh. The
# problem is that a program launched by the shell (like stprov) cannot
# read any input (like string+RET). Also Ctrl-c to that program also
# gosh itself (resulting in kernel crash, "Attempted to kill init").
# This due to the issue that I filed spring 2025: https://github.com/u-root/u-root/issues/3362.
# Which was fixed in May 2026: https://github.com/u-root/u-root/commit/0c4f1c888bf53e8c5e829584abfcdc35964f7c65
(cd cache/u-root &&
    ../bin/u-root\
    -o ../../build/stprov.cpio\
    -uinitcmd "/bin/sh /bin/uinitcmd.sh"\
    -defaultsh gosh\
    -go-build-tags goshsmall\
    -files ../../build/uinitcmd.sh:bin/uinitcmd.sh\
    -files ../../build/1-modules.conf:lib/modules-load.d/1-modules.conf\
    -files ../../build/modules/usr/lib/modules:lib/modules\
    -files ../bin/stprov:bin/stprov\
    -files ../../isrgrootx1.pem:/etc/trust_policy/tls_roots.pem\
    ./cmds/core/{init,gosh,shutdown,cat,cp,dd,echo,grep,hexdump,ls,mkdir,mv,ping,pwd,rm,wget,wc,ip,mount,ps,more}
)

rm -f build/stprov.iso
gzip -f build/stprov.cpio
go run system-transparency.org/stmgr uki create -format iso\
    -signcert saved/db.pem\
    -signkey saved/db.priv\
    -kernel build/kernel.vmlinuz\
    -initramfs build/stprov.cpio.gz\
    -cmdline '-- -v'\
    -out build/stprov.iso

echo "INFO: created signed build/stprov.iso for quite-nuc" >&2
