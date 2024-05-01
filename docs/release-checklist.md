# Releases checklist

This document is intended for maintainers that make releases.

## Making a release

  - [ ] The README.md, RELEASES.md, and MAINTAINER files are up-to-date
  - [ ] The copy-pasted parts of the stprov manual is up-to-date, see
    [instructions](./stprov-manual.README)
  - [ ] All links in the [stprov manual](./stprov-manual) and [stprov system
    document](./stprov-system.md) are consistent and pointing at the reference
    specifications that are currently being implemented (with commits or tags).
  - [ ] List reference specifications and their versions in the NEWS file.
  - [ ] All release tests pass, see [test docs](./testing-stprov.md)
  - [ ] List the interoperability-tested versions of stboot and stmgr in the
    NEWS file.  The versions to use should match the above release testing.
  - [ ] After finalizing the release documentation (in particular the NEWS
    file), create a new tag.  Usually, this means incrementing the third number
    for the most recent tag that was used during our interoperability tests.
  - [ ] Sign the release tag and send an announcement email

## RELEASES-file 

  - [ ] What in the repository is released and supported
  - [ ] The overall release process is described, e.g., where are releases
    announced, how often do we make releases, what type of releases, etc.
  - [ ] The expectation we as maintainers have on users is described
  - [ ] The expectations users can have on us as maintainers is
    described, e.g., what we intend to (not) break in the future or any
    relevant pointers on how we ensure that things are "working".

## NEWS-file 

  - [ ] The previous NEWS entry is for the previous release
  - [ ] Explain what changed
  - [ ] Detailed instructions on how to upgrade on breaking changes, listed
    under the section named "Incompatible changes"
  - [ ] List interoperable repositories and tools, specify commits or tags
  - [ ] List implemented reference specifications, specify commits or tags

Note that the NEWS file is created manually from the git-commit history.

## Announcement email template

```
The ST team is happy to announce a new release of the stprov software,
tag vX.X.X, which succeeds the previous release at tag vY.Y.Y.  The
source code for this release is available from the git repository:

  git clone -b vX.X.X
  https://git.glasklar.is/system-transparency/core/stprov.git

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
  mailto:system-transparency-core-stprov-issues@incoming.glasklar.is

Cheers,
The ST team

<COPY-PASTE EXCERPT OF LATEST NEWS FILE ENTRY HERE>
```
