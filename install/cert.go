package install

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
)

func GenerateKey() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	publicKey := &privateKey.PublicKey
	return privateKey, publicKey, nil
}

func DumpPrivateKeyFile(privatekey *rsa.PrivateKey, filename string) error {
	var keybytes []byte = x509.MarshalPKCS1PrivateKey(privatekey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keybytes,
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = pem.Encode(file, block)
	if err != nil {
		return err
	}
	return nil
}

func DumpPublicKeyFile(publickey *rsa.PublicKey, filename string) error {
	keybytes, err := x509.MarshalPKIXPublicKey(publickey)
	if err != nil {
		return err
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: keybytes,
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = pem.Encode(file, block)
	if err != nil {
		return err
	}
	return nil
}

func BuildCA(caCsr, caName string) error {
	_, err := os.Stat(ResourcePath + caName + ".pem")
	if err == nil {
		return nil
	}
	cfsslCmd := ResourcePath + "cfssl gencert "
	cfssljsonCmd := ResourcePath + "cfssljson -bare "
	initArg := "-initca " + ResourcePath + caCsr + "|"
	cmd := cfsslCmd + initArg + cfssljsonCmd + ResourcePath + caName
	fmt.Println("exec cmd is: ", cmd)
	err = exec.Command("/bin/bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("生成%s CA证书失败.", caName)
	}
	return nil
}

func BuildCert(caPem, caKey, caConfig, certCsr, certName string) error {
	_, err := os.Stat(ResourcePath + certName + ".pem")
	if err == nil {
		return nil
	}
	cfsslCmd := ResourcePath + "cfssl gencert "
	cfssljsonCmd := ResourcePath + "cfssljson -bare "
	certArgs := "-ca=" + ResourcePath + caPem + " -ca-key=" + ResourcePath + caKey + " -config=" + ResourcePath + caConfig + " -profile=kubernetes " + ResourcePath + certCsr + "|"
	cmd := cfsslCmd + certArgs + cfssljsonCmd + ResourcePath + certName
	fmt.Println("exec cmd is: ", cmd)
	err = exec.Command("/bin/bash", "-c", cmd).Run()
	if err != nil {
		return fmt.Errorf("生成%s 证书失败.", certName)
	}
	return nil
}

func BuildKubeConfig(name, vip, credentialName, suffix string, isBootstrap bool) error {
	setCluster := ResourcePath + "kubectl config set-cluster kubernetes --certificate-authority=" + ResourcePath + "ca.pem --embed-certs=true --server=https://" + vip + ":6443 --kubeconfig=" + ResourcePath + name + suffix
	setCredentials := ResourcePath + "kubectl config set-credentials " + credentialName + " --client-certificate=" + ResourcePath + name + ".pem --client-key=" + ResourcePath + name + "-key.pem --embed-certs=true --kubeconfig=" + ResourcePath + name + suffix
	setContext := ResourcePath + "kubectl config set-context default --cluster=kubernetes --user=" + credentialName + " --kubeconfig=" + ResourcePath + name + suffix
	useContext := ResourcePath + "kubectl config use-context default --kubeconfig=" + ResourcePath + name + suffix
	if isBootstrap {
		setCredentials = ResourcePath + "kubectl config set-credentials " + credentialName + " --token=" + BootstrapToken + " --kubeconfig=" + ResourcePath + name + suffix
	}
	_, err := os.Stat(ResourcePath + name + suffix)
	if err == nil {
		return nil
	}
	cmds := []string{setCluster, setCredentials, setContext, useContext}
	for _, cmd := range cmds {
		fmt.Println("exec cmd is: ", cmd)
		err = exec.Command("/bin/bash", "-c", cmd).Run()
		if err != nil {
			return fmt.Errorf("生成%s.%s kubeConfig文件失败.", name, suffix)
		}
	}
	return nil
}

func (c *InitConfig) GenerateCertsAndConfig() error {
	etcdCa := "etcd-ca"
	ca := "ca"
	frontCa := "front-proxy-ca"
	caConfig := "ca-config.json"
	hosts := append(c.Masters, c.Nodes...)
	hosts = append(hosts, c.Etcds...)
	hosts = append(hosts, c.LoadBalancers...)
	hosts = RemoveDuplicateElement(hosts)
	env := make(map[string]interface{})
	caNames := []string{etcdCa, ca, frontCa}
	certNames := []string{"etcd", "admin", "kube-controller-manager", "kube-proxy", "kube-scheduler", "kubernetes", "front-proxy-client"}
	certNames = append(certNames, hosts...)
	for _, caName := range caNames {
		err := BuildCA(caName+"-csr.json", caName)
		if err != nil {
			return err
		}
	}
	for _, certName := range certNames {
		switch certName {
		case "etcd":
			env["Etcds"] = c.Etcds
			err := GenerateContent(certName+"-csr", env)
			if err != nil {
				return err
			}
			err = BuildCert(etcdCa+".pem", etcdCa+"-key.pem", caConfig, certName+"-csr.json", certName)
			if err != nil {
				return err
			}
		case "admin", "kube-controller-manager", "kube-proxy", "kube-scheduler":
			err := BuildCert(ca+".pem", ca+"-key.pem", caConfig, certName+"-csr.json", certName)
			if err != nil {
				return err
			}
			var credentialName string
			if certName == "admin" {
				credentialName = certName
			} else {
				credentialName = "system:" + certName
			}
			err = BuildKubeConfig(certName, c.VIP, credentialName, ".conf", false)
			if err != nil {
				return err
			}
		case "kubernetes":
			Hostnames, err := getHostnames(c.User, c.Passwd, c.Masters)
			if err != nil {
				return err
			}
			env["Hostnames"] = Hostnames
			env["Masters"] = c.Masters
			env["VIP"] = c.VIP
			env["ClusterDNS"] = c.ClusterDNS
			err = GenerateContent(certName+"-csr", env)
			if err != nil {
				return err
			}
			err = BuildCert(ca+".pem", ca+"-key.pem", caConfig, certName+"-csr.json", certName)
			if err != nil {
				return err
			}
		case "front-proxy-client":
			err := BuildCert(frontCa+".pem", frontCa+"-key.pem", caConfig, certName+"-csr.json", certName)
			if err != nil {
				return err
			}
		default:
			hosts := []string{certName}
			Hostnames, err := getHostnames(c.User, c.Passwd, hosts)
			if err != nil {
				return err
			}
			env["Hostname"] = Hostnames[0]
			if _, err := os.Stat(ResourcePath + certName + "-csr.tpl"); err != nil {
				cmd := "cp " + ResourcePath + "kubelet-csr.tpl " + ResourcePath + certName + "-csr.tpl"
				err = exec.Command("/bin/bash", "-c", cmd).Run()
				if err != nil {
					return err
				}
			}
			err = GenerateContent(certName+"-csr", env)
			if err != nil {
				return err
			}
			err = BuildCert(ca+".pem", ca+"-key.pem", caConfig, certName+"-csr.json", certName)
			if err != nil {
				return err
			}
			credentialName := "system:node:" + Hostnames[0]
			err = BuildKubeConfig(certName, c.VIP, credentialName, ".conf", false)
			if err != nil {
				return err
			}
			credentialName = "kubelet-bootstrap"
			err = BuildKubeConfig(certName, c.VIP, credentialName, ".bootstrap", true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
