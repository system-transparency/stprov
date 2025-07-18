# Testing of stprov

This file documents how stprov is tested:

  1. Unit tests: runs locally in any Go environment (`go test ./...`).
  2. QEMU tests: runs locally provided that needed dependencies are available,
     see `.gitlab-ci.yml`.
  3. Hardware tests: the kernel and initramfs produced in (2) is used to create
     a bootable ISO, which is then booted on real hardware.

Unit tests and (most) QEMU tests are run in the stprov CI.  Real hardware tests
are run manually and less frequently, mainly as part of release testing on a
Supermicro X11SCL-F in Glasklar's lab setup.

Before cutting an stprov release, we check that (1)--(2) work locally and that
(3) additionally works in Glasklar's lab setup.  The remainder of this document
describes the QEMU tests in more detail, and what manual tests are done on real
hardware using an stprov ISO.

## QEMU tests

The `integration/` subdirectory contains a script for running stprov with
different invocations in QEMU.  The core/stboot repository contains a QEMU test
that checks if provision mode works with stprov as an OS package.

### integration/qemu.sh

This script performs all provisioning steps by running `stprov local` on the
local system and `stprov remote` in QEMU.  This includes `stprov remote` setting
up the network, writing a host configuration and hostname to EFI NVRAM,
listening for further configuration from `stprov local`, and finally writing the
received secure boot keys and a generated SSH hostkey to EFI NVRAM.  The script
asserts that the expected values are put into the (emulated) EFI NVRAM, that the
provisioned secure boot keys work in QEMU on subsequent boots (i.e., a signed
stprov ISO boots correctly), and that the expected output is shown on the
`stprov local` console.  The script has a non-zero exit code on failures.

The same test is performed multiple times.  What is changed each time is the way
in which `stprov remote` is invoked.  This ensures the `static` and `dhcp`
subcommands get tested, including their many variations with default options.
One of these tests use all compile-time options in stprov's Makefile.

Please note that this script performs happy-path tests, simply by trying to run
as many parts of stprov as possible to see if expected outputs are obtained.

Run the script and all its tests as follows:

    $ ./integration/qemu.sh

Refer to the script for further details and options that, e.g., allow running
just a single failing test or no test at all (to debug in QEMU interactively).

**Warning:** bonding is not tested in QEMU.

### stboot smoke test

There is an automatic CI test that checks interoperability with stboot, see
<https://git.glasklar.is/system-transparency/core/stboot/-/blob/main/integration/qemu-provision.sh>.

Ensure the stboot CI passes for the appropriate versions of stprov, stboot, and
stmgr.  For stprov and stmgr versions, see stboot's `integration/go.mod` file.

## Testing on real hardware

The real hardware tests are performed on a server in the Glasklar lab.  Follow
the [test server][] instructions to mount `stprov.iso` which is produced by the
stprov CI, or build the ISO locally as follows:

    $ SINGLE_TEST=0 ./integration/qemu.sh && ./integration/supermicro-x11scl.sh

`integration/supermicro-x11scl.sh` uses the `stmgr` tool.  Ensure the same
version as the stboot smoke test is used, see `integration/go.mod`.

The instructions for three manual tests will now be outlined.

  - DHCP network configuration, bonding disabled
  - Static network configuration, bonding disabled
  - Static network configuration, bonding enabled (802.3ad)

The DHCP configuration also includes instructions for testing the local-remote
ping pongs.  So, install stprov on the local system, and ensure that the local
system is able to reach the test server (ask someone if you don't know how).

Note: `integration/supermicro-x11scl.sh` sets some defaults so we can type less
in the poor BMC interface.  For example, an OS package URL, two DNS servers, and
the allowed networks are set.  See the Makefile invocation for further details.

During the final release testing, reset the server between each test.  For
quicker sanity checks, you may run all the below tests without such resets.

[test server]: https://git.glasklar.is/glasklar/services/bootlab/-/blob/main/stime.md

### DHCP network configuration, bonding disabled

Enter the UEFI menu (F11, Enter Setup).

Put the server into Secure Boot Setup Mode (Security -> Secure Boot -> Key
Management -> Reset To Setup Mode).

Save and exit (F4).

Use `ip a` to find the name of the interface that can be statically configured.
The [test server][] documentation currently says it is `3c:ec:ef:29:60:2b`.

If the interface name is `eth1`, run:

    # stprov remote dhcp -h qa1 -I eth1

**Troubleshooting:** not getting an IP address? Check `systemctl status udhcpd`
on tee, might need a *restart*.  This is a temporary fix for
[bootlab#5](https://git.glasklar.is/glasklar/services/bootlab/-/issues/5).

Expect to see that the HEAD request on the OS package succeeds.

Determine the configured IP address:

    # ip a

Await further configuration from `stprov local`:

    # stprov remote run -o sikritpassword

Run `stprov local` and specify the build's Secure Boot keys, see `saved/`.

    $ stprov local run -o sikritpassword -i SERVER_ADDR --pk PK.auth --kek KEK.auth --db db.auth --dbx dbx.auth

Expect to see prints saying that the received Secure Boot keys were
provisioned, and that the next boot will go straight into the UEFI menu.

Expect to see that the same entropy is printed in both terminals.

Expect that the EFI variables for hostname, host configuration, and SSH hostkey
have been populated.  Eyeball that these EFI variables look reasonable.  (We
mainly want to be sure that the writes succeeded to non-emulated EFI NVRAM.)

    # cat /sys/firmware/efi/efivars/STHost*

Reboot (shutdown -r).

In the UEFI menu's Secure Boot options, toggle Secure Boot to "Enabled".

Save and exit (F4).

Expect boot to work.  Check with `hexdump` that SetupMode is `00` and
SecureBoot is `01`.  You will find these EFI variables in
`/sys/firmware/efi/efivars/`.  The first four bytes are EFI metadata.

Reboot and disable secure boot before doing the remaining tests.

### Configure network with static IP address

Use the same interface as in the DHCP test.  Look at the [test server][]
documentation to learn the static IP address, network prefix, and gateway.

    # stprov remote static -h qa2 -I eth1 -i 91.223.231.242/29 -g 91.223.231.241

Expect to see the HEAD request on the OS package succeed again.  Expect to see
that the hostname and host configuration EFI variables changed appropriately.

### Configure network with static IP address and bonding

Use `ip a` to find the name of the interfaces that can be bonded.  The [test
server][] documentation says the MAC addresses of these interfaces are
`00:0a:f7:2a:59:bc` and `00:0a:f7:2a:59:bd`.  Look at the [test server][]
documentation to learn the static IP address, network prefix, and gateway.

    # stprov remote static -h qa3 -b eth2 -b eth3 -i 91.223.231.242/29 -g 91.223.231.241

Expect to see the HEAD request on the OS package succeed again.  Expect to see
that the hostname and host configuration EFI variables changed appropriately.
