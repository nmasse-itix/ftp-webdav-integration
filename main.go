package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/secsy/goftp"
	"github.com/spf13/viper"
	"github.com/studio-b12/gowebdav"
)

type Integration struct {
	ftp                *goftp.Client
	dav                *gowebdav.Client
	davFolder          string
	ftpPollingDuration time.Duration
}

type IntegrationConfig struct {
	FtpHostname        string
	FtpUsername        string
	FtpPassword        string
	WebdavUrl          string
	WebdavUsername     string
	WebdavPassword     string
	WebdavFolder       string
	FtpPollingDuration time.Duration
}

func NewIntegration(config IntegrationConfig) (Integration, error) {
	var err error
	integration := Integration{
		davFolder:          config.WebdavFolder,
		ftpPollingDuration: config.FtpPollingDuration,
	}
	var ftpConfig goftp.Config = goftp.Config{
		User:     config.FtpUsername,
		Password: config.FtpPassword,
	}
	integration.ftp, err = goftp.DialConfig(ftpConfig, config.FtpHostname)
	if err != nil {
		return Integration{}, err
	}
	integration.dav = gowebdav.NewClient(config.WebdavUrl, config.WebdavUsername, config.WebdavPassword)

	// integration.dav.SetTransport(&http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: &tls.Config{InsecureSkipVerify: true}})
	// integration.dav.SetInterceptor(func(method string, rq *http.Request) {
	//     dump, err := httputil.DumpRequest(rq, true)
	//	   if err == nil {
	//         log.Println(string(dump))
	//     } else {
	//         log.Println(err)
	//     }
	// })

	err = integration.dav.Connect()
	if err != nil {
		return Integration{}, err
	}

	return integration, nil
}

func (i *Integration) Do() {
	for {
		entries, err := i.ftp.ReadDir("/")
		if err != nil {
			log.Println(err)
			continue
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			log.Printf("Found a new file: %s", e.Name())

			err = i.Download(e.Name())
			if err != nil {
				log.Printf("%s: %s", e.Name(), err)
			}
		}

		time.Sleep(i.ftpPollingDuration)
	}
}

func (i *Integration) Download(filename string) error {
	davFolder := i.davFolder
	err := i.dav.MkdirAll(davFolder, 0755)
	if err != nil {
		return err
	}

	f, err := ioutil.TempFile("", "tmpfile-")
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(f.Name())

	davFilePath := path.Join(davFolder, filename)
	err = i.ftp.Retrieve(filename, f)
	if err != nil {
		return err
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// HEADS UP !
	//
	// Because of a potential bug with the default Nextcloud configuration,
	// the whole file is loaded in memory before being sent over the network.
	//
	// Long explanation:
	//
	// The golang net/http library behaves differently depending on the
	// implementation behind the io.Reader interface.
	//
	// * bytes.Reader, strings.Reader and bytes.Buffer: Content-Length is set
	//   to the size of the content.
	//
	// * others: no content-length is set and therefore chunked encoding is used.
	//
	// It looks like the default Nginx configuration for Nextcloud does not like
	// chunked encoding...
	//
	// See https://github.com/photoprism/photoprism/issues/443#issuecomment-685608490
	// and https://github.com/studio-b12/gowebdav/issues/35
	content, err := ioutil.ReadAll(f)
	reader := bytes.NewReader(content)

	delete := false
	for j := 0; j < 5; j++ {
		err = i.dav.WriteStream(davFilePath, reader, 0644)
		if err != nil {
			return err
		}

		davInfo, err := i.dav.Stat(davFilePath)
		if err != nil {
			return err
		}

		fileInfo, err := os.Stat(f.Name())
		if err != nil {
			return err
		}

		if davInfo.Size() == fileInfo.Size() {
			delete = true
			break
		} else {
			log.Printf("File size mismatch (%d != %d), retrying an upload!", fileInfo.Size(), davInfo.Size())
		}

		time.Sleep(5 * time.Second)
	}

	if delete {
		log.Printf("Deleting %s...", filename)
		err = i.ftp.Delete(filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func initConfig() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s config.yaml\n", os.Args[0])
		os.Exit(1)
	}

	fd, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("open: %s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
	defer fd.Close()

	viper.SetConfigType("yaml")
	err = viper.ReadConfig(fd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, config := range []string{"FTP.Hostname", "FTP.Username", "FTP.Password", "WebDAV.URL", "WebDAV.Username", "WebDAV.Password", "WebDAV.Folder"} {
		if viper.GetString(config) == "" {
			fmt.Printf("key %s is missing from configuration file\n", config)
			os.Exit(1)
		}
	}
	viper.SetDefault("FTP.PollingDuration", 5*time.Second)
}

func main() {
	initConfig()

	config := IntegrationConfig{
		FtpHostname:        viper.GetString("FTP.Hostname"),
		FtpUsername:        viper.GetString("FTP.Username"),
		FtpPassword:        viper.GetString("FTP.Password"),
		WebdavUrl:          viper.GetString("WebDAV.URL"),
		WebdavUsername:     viper.GetString("WebDAV.Username"),
		WebdavPassword:     viper.GetString("WebDAV.Password"),
		WebdavFolder:       viper.GetString("WebDAV.Folder"),
		FtpPollingDuration: viper.GetDuration("FTP.PollingDuration"),
	}

	integration, err := NewIntegration(config)
	if err != nil {
		log.Fatal(err)
	}

	integration.Do()
}
