# stprov manual

## NAME

stprov---a System Transparency provisioning tool

## SYNOPOSIS

    stprov help
    stprov version
    stprov local <SUBCOMMAND> [Options]
    stprov remote <SUBCOMMAND> [Options]

The local and remote commands accept the subcommand "help".

## VERSION

This manual describes stprov v0.3.5.

## DESCRIPTION

stprov is a tool for provisioning a platform with host-specific configuration.
The provisioned configuration can for example be read by the System Transparency
bootloader stboot and the OS packages that are booted by it.

stprov uses a client-server architecture to help the operator provision the
platform with relatively few keystrokes.  The "local" command represents the
client.  The "remote" part represents the server being provisioned.  The two
commands exchange information with each other using a secure channel.

Use of the remote command is required.  It configures and sanity-checks the
platform's network configuration using the subcommands "static" or "dhcp".

Use of the local command is optional.  The operator may use the remote
subcommand "run" to await further configuration from the local subcommand "run".
In short, the local command provides the remote command with entropy.  The
remote command mixes that into key derivations when provisioning an SSH hostkey.

## COMMANDS

The syntax `-A val | -B val` denotes that option A or option B is required.

The syntax `-A val [-A ...]` denotes that option A can be repeated one or more
times.  Repeated values may be specified with comma-separation: `-A val,val`.

The syntax `{}` denotes a block of commands, only used if needed for clarity.

