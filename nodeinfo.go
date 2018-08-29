package readas

import (
	"github.com/writeas/go-nodeinfo"
)

type nodeInfoResolver struct{ app *app }

func nodeInfoConfig(cfg *config) *nodeinfo.Config {
	name := "Read.as"
	return &nodeinfo.Config{
		BaseURL: cfg.Host,
		InfoURL: "/api/nodeinfo",

		Metadata: nodeinfo.Metadata{
			NodeName:        name,
			NodeDescription: "ActivityPub-enabled long-form reader.",
			Private:         false,
			Software: nodeinfo.SoftwareMeta{
				HomePage: "https://read.as",
				GitHub:   "https://github.com/writeas/Read.as",
			},
		},
		Protocols: []nodeinfo.NodeProtocol{
			nodeinfo.ProtocolActivityPub,
		},
		Services: nodeinfo.Services{
			Inbound:  []nodeinfo.NodeService{},
			Outbound: []nodeinfo.NodeService{},
		},
		Software: nodeinfo.SoftwareInfo{
			Name:    "read.as",
			Version: softwareVersion,
		},
	}
}

func (r nodeInfoResolver) IsOpenRegistration() (bool, error) {
	return false, nil
}

func (r nodeInfoResolver) Usage() (nodeinfo.Usage, error) {
	users, err := r.app.getUsersCount()
	return nodeinfo.Usage{
		Users: nodeinfo.UsageUsers{
			Total: int(users),
		},
		LocalPosts: 0,
	}, err
}
