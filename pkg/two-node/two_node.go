package twonode

const (
	PrepareTwoNodeSqliteDb     = "Prepare two-node sqlite db"
	PrepareMarmotConfig        = "Prepare marmot config"
	EnableMarmotSystemdService = "Enable marmot systemd service"

	ConfigPath = "/etc/two-node"

	Marmot     = "marmot"
	MarmotUnit = `
[Unit]
Description=Marmot synchronizes the k8s state in SQLite between nodes in a two node topology

[Service]
TimeoutStartSec=0
Restart=always

ExecStart=/opt/spectrocloud/bin/marmot -config /etc/two-node/marmot.toml

[Install]
WantedBy=multi-user.target
`
	MarmotLeader = `
# Path to target SQLite database
seq_map_path="/etc/kubernetes/marmot-sm.cbor"
db_path="/var/lib/rancher/k3s/server/db/state.db"
node_id=1

[nats]
server_config="/etc/two-node/nats.config"

# Console STDOUT configurations
[logging]
# Configure console logging
verbose=true
# "console" | "json"
format="console"
`
	MarmotFollower = `
# Path to target SQLite database
seq_map_path="/etc/kubernetes/marmot-sm.cbor"
db_path="/var/lib/rancher/k3s/server/db/state.db"

[nats]
# address of the nats leader
urls=[
  "nats://uninitialized:4222"
]

# Console STDOUT configurations
[logging]
# Configure console logging
verbose=true
# "console" | "json"
format="console"
`
	NatsConfig = "listen: 0.0.0.0:4222"
)
