package install

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

func (c *InitConfig) GetConf(configFile string) {
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatal(err)
	}
}

func Credentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter SSH Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter SSH Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err == nil {
		fmt.Println("\nPassword typed: " + string(bytePassword))
	}
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

func (c *InitConfig) BasicInit() error {
	var hosts []string
	var errs []error
	hosts = append(hosts, c.LoadBalancers...)
	hosts = append(hosts, c.Etcds...)
	hosts = append(hosts, c.Masters...)
	hosts = append(hosts, c.Nodes...)
	hosts = RemoveDuplicateElement(hosts)
	for _, host := range hosts {
		fmt.Printf("[%s] 检查linux内核版本是否符合要求，当前要求内核版本为%s\n", host, KernelVersion)
		flag := CheckKernel(c.User, c.Passwd, host)
		if flag {
			fmt.Printf("[%s] 当前内核版本为%s, 无需升级内核\n", host, KernelVersion)
		} else {
			fmt.Printf("[%s] 升级%s内核版本\n", host, KernelVersion)
			err := UpdateKernel(c.User, c.Passwd, host)
			if err != nil {
				errs = append(errs, fmt.Errorf("[%s] 升级%s内核失败，原因:%v", host, KernelVersion, err))
				continue
			}
			fmt.Printf("[%s] 升级内核成功，重启中\n", host)
		}
		err := RebootCheck(c.User, c.Passwd, host)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		fmt.Printf("[%s] 检查是否已安装docker及docker版本是否符合要求. 要求docker版本在18.03以上\n", host)
		flag, isRunning := CheckDocker(c.User, c.Passwd, host)
		if !flag {
			fmt.Printf("[%s] 正在安装docker.\n", host)
			err := InstallDocker(c.User, c.Passwd, host, false)
			if err != nil {
				errs = append(errs, fmt.Errorf("[%s] 安装docker %s失败，原因:%v", host, DockerVersion, err))
				continue
			}
		} else if !isRunning {
			fmt.Printf("[%s] 应用标准docker配置.\n", host)
			err := InstallDocker(c.User, c.Passwd, host, true)
			if err != nil {
				errs = append(errs, fmt.Errorf("[%s] 应用标准docker配置失败，原因:%v", host, err))
				continue
			}
		}

		fmt.Printf("[%s] 检查是否已安装kubelet及kubelet版本是否符合要求. 要求kubelet版本是%s.\n", host, c.Version)
		flag = CheckKubelet(c.User, c.Passwd, host, c.Version)
		if !flag {
			fmt.Printf("[%s] 正在安装kubelet.\n", host)
			err := InstallKubelet(c.User, c.Passwd, host)
			if err != nil {
				errs = append(errs, fmt.Errorf("[%s] 安装kubelet失败，原因:%v", host, err))
				continue
			}
		}
	}
	if len(errs) != 0 {
		for _, e := range errs {
			fmt.Println(e)
		}
		return errors.New("执行出现错误，执行终止")
	}
	return nil
}

func getHostnames(user, password string, hosts []string) (hostnames []string, err error) {
	cmd := `hostname`
	for _, host := range hosts {
		result, err := Cmd(user, password, host, cmd)
		if err != nil {
			return hostnames, err
		}
		hostname := string(result[:len(result)-2])
		hostnames = append(hostnames, hostname)
	}
	return
}

func UpdateKernel(user, password, host string) error {
	var num int
	err := SftpTransfer(user, password, host, RemotePath, KernelFiles)
	if err != nil {
		return err
	}
	for _, pkg := range KernelFiles {
		cmd := "sudo rpm -ivh " + RemotePath + pkg
		output, err := Cmd(user, password, host, cmd)
		if err != nil {
			if strings.Index(string(output), "already installed") == -1 {
				return err
			}
		}
	}
	cmd := "sudo grub2-mkconfig -o /boot/grub2/grub.cfg"
	_, err = Cmd(user, password, host, cmd)
	if err != nil {
		return err
	}
	cmd = `sudo grep '^menuentry' /boot/grub2/grub.cfg | grep -n ` + KernelVersion
	result, err := Cmd(user, password, host, cmd)
	if err != nil {
		return err
	}
	if len(result) != 0 {
		tmp := strings.Split(string(result), ":")
		num, _ = strconv.Atoi(tmp[0])
		num = num - 1
	} else {
		return fmt.Errorf("update kernel failure, grub.cfg not found %s menuentry", KernelVersion)
	}
	cmd = `sudo sed -i 's@default="\${saved_entry}@default="` + strconv.Itoa(num) + `@' /boot/grub2/grub.cfg`
	_, err = Cmd(user, password, host, cmd)
	if err != nil {
		return err
	}
	cmd = `sudo shutdown -r now`
	_, _ = Cmd(user, password, host, cmd)
	return nil
}

func InstallDocker(user, password, host string, flag bool) error {
	num := 0
	if flag {
		num = 2
	}
	err := SftpTransfer(user, password, host, RemotePath, DockerFiles)
	if err != nil {
		return err
	}
	for _, cmd := range InstallDockerCmds[num:] {
		_, err := Cmd(user, password, host, cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func InstallKubelet(user, password, host string) error {
	return nil
}

func TemplateFromContent(tmplContent string, env map[string]interface{}) (output []byte, err error) {
	tmpl, err := template.New("text").Parse(tmplContent)
	if err != nil {
		return
	}
	var buffer bytes.Buffer
	_ = tmpl.Execute(&buffer, env)
	return buffer.Bytes(), err
}

func GenerateContent(csrFile string, env map[string]interface{}) error {
	if _, err := os.Stat(ResourcePath + csrFile + ".json"); err != nil {
		buf := make([]byte, 4096)
		buf, err := ioutil.ReadFile(ResourcePath + csrFile + ".tpl")
		if err != nil {
			return err
		}
		output, err := TemplateFromContent(string(buf), env)
		if err != nil {
			return err
		}
		f, err := os.Create(ResourcePath + csrFile + ".json")
		if err != nil {
			return err
		}
		defer f.Close()
		f.WriteString(string(output))
	}
	return nil
}

func RemoveDuplicateElement(addrs []string) []string {
	result := make([]string, 0, len(addrs))
	temp := map[string]struct{}{}
	for _, item := range addrs {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
