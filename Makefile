DEFAULT_DOMAIN=localhost.local
DEFAULT_USER=stboot
DEFAULT_PASSWORD=stboot
DEFAULT_DNS=9.9.9.9
DEFAULT_ALLOWED_NETWORKS=127.0.0.1/32
DEFAULT_BONDING_MODE=balance-rr

FLAGS = -X 'system-transparency.org/stprov/internal/version.Version=$(shell git describe --tags --always)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefHostname=$(DEFAULT_DOMAIN)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefUser=$(DEFAULT_USER)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefPassword=$(DEFAULT_PASSWORD)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefDNS=$(DEFAULT_DNS)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefAllowedNetworks=$(DEFAULT_ALLOWED_NETWORKS)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefBondingMode=$(DEFAULT_BONDING_MODE)'

# https://github.com/golang/go/issues/56174
ENV = GOPRIVATE=git.glasklar.is/system-transparency/core/stauth

all: build
build: stprov

.PHONY: stprov
stprov:
	$(ENV) go build -ldflags="$(FLAGS)" -o $@ ./cmd/stprov
