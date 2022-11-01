package install

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"regexp"
	"strings"
	"time"
)

func (c *InitConfig) CheckValid() {
	var flag bool
	var err error
	var session *ssh.Session
	hosts := append(c.Masters, c.Nodes...)
	hosts = append(hosts, c.Etcds...)
	hosts = append(hosts, c.LoadBalancers...)
	hosts = RemoveDuplicateElement(hosts)
	for _, host := range hosts {
		session, err = connect(c.User, c.Passwd, host)
		connect(c.User, c.Passwd, host)
		if err != nil {
			flag = true
			fmt.Printf("[Error] host %s connected failure, reason: %v\n", host, err)
		}
	}
	defer func() {
		if session != nil {
			session.Close()
		}
	}()
	if flag {
		os.Exit(1)
	}
}

func RebootCheck(user, password, host string) error {
	ticker := time.NewTicker(60 * time.Second)
	i := 0
	for {
		<-ticker.C
		i++
		session, err := connect(user, password, host)
		if err != nil {
			if i == 9 {
				return fmt.Errorf("[%s] 连接超时\n", host)
			}
			fmt.Printf("[%s] 连接失败，正在重试\n", host)
		} else {
			session.Close()
			ticker.Stop()
			break
		}
	}
	return nil
}

func CheckKernel(user, password, host string) bool {
	cmd := `uname -a`
	result, err := Cmd(user, password, host, cmd)
	if err != nil {
		return false
	}
	if strings.Index(string(result), KernelVersion) == -1 {
		return false
	}
	return true
}

func CheckDocker(user, password, host string) (flag bool, isRunning bool) {
	cmd := `sudo docker version`
	result, _ := Cmd(user, password, host, cmd)
	reg := regexp.MustCompile(`Version.*18`)
	output := reg.FindAllString(string(result), -1)
	if len(output) == 0 {
		return false, false
	}
	if strings.Index(string(result), "connect") != -1 {
		fmt.Printf("[%s] docker已安装，但没有运行. 尝试启动docker\n", host)
		cmd := `sudo systemctl enable docker && sudo systemctl start docker`
		_, err := Cmd(user, password, host, cmd)
		if err != nil {
			fmt.Printf("[%s] 尝试启动docker失败\n", host)
			return true, false
		}
	}
	return true, true
}

func CheckKubelet(user, password, host, kubeletVersion string) bool {
	cmd := "kubelet --version"
	result, _ := Cmd(user, password, host, cmd)
	if strings.Index(string(result), kubeletVersion) == -1 {
		return false
	}
	cmd = `ps aux|grep '/usr/bin/kubelet' && sudo ls -l /var/lib/kubelet/kubelet-flags`
	_, err := Cmd(user, password, host, cmd)
	if err != nil {
		return false
	}
	return true
}
