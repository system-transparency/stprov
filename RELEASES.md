# Releases of stprov

## What is being released?

The following program is released and supported:

  - `cmd/stprov`

New releases are announced on the System Transparency [announce list][].  What
changed in each release is documented in a [NEWS file](./NEWS).  The NEWS file
also specifies which other System Transparency components are known to be
interoperable, as well as which reference specifications are being implemented.

Note that a release is simply a git-tag specified on our mailing list.  The
source for this git-tag becomes available on the repository's release page:

  https://git.glasklar.is/system-transparency/core/stprov/-/releases

The stprov Go module is **not** considered stable before a v1.0.0 release.  By
the terms of the LICENSE file you are free to use this code "as is" in almost
any way you like, but for now, we support its use _only_ via the above program.
We don't aim to provide any backwards-compatibility for internal interfaces.

We encourage use of `stprov` to provision new platforms.  It is up to you to
make `stprov` available to the platform, e.g., as a provisioning OS package.

[announce list]: https://lists.system-transparency.org/mailman3/postorius/lists/st-announce.lists.system-transparency.org/

## What release cycle is used?

We make feature releases when something new is ready.  As a rule of thumb,
feature releases will not happen more often than once per month.

In case critical bugs are discovered, we intend to provide bug-fix-only updates
for the latest release in a timely manner.  Backporting bug-fixes to older
releases than the latest one will be considered on a case-by-case basis.  Such
consideration is most likely if the latest feature release is very recent or
upgrading to it is particularly disruptive due to the changes that it brings.

## Upgrading

You are expected to upgrade linearly from one advertised release to the next
advertised release, e.g., from v0.1.1 to v0.2.1.  We strive to make such linear
upgrades easy and well-documented to help with forward-compatibility.  Any
complications that are caused by changed reference specifications, command-line
flags, or similar will be clearly outlined in the [NEWS files](./NEWS).  Pay
close attention to the "Breaking changes" section for these migration notes.

Downgrading is in general not supported.  It is further assumed that `stprov`
clients and servers use the same release.  Mixed releases are not tested.

## Expected changes in upcoming releases

  - Changes to the `stprov` command-line interface are likely to happen (as part
    of refactoring).
  - Any changes to the System Transparency reference specifications will be
    implemented.  This could for example affect the format or configuration
    being stored in EFI-NVRAM.
  - New provisioning features, such as remote attestation.
