# System Transparency provisioning tool

This repository provides `stprov`, a System Transparency provisioning tool that
can be used for writing necessary [stboot][] configurations to EFI-NVRAM.

[stboot]: https://git.glasklar.is/system-transparency/core/stboot/

## Building

Clone and build using `make`.  For example:

    $ git clone https://git.glasklar.is/system-transparency/core/stprov.git
    $ cd stprov
    $ make stprov DEFAULT_USER=alice

See [Makefile](./Makefile) for all options that can be customized.  If the
pre-defined defaults are good enough, you may use Go's tooling directly:

    $ go install system-transparency.org/stprov/cmd/stprov@latest

## Provisioning

One way to use `stprov` for platform provisioning is by building an OS package
that contains it.  This OS package can then be written to the `stboot`
initramfs, and be loaded by default using so called [provisioning mode][].  In
other words, on missing EFI-NVRAM configuration the stboot image would drop into
a provisioning environment where the `stprov remote` program is available.
Another way to achieve the same thing is to have a separate image just for
provisioning, e.g., built as an ISO that can be mounted on the platform.

The [stprov CI](./integration/ci-images.yml) contains examples of how to build
provisioning OS packages and separate ISOs that use u-root's shell environment.

[provisioning mode]: https://git.glasklar.is/system-transparency/core/stboot/-/blob/v0.4.1/docs/stboot-system.md?ref_type=tags#host-configuration

## Development

### Contributing

You are encouraged to [file issues][] and open [merge requests][].  For
information on how we collaborate in GitLab, see [accepted proposal][] that
describes this.

If you are a first-time contributor, please review the stprov
[LICENSE](./LICENSE) and copyright in the [AUTHORS](./AUTHORS) file.  Append
your name to the list of authors at the bottom in a separate commit.

[file issues]: https://git.glasklar.is/system-transparency/core/stprov/-/issues
[merge requests]: https://git.glasklar.is/system-transparency/core/stprov/-/merge_requests
[accepted proposal]: https://git.glasklar.is/system-transparency/project/documentation/-/blob/main/proposals/2023-09-25-gitlab-roles-and-conventions.md

### Testing

Our [CI configuration](./gitlab-ci) builds the `stprov` program, runs (most)
unit tests, and performs a QEMU integration test.  The QEMU integration test
contains a working example of `stprov remote` and `stprov local`.  See the
[testing stprov](./docs/testing-stprov.md) document for further details.

Please make sure that all CI tests pass before requesting review.

There are a few additional unit tests that are not running in our CI.  These
tests write to the system's EFI-NVRAM - **be warned** - and require root
privileges.  Feel free to skip this unless your changes concerned EFI NVRAM.

    $ TEST_CLOBBER_EFI_NVRAM=y go test ./...

Add `sudo` to the above if you want EFI-NVRAM read/writes to succeed.

### Documentation

If you're updating the stprov command-line interface, please update the stprov
manual as well.  See instructions on how [here](./docs/stprov-manual.md.README).

If you're significantly changing or extending stprov's behavior, consider if an
update to the [system documentation](./docs/stprov-system.md) is appropriate.

### Commits

We enforce conventional commits using [commitlint][].  The expected git-commit
message format is as follows:

    <type>: <Description starting with a capital letter>
    
    [optional body]

    [optional footer(s)]

Pick `<type>` from the following list:

  - **build:** changes that are tooling/building related
  - **chore:** housekeeping, dependency management, go.mod, etc.
  - **ci:** continuous integration, workflows, etc.
  - **docs:** README, .md files, documentation in code, etc.
  - **feat:** source code changes introducing new functionality
  - **fix:** bug fixes, no new functionality
  - **refactor:** source code changes without changing behavior
  - **revert:** used if something needs to be reverted
  - **test:** e.g., when adding unit tests and QEMU tests that were not
    committed as part of a fix or feat commit

Note that we are not picky about an MR containing multiple conventional commits
as a result of review.  Keep all commits or rebase based on what feels easiest.

[commitlint]: https://commitlint.js.org/

## Contact

  - IRC room `#system-transparency` @ OFTC.net
  - Matrix room `#system-transparency` which is bridged with IRC
  - System Transparency [discuss list][]

[discuss list]: https://lists.system-transparency.org/mailman3/postorius/lists/st-discuss.lists.system-transparency.org/
