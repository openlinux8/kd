package install

var (
	ConfigFile     string
	KernelVersion  = "4.4.176"
	DockerVersion  = "18.09.0-ce"
	RemotePath     = "/tmp/"
	ResourcePath   = "resource/"
	BootstrapToken = "e373d6a579ede35b4ed0e6a2a4b72fdd"
)

type InitConfig struct {
	LoadBalancers []string `yaml:"loadBalancers,omitempty"`
	Etcds         []string `yaml:"etcds,omitempty"`
	Masters       []string `yaml:"masters,omitempty"`
	Nodes         []string `yaml:"nodes,omitempty"`
	VIP           string   `yaml:"clusterVip,omitempty"`
	User          string   `yaml:"sshUser,omitempty"`
	Passwd        string   `yaml:"sshPassword,omitempty"`
	EtcdVersion   string   `yaml:"etcdVersion,omitempty"`
	Version       string   `yaml:"clusterVersion,omitempty"`
	ImageRepo     string   `yaml:"imageRepo,omitempty"`
	ServiceSubnet string   `yaml:"serviceSubnet,omitempty"`
	PodSubnet     string   `yaml:"podSubnet,omitempty"`
	ResolvConf    string   `yaml:"resolvConf,omitempty"`
	ClusterDNS    string   `yaml:"clusterDNS,omitempty"`
}

var KernelFiles = []string{
	"kernel-lt-4.4.176-1.el7.elrepo.x86_64.rpm",
}

var DockerFiles = []string{
	"docker-18.09.0.tgz",
	"docker-sysconfig",
	"docker-init.sh",
	"docker.service",
	"docker.socket",
}

var InstallDockerCmds = []string{
	"sudo tar -xvf " + RemotePath + DockerFiles[0] + " -C " + RemotePath,
	"sudo mv " + RemotePath + "docker/* /usr/bin/",
	"sudo mv " + RemotePath + "docker-sysconfig /etc/sysconfig/docker",
	"sudo mv " + RemotePath + "docker-init.sh /usr/local/bin/",
	"sudo mv " + RemotePath + "docker.s* /usr/lib/systemd/system/",
	"sudo chmod 755 /usr/local/bin/docker-init.sh",
	"sudo chown root:root /usr/local/bin/docker-init.sh /etc/sysconfig/docker /usr/lib/systemd/system/docker.s*",
	"sudo rm -rf " + RemotePath + "docker*",
	"sudo systemctl enable docker && sudo systemctl start docker",
}

var KubeletFiles = []string{
	"kubectl",
	"kubelet",
	"kubelet.service",
	"kubelet-flags",
	"config.yaml",
}
