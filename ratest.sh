#!/bin/bash

cat <<'EOF' >ratest-in-container.sh
#!/bin/bash

# Stuff copypasted from st-complete-poc tests
rm -rf tmp.tpm
mkdir -p tmp.tpm/state tmp.tpm/config
tpmd=$(pwd)/tmp.tpm
XDG_CONFIG_HOME=$tpmd/config swtpm_setup --create-config-files > swtpm_setup.log
XDG_CONFIG_HOME=$tpmd/config swtpm_setup --tpm2 --tpmstate $tpmd/state \
	       --create-ek-cert --create-platform-cert --lock-nvram >> swtpm_setup.log
swtpm socket --tpmstate "dir=$tpmd/state" --tpm2 --pid "file=$tpmd/swtpm.pid" \
      --ctrl "type=unixio,path=$tpmd/swtpm.socket" --log file=swtpm.log,level=10,truncate &
export TPM_SOCKET=$tpmd/swtpm.socket

./integration/qemu.sh || true

swtpm_pid=$(cat $tpmd/swtpm.pid 2>/dev/null) || true
[[ -z "${swtpm_pid}" ]] || kill "${swtpm_pid}"
EOF

chmod +x ratest-in-container.sh

podman run -it --pull=always \
       -v $(pwd):/src -w /src \
       -e SINGLE_TEST=0 \
       git.glasklar.is:5050/glasklar/infra/containers/stboot:integration \
       ./ratest-in-container.sh
