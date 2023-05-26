DEFAULT_DOMAIN=localhost.local
DEFAULT_USER=stboot
DEFAULT_PASSWORD=stboot
DEFAULT_DNS=8.8.8.8
DEFAULT_ALLOWED_NETWORKS=127.0.0.1/32
DEFAULT_BONDING_MODE=balance-rrheyho

FLAGS = -X 'system-transparency.org/stprov/internal/version.Version=$(shell git describe --tags --always)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefHostname=$(DEFAULT_DOMAIN)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefUser=$(DEFAULT_USER)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefPassword=$(DEFAULT_PASSWORD)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefDNS=$(DEFAULT_DNS)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefAllowedNetworks=$(DEFAULT_ALLOWED_NETWORKS)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefBondingMode=$(DEFAULT_BONDING_MODE)'

all: build
build: stprov

.PHONY: stprov
stprov:
	go build -ldflags="$(FLAGS)" -o $@ ./cmd/stprov
