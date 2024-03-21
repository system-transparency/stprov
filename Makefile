DEFAULT_TEMPLATE_URL=https://user:password@stpackage.example.org/os-stable.json
DEFAULT_DOMAIN=localhost.local
DEFAULT_USER=stboot
DEFAULT_PASSWORD=stboot
DEFAULT_DNS=9.9.9.9,149.112.112.112
DEFAULT_ALLOWED_NETWORKS=127.0.0.1/32
DEFAULT_BONDING_MODE=balance-rr

FLAGS =-X 'system-transparency.org/stprov/internal/version.Version=$(shell git describe --tags --always)'
FLAGS+=-X 'system-transparency.org/stprov/internal/options.DefTemplateURL=$(DEFAULT_TEMPLATE_URL)'
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
