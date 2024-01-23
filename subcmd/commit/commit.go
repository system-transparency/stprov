package commit

import (
	"encoding/json"
	"fmt"
	"os"

	"system-transparency.org/stboot/host"

	"system-transparency.org/stprov/internal/st"
)

const usage = `Usage:

  stprov commit HOSTCONFIG

    Persists a given host configuration to EFI variable store.
`

func checkAndPersistHostConfig(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("host configuration file %q could not be read: %w", path, err)
	}

	var config host.Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return fmt.Errorf("host configuration file %q is invalid: %w", path, err)
	}

	err = st.WriteHostConfigEFI(&config)
	if err != nil {
		return fmt.Errorf("failed to persist host config: %w", err)
	}

	return nil
}

func Main(args []string) error {
	var err error

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, usage)
	} else {
		err = checkAndPersistHostConfig(args[0])
	}

	return err
}
