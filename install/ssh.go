package install

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"os"
	"time"
)

func connect(user, passwd, host string) (*ssh.Session, error) {
	config := &ssh.ClientConfig{
		Timeout:         time.Second,
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	config.Auth = []ssh.AuthMethod{ssh.Password(passwd)}
	addr := fmt.Sprintf("%s:22", host)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return nil, err
	}

	return session, nil
}

func Cmd(user, password, host, cmd string) (output []byte, err error) {
	fmt.Printf("[%s]exec cmd is : %s\n", host, cmd)
	session, err := connect(user, password, host)
	if err != nil {
		return
	}
	defer session.Close()
	output, err = session.CombinedOutput(cmd)
	if err != nil {
		return
	}
	return
}

func SftpTransfer(user, passwd, host, remotePath string, localFiles []string) error {
	var (
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)

	clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(passwd)},
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr = fmt.Sprintf("%s:22", host)

	sshClient, err = ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sftpClient, err = sftp.NewClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	for _, localFile := range localFiles {
		srcFile, err := os.Open(ResourcePath + localFile)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		remoteFile := remotePath + localFile
		dstFile, err := sftpClient.Create(remoteFile)
		if err != nil {
			return err
		}
		buf := make([]byte, 4096)
		//total := 0
		for {
			n, _ := srcFile.Read(buf)
			if n == 0 {
				break
			}
			_, _ = dstFile.Write(buf[0:n])
			//total += length
		}
		//fmt.Printf("host:[%s] transfer file %s, total size is: %d\n", host, remoteFile, total)
	}
	return nil
}
