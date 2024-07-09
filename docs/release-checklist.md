# Releases checklist

This document is intended for maintainers that make releases.

## Making a release

  - [ ] All direct dependencies are up to date, or are explicitly kept at older
    versions we want.  Look for updates with `go list -m -u example.org/MODULE`.
  - [ ] All files in docs/ looks reasonable on a read through.
  - [ ] RELEASES.md is up-to-date, see expectations below.
  - [ ] README.md is up-to-date.
  - [ ] The copy-pasted parts of the stprov manual is up-to-date, see
    [instructions](./stprov-manual.README) on what needs to be done.
  - [ ] All links in the [stprov manual](./stprov-manual) and [stprov system
    document](./stprov-system.md) are consistent and pointing at the reference
    specifications that are currently being implemented (with commits or tags).
    Also check the links that contain versions in the [README](../README.md).
  - [ ] Reference specifications and their versions are listed in the NEWS file.
  - [ ] All release tests pass, see [test docs](./testing-stprov.md).  You may
    need to create a new intermediate tag for stprov before doing these tests.
  - [ ] The interoperability-tested versions of stprov, stboot, stmgr are listed
    in the NEWS file.
  - [ ] Finalize the NEWS file, see expectations below.  In the MR that bumps
    the NEWS version, ensure to also set the same version in stprov's manual.
  - [ ] Create a signed tag.  Usually, this means incrementing the third number
    for the most recent tag that was used during interoperability testing.
  - [ ] Send an announcement email

## RELEASES-file 

  - [ ] It is specified what in the repository is released and supported.
  - [ ] The overall release process is described, e.g., where are releases
    announced, how often do we make releases, what type of releases, etc.
  - [ ] The expectation we as maintainers have on users is described.
  - [ ] The expectations users can have on us as maintainers is described, e.g.,
    what we intend to (not) break in the future or any relevant pointers on how
    we ensure that things are "working".

## NEWS-file 

  - [ ] The previous NEWS entry is for the previous release.
  - [ ] It is explained what changed since the previous release.
  - [ ] There are detailed instructions on how to upgrade on breaking changes,
    listed under the section named "Incompatible changes".
  - [ ] Interoperable repositories and tools are listed with commits or tags.
  - [ ] Implemented reference specifications are listed with commits or tags.

## Announcement email template

```
The ST team is happy to announce a new release of the stprov software,
tag vX.X.X, which succeeds the previous release at tag vY.Y.Y.  The
source code for this release is available from the git repository:

  git clone -b vX.X.X https://git.glasklar.is/system-transparency/core/stprov.git

Authoritative ST release signing keys are published at

  https://www.system-transparency.org/keys/

and the tag signature can be verified using the command

  git -c gpg.format=ssh \
      -c gpg.ssh.allowedSignersFile=allowed-ST-release-signers \
      tag --verify vX.X.X

The expectations and intended use of the stprov software is documented
in the repository's RELEASES file.  This RELEASES file also contains
more information concerning the overall release process, see:

  https://git.glasklar.is/system-transparency/core/stprov/-/blob/vX.X.X/RELEASES.md

Learn about what's new in a release from the repository's NEWS file.  An
excerpt from the latest NEWS-file entry is listed below for convenience.

If you find any bugs, please report them on the System Transparency
discuss list or open an issue on GitLab in the stprov repository:

  https://lists.system-transparency.org/mailman3/postorius/lists/st-discuss.lists.system-transparency.org/
  https://git.glasklar.is/system-transparency/core/stprov/-/issues
  system-transparency-core-stprov-issues@incoming.glasklar.is

Cheers,
The ST team

<COPY-PASTE EXCERPT OF LATEST NEWS FILE ENTRY HERE>
```
