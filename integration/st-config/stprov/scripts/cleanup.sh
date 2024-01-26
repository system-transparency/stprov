#! /bin/bash
set -eu

# Clean up rootfs.
# Remove everything that can not be reproducably built.
# Remove to save space too.

ROOTDIR="$1"; shift
[[ "$ROOTDIR" = "" || "$ROOTDIR" = "/" ]] && { echo "$0: bad ROOTDIR: $ROOTDIR"; exit 1; }

# Remove systemd machine id
rm -f ${ROOTDIR}/etc/machine-id

# Remove ssh keys
rm -f ${ROOTDIR}/etc/ssh/ssh_host*

# Remove ldconfig cache
rm -f ${ROOTDIR}/var/cache/ldconfig/aux-cache

# Remove systemd catalog file
rm -rf ${ROOTDIR}/var/lib/systemd/catalog/database

# Clear installation log
find ${ROOTDIR}/var/log -type f | while read -r line ; do rm -f "$line" ; done

# Remove pycache
find ${ROOTDIR} -type d -name __pycache__ | while read -r line ; do rm -rf "$line" ; done

# Remove initrd as it's not needed nor is it reproducible
rm -rf ${ROOTDIR}/var/lib/initramfs-tools/*
rm -f ${ROOTDIR}/boot/initrd.img*

# Remove kernels and debs, to save space
rm -f ${ROOTDIR}/boot/vmlinuz-*
rm -f ${ROOTDIR}/var/cache/apt/archives/*.deb

# Remove .git directories
find ${ROOTDIR} -type d -name .git | while read -r line; do rm -rf "$line"; done