There are multiple ways to specify the same option.  For example, `-A val` and
`-A=val` are equivalent.  All options have short and long names, see OPTIONS.

    stprov help

      Outputs an overview of available commands.


    stprov version

      Outputs a version string that was set at compile-time.


    stprov local run -o OTP -i IP_ADDR [-p PORT]

      Contributes entropy to stprov remote, which is listening on a given IP
      address (-i) and port (-p).  A one-time password (-o) is used to bootstrap
      HTTPS.  Outputs the following key-value pairs on success:

      fingerprint=<the platform's SSH hostkey fingerprint>
      hostname=<the platform's hostname>
      ip=<the platform's IP address>


    stprov remote run -o OTP [-i IP_ADDR] [-p PORT] [-a ALLOWED_HOST [-a ALLOWED_HOST ...]

      Starts a server on a given IP address (-i) and port (-o), waiting for
      commands from stprov local.  A one-time password (-o) is used to establish
      a mutually authenticated HTTPS connection.  Connections are only accepted
      from the allowed hosts (-a), a repeated option that uses CIDR notation.

      An SSH hostkey is written to EFI NVRAM on success.


    stprov remote dhcp -h HOSTNAME | -H FULL_HOSTNAME
                       -r OSPKG_URL [-r OSPKG_URL ...] [-u USER] [-p PASSWORD]
                       [-m MAC | -I INTERFACE | -w WAIT]
                       [-d DNS [-d DNS ...]]

      Configures the network using DHCP. If none of -m and -I are specified, the
      interface is guessed.

      A host configuration and a hostname is written to EFI NVRAM on success.


    stprov remote static -i HOST_ADDR
                         -h HOSTNAME | -H FULL_HOSTNAME
                         -r OSPKG_URL [-r OSPKG_URL ...] [-u USER] [-p PASSWORD]
                         [-A | -m MAC | -I INTERFACE | {-B | -b INTERFACE [-b INTERFACE ...]} [-M BONDING_MODE]] [-w WAIT]
                         [-g GATEWAY] [-x] [-f]
                         [-d DNS [-d DNS ...]]

      Configures a static network configuration and persist it to EFI-NVRAM.  If
      none of -m and -I are specified, the network interface is guessed.  If -A
      is specified, the interface guessing involves pinging the gateway.  If -B
      is specified, the interface guessing is instead tailored for bonding.

      A host configuration and a hostname is written to EFI NVRAM on success.

## OPTIONS

The options of "stprov local run" are listed below.

    -o, --otp   One-time password to establish a secure connection
    -i, --ip    Listening address (e.g., 10.0.2.10)
    -p, --port  Listening port (Default: 2009)

The options of "stprov remote run" are listed below.

    -o, --otp    One-time password to establish a secure connection
    -i, --ip     Listening address (Default: 0.0.0.0)
    -p, --port   Listening port (Default: 2009)
    -a, --allow  Source IP addresses allowed to connect in CIDR notation
                 (Default: 127.0.0.1/32; can be repeated)

    If the subnet mask is omitted with the -a option, it defaults to "/32"
    (IPv4) or "/128" (IPv6).  E.g., 10.0.0.1 and 10.0.0.1/32 are equivalent.

The options of "stprov remote dhcp|static" are listed below.  Note that only a
subset of these options are supported by "dhcp", see COMMANDS.

    -i, --ip               Host address in CIDR notation (e.g., 10.0.2.10/26)
    -h, --host             Host name prefix (full host name becomes HOSTNAME.localhost.local)
    -H, --full-host        Full host name (e.g., host.example.org)
    -r, --url              OS package URLs (see defaults below; can be repeated)
    -u, --user             User name when using a templated user:password URL (Default: stboot)
    -p, --pass             Password when using a templated user:password URL (Default: stboot)
    -m, --mac              MAC address of network interface to select (e.g., aa:bb:cc:dd:ee:ff)
    -I, --interface        Name of network interface to select (e.g., eth0)
    -A, --autodetect       Autodetect network interface and ping gateway
    -B, --bonding-auto     Autodetect network interfaces to bond into bond0
    -b, --bonding          Name of network interface to bond into bond0 (can be repeated)
    -M, --bonding-mode     Bonding mode (Default: balance-rr)
    -w, --wait             Wait at most this long for link up (Default: 4s)
    -g, --gateway          Gateway IP address (Default: assuming first address in HOST_ADDR's network)
    -x, --try-last-gateway Override default gateway and instead assume last address in HOST_ADDR's network
    -f, --force            Proceed despite failing configuration sanity checks, logging ignored issues
    -d, --dns              DNS server IP addresses (Default: 9.9.9.9, 149.112.112.112; can be repeated)

    The first occurrence of the pattern user:password in the specified OS
    package URL(s) are substituted with the values of -u and -p.  For example,
    "user:password" might get substituted to "alice:sikritpassword".

    The default OS package URL(s) are:
    https://user:password@stpackage.example.org/os-stable.json.

    Bonding mode (-M) is one of: balance-rr, active-backup, balance-xor,
    broadcast, 802.3ad, balance-tlb, balance-alb.

## FILES AND DIRECTORIES

stprov reads TLS roots from the [trust policy][] directory "/etc/trust_policy".
These TLS roots are required and used to HEAD-request all OS package URLs.

stprov writes a [host configuration][], a hostname, and an SSH hostkey to EFI
NVRAM, see the [EFI variables reference][].  The SSH hostkey is only written if
the "run" subcommand is used for client-server exchanges.

[trust policy]: https://git.glasklar.is/system-transparency/project/docs/-/blob/v0.2.0/content/docs/reference/trust_policy.md
[EFI variables reference]: https://git.glasklar.is/system-transparency/project/docs/-/blob/v0.2.0/content/docs/reference/efi-variables.md
[host configuration]: https://git.glasklar.is/system-transparency/project/docs/-/blob/v0.2.0/content/docs/reference/host_configuration.md

## VARIABLES

Several default values can be overridden at compile time.  Each default value is
a string.  If the default value takes multiple values, use comma for separation.

Refer to the stprov Makefile for details on building with custom default values.

## RETURN CODES

A non-zero return code is used to indicate failure.

## EXAMPLES

Configure a DHCP network using the eth0 interface, hostname "st.example.org",
two OS package URLs, and two DNS servers.

    stprov remote dhcp -I eth0 -H st.example.org\
        -r https://ospkg-01.example.org/bookworm.json -r https://ospkg-02.example.org/bookworm.json\
        -d 9.9.9.9 -d 149.112.112.112

Configure a static network using the eth0 interface, host network
192.168.0.4/24, hostname "st.example.org", two OS package URLs, and two DNS
servers.  The default gateway defaults to 192.168.0.1 in this example.

    stprov remote static -I eth0 -i 192.168.0.4/24 -H st.example.org\
        -r https://ospkg-01.example.org/bookworm.json -r https://ospkg-02.example.org/bookworm.json\
        -d 9.9.9.9 -d 149.112.112.112

Configure a static network configuration with bonding while typing as little as
possible.  This depends on appropriate compile-time defaults, see VARIABLES.

    stprov remote static -i 192.168.0.4/24 -h st -B

Wait for commands from "stprov local", which connects from 192.168.0.1/26.

    stprov remote run -o sikritpassword -a 192.168.0.1/26

Provide commands to "stprov remote", which listens on 192.168.1.24.

    stprov local run -o sikritpassword -i 192.168.1.24

## SECURITY CONSIDERATIONS

The HTTPS connection used in the client-server exchanges is no more secure than
the entropy used for the one-time password.  The impact of the one-time password
being guessed is that a passive attacker can observe the entropy from "local".
The SSH hostkey would as a result only depend on the platform's own entropy.

It would not go unnoticed if an active attacker from an allowed network guessed
the one-time password during provisioning.  Incoming connection attempts are
shown, and the operator is presented with relevant values that get committed.

## REPORT BUGS

Refer to the project's issue tracker at:
https://git.glasklar.is/system-transparency/core/stprov/-/issues.

Send email to the above issue tracker:
system-transparency-core-stprov-issues@incoming.glasklar.is

Send email to the project's discuss list:
https://lists.system-transparency.org/mailman3/postorius/lists/st-discuss.lists.system-transparency.org/.

## SEE ALSO

The [stprov system documentation][] describes stprov from a
design and intended usage perspective without being a dense reference manual.

[stprov system documentation]: ./stprov-system.md
