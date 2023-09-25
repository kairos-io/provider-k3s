package services

import (
	"fmt"
	"github.com/kairos-io/kairos-sdk/machine"
	"github.com/kairos-io/kairos-sdk/machine/openrc"
	"github.com/kairos-io/kairos-sdk/machine/systemd"
	"github.com/kairos-io/kairos-sdk/utils"
	"os"
	"path/filepath"
)

const edgevpnSystemd string = `[Unit]
Description=EdgeVPN Daemon
After=network.target
[Service]
EnvironmentFile=/etc/systemd/system.conf.d/edgevpn-%i.env
LimitNOFILE=49152
ExecStart=edgevpn
Restart=always
[Install]
WantedBy=multi-user.target`

const edgevpnOpenRC string = `#!/sbin/openrc-run

depend() {
	after net
	provide edgevpn
}

supervisor=supervise-daemon
name="edgevpn"
command="edgevpn"
supervise_daemon_args="--stdout /var/log/edgevpn.log --stderr /var/log/edgevpn.log"
pidfile="/run/edgevpn.pid"
respawn_delay=5
set -o allexport
if [ -f /etc/environment ]; then source /etc/environment; fi
if [ -f /etc/systemd/system.conf.d/edgevpn-kairos.env ]; then source /etc/systemd/system.conf.d/edgevpn-kairos.env; fi
set +o allexport`

const edgeVPNDefaultInstance string = "kairos"

func EdgeVPN(instance, rootDir string) (machine.Service, error) {
	if utils.IsOpenRCBased() {
		return openrc.NewService(
			openrc.WithName("edgevpn"),
			openrc.WithUnitContent(edgevpnOpenRC),
			openrc.WithRoot(rootDir),
		)
	}

	return systemd.NewService(
		systemd.WithName("edgevpn"),
		systemd.WithInstance(instance),
		systemd.WithUnitContent(edgevpnSystemd),
		systemd.WithRoot(rootDir),
	)
}

func SetupVPN(token, rootDir string) error {
	svc, err := EdgeVPN(edgeVPNDefaultInstance, rootDir)
	if err != nil {
		return fmt.Errorf("could not create svc: %w", err)
	}

	vpnOpts := map[string]string{
		"DHCP":         "true",
		"DHCPLEASEDIR": "/usr/local/.kairos/lease",
	}
	if token != "" {
		vpnOpts["EDGEVPNTOKEN"] = token
	}

	vpnOpts["EDGEVPNDHT"] = "false"

	os.MkdirAll("/etc/systemd/system.conf.d/", 0600) //nolint:errcheck
	// Setup edgevpn instance
	err = utils.WriteEnv(filepath.Join(rootDir, "/etc/systemd/system.conf.d/edgevpn-kairos.env"), vpnOpts)
	if err != nil {
		return fmt.Errorf("could not create write env file: %w", err)
	}

	err = svc.WriteUnit()
	if err != nil {
		return fmt.Errorf("could not create write unit file: %w", err)
	}

	err = svc.Start()
	if err != nil {
		return fmt.Errorf("could not start svc: %w", err)
	}

	return svc.Enable()
}
