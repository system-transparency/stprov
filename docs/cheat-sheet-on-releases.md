# Cheat-sheet on releases

This document is intended for maintainers that make releases.

## Checklist

Making a release:

  - [ ] README, RELEASES, and MAINTAINER files are up-to-date
  - [ ] Test the procedure of upgrading from the previous release (if any)
  - [ ] Test and document which other System Transparency components are
    interoperable.  We currently test that stboot provision mode works, which
    means stprov is able to produce EFI-NVRAM configurations for network-boot.
  - [ ] Test/check that the tutorial, how-to, and explanation sections of
    docs.system-transparency.org (branch main) are up-to-date for stprov
  - [ ] After finalizing the release documentation (in particular the NEWS
    file), create a new tag.  Usually, this means incrementing the third number
    for the most recent tag that was used during our interoperability tests.
  - [ ] Create release page
  - [ ] Send announcement email

## RELEASES-file checklist

  - [ ] What in the repository is released and supported
  - [ ] The overall release process is described, e.g., where are releases
    announced, how often do we make releases, what type of releases, etc.
  - [ ] The expectation we as maintainers have on users is described
  - [ ] The expectations users can have on us as maintainers is
    described, e.g., what we intend to (not) break in the future or any
    relevant pointers on how we ensure that things are "working".

## NEWS-file checklist

  - [ ] The previous NEWS entry is for the previous release
  - [ ] Explain what changed
  - [ ] Detailed instructions on how to upgrade on breaking changes
  - [ ] List interoperable repositories and tools, specify commits or tags
  - [ ] List implemented reference specifications, specify commits or tags

## Announcement email template

```
The ST team is happy to announce a new release of the stprov software,
tag v0.2.1, which succeeds the previous release at tag v0.1.1.  The
source code is available as an archive on our GitLab's release page:

  https://git.glasklar.is/system-transparency/core/stprov/-/releases

Alternatively, you can checkout the git-repository:

  git clone -b v0.2.1 https://git.glasklar.is/system-transparency/core/stprov.git

Or install using Go's tooling:

  go install system-transparency.org/stprov@v0.2.1

The expectations and intended use of the stprov software is documented
in the repository's RELEASES file.  This RELEASES file also contains
more information concerning the overall release process, see:

  https://git.glasklar.is/system-transparency/core/stprov/-/blob/main/RELEASES.md

Learn about what's new in a release from the repository's NEWS file.  An
excerpt from the latest NEWS-file entry is listed below for convenience.

If you find any bugs, please report them on the System Transparency
discuss list or open an issue on GitLab in the stprov repository:

  https://lists.system-transparency.org/mailman3/postorius/lists/st-discuss.lists.system-transparency.org/
  https://git.glasklar.is/system-transparency/core/stprov/-/issues

Cheers,
The ST team

<COPY-PASTE EXCERPT OF LATEST NEWS FILE ENTRY HERE>
```