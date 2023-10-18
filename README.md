# System Transparency provisioning tool

This repository provides `stprov`, a System Transparency provisioning tool that
can be used for writing necessary [stboot][] configurations to EFI-NVRAM.  A
future version of `stprov` will likely add additional provisioning features.

[stboot]: https://git.glasklar.is/system-transparency/core/stboot/

## Building

Clone and build using `make`.  For example:

    $ git clone https://git.glasklar.is/system-transparency/core/stprov.git
    $ cd stprov
    $ make stprov DEFAULT_USER=ninja

See [Makefile](./Makefile) for all options that can be customized.  If the
pre-defined defaults are good enough, you may use Go's tooling directly:

    $ go install system-transparency.org/stprov/cmd/stprov@latest

## Provisioning

One way to use `stprov` for platform provisioning is by building a minimal OS
package that contains it.  This OS package can then be written to the `stboot`
initramfs, and be loaded by default using so called _provisioning mode_.

In other words, on missing EFI-NVRAM configuration the `stboot` ISO would boot
into a provisioning environment where the `stprov remote` program is available.

TODO: expand this subsection with an appropriate amount of detail, and/or link
to further documentation related to this.  We're also missing a good overview.

## Development

### Contributing

You are encouraged to file issues and open merge requests.  For more information
on how we collaborate in GitLab, see [accepted proposal][] that describes this.

[accepted proposal]: https://git.glasklar.is/system-transparency/project/documentation/-/blob/main/proposals/2023-09-25-gitlab-roles-and-conventions.md

### Testing

Our [CI configuration](./gitlab-ci) builds the `stprov` program, runs (most)
unit tests, and performs a QEMU integration test.  The QEMU integration test
contains a working example of `stprov remote static` and `stprov local`.

Please make sure that all CI tests pass.

There are a few additional unit tests that are not running in our CI.  These
tests write to the system's EFI-NVRAM - **be warned** - and require root
privileges.

    $ TEST_CLOBBER_EFI_NVRAM=y go test ./...

Add `sudo` to the above if you want EFI-NVRAM read/writes to succeed.

### Commits

We are currently trying to enforce conventional commits using `commitlint`.  The
expected git-commit message format is as follows:

    <type>: <Description starting with a capital letter>
    
    [optional body]
    
    [optional footer(s)]

For more information about the available types, see [commitlint proposal][].

**Note:** `commitlint` runs in our CI pipelines.  Local installation is
optional.

[commitlint proposal]: https://git.glasklar.is/system-transparency/project/documentation/-/blob/main/proposals/2023-01-19-commitlint-proposal.md

## Contact

  - IRC room `#system-transparency` @ OFTC.net
  - Matrix room `#system-transparency` which is bridged with IRC
  - System Transparency [mailing list][]

[mailing list]: https://lists.sigsum.org/mailman3/postorius/lists/system-transparency.lists.system-transparency.org/
