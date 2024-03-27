# System documentation for stprov

This document describes a tool for provisioning host-specific configuration to a
System-Transparency booted platform.  Refer to the [stprov
manual](./stprov-manual) for detailed usage of the implemented tool.

## Introduction

stprov is a System Transparency provisioning tool designed to lock down initial
trust in a platform.  The platform to be provisioned is assumed to have a poor
management interface.  For example, it may be a remote web console that does not
support copy-paste.  The platform is further assumed to use the same stboot
image as several other platforms.  In other words, it needs to be provisioned
with its own host-specific configuration to become useful.

The stprov architecture is composed of two parts:

  1. Local: client-side that the operator runs on, e.g., a provisioning laptop.
  2. Remote: server-side that the operator runs on the platform to provision.

As alluded to above, the operator's user interface to the platform is poor.
Therefore, the input and output needed in the remote part is kept to a minimum.
This amounts to configuring the network and setting up a secure channel to the
operator's local system that has reliable input and output.

An example deployment that makes use of stprov is shown below.  The operator
accesses a new platform through a remote management interface that allows
mounting a provisioning ISO.  When booted, the ISO drops into a terminal where
the operator can use the stprov-remote program to configure the network and
start an HTTPS service.  The operator completes the provisioning by connecting to
this service using stprov-local.  After provisioning, the platform's EFI NVRAM
contains a [host configuration](), a hostname, and an SSH hostkey.  The SSH
hostkey can, e.g., be used by an OS package to provide SSH host authentication.

```mermaid
graph LR
    subgraph DC ["Trusted network"]
        mserver("Management server")
        platform1("Provisioned platform")
        platform2("Provisioned platform")
        platform3("Unprovisioned platform")
    end

    classDef hidden display: none;
	router1:::hidden
	router2:::hidden
	router3:::hidden

    operator("Operator")
    operator -- "Web interface" --> mserver

    mserver -.- platform1
    mserver -.- platform2
    mserver -- "mount ISO over LAN<br>run stprov-remote" --> platform3

    platform1 -. "public internet" .-o router1
    platform2 -. "public internet" .-o router2
    platform3 -- "HTTPS service<br>for provisioning" --o router3
    operator -- "stprov-local connects"--o router3
```

## How to make stprov run

The stprov-local program is simple to just run on the operator's own system.

The stprov-remote program is typically embedded in a provisioning ISO, or as an
OS package in stboot's initramfs for use in so-called provisioning mode.  It is
not within the scope of this document to describe creation of these artifacts,
but the reader may find [stmgr]() helpful for creating such artifacts.  In lack
of other good documentation, considering peeking at the stprov and stboot CI.

## Provisioned configuration

The following configuration is provisioned by stprov-remote:

  - [Host configuration](): primarily used by stboot to network-boot an OS
    package.  It can also be used by OS packages to configure their networks.
  - Hostname: an arbitrary hostname that OS packages may use.

The following configuration is provisioned with the help of stprov-local:

  - SSH hostkey: a cryptographic identity that OS packages may use.

The SSH hostkey is derived from entropy provided by the operator (local) and the
platform's own entropy (remote).  In more detail, HKDF is used to derive a
unique secret from 128-bits of local and remote entropy.  HKDF is then used
again to derive an SSH hostkey deterministically from that.  Assuming it is hard
to gain access to EFI NVRAM, the derived SSH hostkey never leaves the platform.

All configuration is written to EFI NVRAM, see details in the common [storage
index]().

## Provisioning flow
```mermaid
sequenceDiagram

participant Operator
participant platform

Operator->>platform: (1) Gain access to console with stprov

Operator->>platform: (2) stprov remote {dhcp,static} [Options]

platform->>platform: (2.1) Configure and verify network
platform->>platform: (2.2) Write host configuration
platform->>platform: (2.3) Write hostname

Operator->>platform: (3) stprov remote run [Options]
platform->>platform: (3.1) Start HTTPS service

Operator->>Operator: (4) stprov local [Options]
Operator->>platform: (4.1) Add 128-bits of entropy
Operator->>platform: (4.2) Commit

platform->>platform: (3.2) Sample 128-bits of entropy, HKDF
platform->>platform: (3.3) Derive and write SSH hostkey

platform->>Operator: (4.3) Receive system manifest
```

The above figure provides an overview of the provisioning flow.  In more detail:

  1. The operator first gains access to the platform's console where the stprov
     program is available.  See "how to make stprov run" as well as the the
     introductory figure for an overview of the intended setting.
  2. A static or a dynamic network configuration is then applied based on
     relevant options such as IP address, default gateway, and hostname.  If
     configuration succeeds, a host configuration and hostname is committed to
     EFI NVRAM.  Provisioning ends here if the SSH hostkey is not essential.
  3. The stprov-remote program is used again on the platform to start an HTTPS
     server that awaits further input from stprov-local.  Important options here
     include a one-time password used to establish a mutually authenticated
     HTTPS session, as well as allowed networks the operator can connect from.
  4. The stprov-local program contributes with entropy that stprov-remote mixes
     into its key derivations.  The operator is prompted to review the changes
     before generating and committing the derived SSH hostkey to EFI NVRAM.  The
     returned system manifest most notably includes the SSH hostkey fingerprint.

The [stprov manual](./stprov-manual) describes all key-value output from
stprov-local, which is based on the system manifest received in the final step.

## Client-server API

The exchanges between stprov-local and stprov-remote take place using an HTTP
API.  It is not in scope of this document's revision to describe it in detail.
The short summary would be that stprov-remote has HTTP endpoints that accept
JSON key-value pairs.  Output is also encoded as JSON key-value pairs (if any).

## Security considerations

The provisioning itself hinges on the platform's management interface being
"secure".  On one side of the spectrum is a management interface requiring real
physical presence.  On the other side of the spectrum is the type of management
interface described in the introduction; where a web interface is used to
connect to a management server, which in turn may send commands in plaintext
over a LAN to the platform.  A detailed analysis is out of scope because it is
deployment specific.  What can be said is that a passive on-LAN attacker may
trivially learn the operator's one-time password.  This makes the entropy
provided by stprov-local deterministic, and so reduces the entropy of any
generated key material to the platform's own entropy source.  An active on-LAN
attacker at this early stage would completely undermine the provisioning.

An on-path or Internet attacker cannot do much, expect for disturbing the
provisioning with dropped packets or connecting from non-allowed networks (and
failing).  Even adversarial connections from an allowed network with a correct
one-time password would be detectable due to logging in stprov's UX.  The
difficulty of brute-forcing the one-time password after-the-fact depends on its
entropy.  Operators should pick a one-time password that is hard to guess.  A
predictable one-time password is similar to a passive on-LAN attacker above.

Access to EFI-NVRAM is assumed to be hard, both if physical attacks happen to be
possible or as the platform is operated with stboot after provisioning.  If this
assumption does not hold, the platforms configuration may be revealed and/or
tampered with.  Tampering could result in denial-of-service e.g., due to an
invalid configuration.  Leaked cryptographic secrets could result in
machine-in-the-middle attacks and additional information disclosure.

## Future work

A non-exhaustive list:

  - Support for other storage mediums than EFI NVRAM?
  - Provisioning of additional configuration useful for OS packages?
  - Provisioning related to remote attestation?
  - Provisioning of UEFI Secure Boot keys?
  - Use of a platform's TPM to encrypt provisioned secrets at rest?
