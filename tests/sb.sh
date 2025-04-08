#/bin/bash

set -eu
trap clean_up EXIT

function clean_up() {
	echo "clean up" >/dev/null
	rm -f  pk{,.pem,.esl}
	rm -f kek{,.pem,.esl}
	rm -f  db{,.pem,.esl}
	rm -f dbx{,.pem,.esl}
}

function info() {
	echo "INFO: $*" >&2
}

for key in pk kek db dbx; do
	guid=$(uuidgen)
	openssl req -newkey rsa:4096 -nodes -keyout "$key" -x509 -days 1 -out "$key.pem" -subj "/O=Testonly/CN=Secure Boot -- $key/"  2>/dev/null
	cert-to-efi-sig-list -g "$(uuidgen)" "$key.pem" "$key.esl"

	info "created $guid <-- $key"
done

cat dbx.esl >> db.esl
info "adding dbx to db, so #db=2, where one cert is revoked"

#go run ../cmd/stprov local run --ip 127.0.0.1 --otp 1234 --db db.esl
