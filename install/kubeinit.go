package install

import (
	"fmt"
	"github.com/kinvin/kd/asset"
	"log"
	"os"
)

func generateResource() error {
	if err := asset.RestoreAssets("./", "resource"); err != nil {
		return fmt.Errorf("RestoreAssets error: %v\n", err)
	}
	privateKeyFile := ResourcePath + "sa.key"
	publicKeyFile := ResourcePath + "sa.pub"
	privateKey, publicKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("Generate key err: %v\n", err)
	}
	err = DumpPrivateKeyFile(privateKey, privateKeyFile)
	if err != nil {
		return fmt.Errorf("Dump PrivateKey error: %v\n", err)
	}
	err = DumpPublicKeyFile(publicKey, publicKeyFile)
	if err != nil {
		return fmt.Errorf("Dump PublicKey error: %v\n", err)
	}
	return nil
}

func Kubeinit() {
	if len(ConfigFile) == 0 {
		log.Fatal(fmt.Errorf("Please provider kubernetes config file, Use --config"))
	}
	config := new(InitConfig)
	config.GetConf(ConfigFile)
	if len(config.User) == 0 || len(config.Passwd) == 0 {
		config.User, config.Passwd = Credentials()

	}
	_, err := os.Stat(ResourcePath)
	if err != nil {
		if err := generateResource(); err != nil {
			log.Fatalf("Generate Resource file Failure, error: %v\n", err)
		}
	}
	/*
		err = config.GenerateCertsAndConfig()
		if err != nil {
			fmt.Println(err)
		}
		config.CheckValid()
		/*
			err = config.BasicInit()
			if err != nil {
				log.Fatal(err)
			}
	*/
}
