# Releases of stprov

## What is being released?

The following program is released and supported:

  - `cmd/stprov`

New releases are announced on the System Transparency [announce list][].  What
changed in each release is documented in a [NEWS file](./NEWS).  The NEWS file
also specifies which other System Transparency components are known to be
interoperable, as well as which reference specifications are being implemented.

Note that a release is simply a signed git-tag specified on our mailing list,
accessed from the [stprov repository][].  To verify tag signatures, get the
`allowed-ST-release-signers` file published at the [signing-key page][], and
verify the tag `vX.Y.Z` using the following command:

    git -c gpg.format=ssh -c gpg.ssh.allowedSignersFile=allowed-ST-release-signers tag --verify vX.Y.Z

The above configuration can be stored permanently using `git config`.

The stprov Go module is **not** considered stable before a v1.0.0 release.  By
the terms of the LICENSE file you are free to use this code "as is" in almost
any way you like, but for now, we support its use _only_ via the above program.
We don't aim to provide any backwards-compatibility for internal interfaces.

We encourage use of stprov to provision new platforms.  Make stprov available to
the platform as a provisioning OS package or a separate image.  The stprov
[README](./README.md#provisioning) refers to some examples related to this.

[announce list]: https://lists.system-transparency.org/mailman3/postorius/lists/st-announce.lists.system-transparency.org/
[stprov repository]: https://git.glasklar.is/system-transparency/core/stprov
[signing-key page]: https://www.system-transparency.org/keys/

## Release testing

See the [test documentation](./docs/testing-stprov.md) for information on how
stprov is tested with unit tests, in QEMU, and on real hardware.

## What release cycle is used?

We make feature releases when something new is ready.  As a rule of thumb,
feature releases will not happen more often than once per month.

In case critical bugs are discovered, we intend to provide bug-fix-only updates
for the latest release in a timely manner.  Backporting bug-fixes to older
releases than the latest one will be considered on a case-by-case basis.

## Upgrading

We strive to make stprov upgrades easy and well-documented.  Any complications
that are caused by changed configuration syntax, command-line flags, or similar
will be clearly outlined in the [NEWS file](./NEWS).  Pay close attention to
the "Incompatible changes" section before upgrading to a new version.

Downgrading is in general not supported.  Mixing stprov-local and stprov-remote
version is in general also not supported.
