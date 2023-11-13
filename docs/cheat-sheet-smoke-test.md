# Manual release testing notes for stprov

In core/system-transparency, commit-id 5b6a9a36d769387dd602773423e17c905125f29a.
This means we will be using stboot, tag v0.2.1, for all of the below.

```
$ task clean-all
$ source setup.env
$ go install $TASK_PACKAGE
$ go install system-transparency.org/stprov/cmd/stprov@v0.2.0
$ task iso-provision demo:ospkg qemu:iso
```

In the qemu-shell:
```
# stprov remote static -h myhostname -i 10.0.2.15/24 -g 10.0.2.2 -r http://10.0.2.2:8080/os-pkg-example-ubuntu20.json
...
[INFO] stboot: eth0: IP configuration successful
2023/11/13 19:01:55 HEAD request on provisioning url gave content-length: 1252, content-type: "application/json"
# printf "\n" | stprov remote run -p 3000 --allow=0.0.0.0/0 --otp=sikritpassword
2023/11/13 19:02:20 starting server on 0.0.0.0:3000
```

In a new terminal:
```
$ ./cache/go/bin/stprov local run --ip 127.0.0.1 -p 3000 --otp sikritpassword
2023/11/13 19:02:33 added entropy

   0  19 49 35 0C A2 CF 7B 96  B2 CF 34 4A 05 09 8D 1A  .I5...{...4J....
  16  4C 26 58 6D D1 53 EF F4  36 0E 21 9E 2C 19 35 62  L&Xm.S..6.!.,.5b

fingerprint=SHA256:QcNVmTLvxaMuJOyyIWqYxDe92NO10ADMnSrnFb0tX1g
hostname=myhostname.localhost.local
ip=127.0.0.1
```

Back to the qemu-shell:
```
2023/11/13 19:02:32 received entropy

   0  19 49 35 0C A2 CF 7B 96  B2 CF 34 4A 05 09 8D 1A  .I5...{...4J....
  16  4C 26 58 6D D1 53 EF F4  36 0E 21 9E 2C 19 35 62  L&Xm.S..6.!.,.5b

/# cat /sys/firmware/efi/efivars/STHostConfig-f401f2c1-b005-4be0-8cee-f2e5945bcbe7
{"version":1,"network_mode":"static","host_ip":"10.0.2.15/24","gateway":"10.0.2.2","dns":["9.9.9.9"],"network_interfaces":[{"interface_name":"eth0","mac_address":"52:54:00:12:34:56"}],"ospkg_pointer":"http://10.0.2.2:8080/os-pkg-example-ubuntu20.json","identity":"bar","authentication":"foo","bonding_mode":"","bond_name":""}
/# cat /sys/firmware/efi/efivars/STHostKey-f401f2c1-b005-4be0-8cee-f2e5945bcbe7
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz
c2gtZWQyNTUxOQAAACD6rwy/Na0LB5QI8oYMMJ6zXIZvlSLnztHjuzvuvKk3lwAA
AKAj3K6BI9yugQAAAAtzc2gtZWQyNTUxOQAAACD6rwy/Na0LB5QI8oYMMJ6zXIZv
lSLnztHjuzvuvKk3lwAAAECQWOSDxzlK/OSb366gHtm5JnpmDkdxrNAauUSPS7Zw
WvqvDL81rQsHlAjyhgwwnrNchm+VIufO0eO7O+68qTeXAAAAGW9zcGtnQHN5c3Rl
bS10cmFuc3BhcmVuY3kBAgME
-----END OPENSSH PRIVATE KEY-----
/# dd if=/sys/firmware/efi/efivars/STHostName-f401f2c1-b005-4be0-8cee-f2e5945bcbe7 bs=1 skip=4
myhostname.localhost.local26 bytes (0.000 MB, 0.000 MiB) copied, 0.001 s, 0.022 MB/s
```

Looks reasonable.

(Note: the appended "localhost.local" is due to using -h, expected.  Use
`--full-host` if you would rather test with an absolute hostname.)

In the qemu-shell:
```
/# shutdown -r
...

Ubuntu 20.04 LTS ubuntu ttyS0

ubuntu login:
```

We got login-prompt, great.  This smoke-test passed!
