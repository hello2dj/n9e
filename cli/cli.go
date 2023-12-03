package cli

import (
	"cncamp/pkg/third_party/nightingale/cli/upgrade"
)

func Upgrade(configFile string) error {
	return upgrade.Upgrade(configFile)
}
