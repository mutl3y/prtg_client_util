package util

import (
	"fmt"
	"github.com/PaesslerAG/go-prtg-sensor-api"
	"github.com/appleboy/easyssh-proxy"
	"os"
	"strings"
	"time"
)

type conn struct{ easyssh.MakeConfig }

type SshStruct = struct {
	User     string
	Server   string
	Key      string
	KeyPath  string
	Port     string
	Password string
	Timeout  time.Duration
}

func NewCon(dest, proxy SshStruct) *conn {
	c := easyssh.MakeConfig{}
	c.User = dest.User
	c.Server = dest.Server
	c.Key = dest.Key
	c.KeyPath = dest.KeyPath
	c.Port = dest.Port
	c.Password = dest.Password
	c.Timeout = dest.Timeout
	c.Proxy = proxy
	mc := conn{c}
	return &mc
}

func (ssh *conn) Remote(command string, timeout time.Duration) error {
	dir := "/var/prtg/scriptsxml/prtg_client_util "
	stdoutChan, stderrChan, doneChan, errChan, err := ssh.Stream(dir+command, timeout)
	// Handle errors
	if err != nil {
		FailRemote(fmt.Errorf("%v %v", err, dir))
		return fmt.Errorf("Can't run remote command: %v", err.Error())
	} else {
		// read from the output channel until the done signal is passed
		isTimeout := true
	loop:
		for {
			select {
			case isTimeout = <-doneChan:
				break loop
			case outline := <-stdoutChan:
				fmt.Println(outline)
			case errline := <-stderrChan:
				err = fmt.Errorf(errline)
			case err = <-errChan:
			}
		}

		// get exit code or command error.
		if err != nil {
			if err.Error() == "Process exited with status 127" {
				err = fmt.Errorf("command not found")
			}
			FailRemote(fmt.Errorf("%v %v", err, dir))
			return err
		}

		// command time out
		if !isTimeout {
			err := fmt.Errorf("error: command timeout")
			FailRemote(err)
			return err
		}
	}

	return nil
}

func FailRemote(err error) {
	r := prtg.SensorResponse{}
	r.SensorResult.Error = 1
	r.SensorResult.Text = fmt.Sprintf("%v", err)

	fmt.Println(r.String())
}

type platformSpec struct {
	GOOS   string
	GOARCH string
}

func (ssh *conn) getUname() (platformSpec, error) {
	platform, errStr, isTimeout, err := ssh.Run("uname -s")
	// Handle errors
	if err != nil {
		return platformSpec{}, fmt.Errorf("can't run remote command: %v %v", err.Error(), errStr)
	}
	if !isTimeout {
		err := fmt.Errorf("error: command timeout")
		FailRemote(err)
		return platformSpec{}, err
	}

	// get processor family
	arch, errStr, isTimeout, err := ssh.Run("arch")
	// Handle errors
	if err != nil {
		FailRemote(fmt.Errorf("%v ", err))
		return platformSpec{}, fmt.Errorf("can't run remote command: %v %v", err.Error(), errStr)
	}

	if !isTimeout {
		err := fmt.Errorf("error: command timeout")
		FailRemote(err)
		return platformSpec{}, err
	}

	platform = strings.ToLower(strings.TrimSpace(platform))
	arch = strings.ToLower(strings.TrimSpace(arch))

	if platform == "" || arch == "" {
		return platformSpec{}, fmt.Errorf("could not id platform and processor family using uname")
	}

	switch arch {
	case "x86_64":
		fallthrough
	case "x64":
		arch = "amd64"

	case "i386":
		fallthrough
	case "i686":
		arch = "386"

	case "armv6l":
		fallthrough
	case "armv7l":
		fallthrough
	case "armv8l":
		arch = "arm64"

	default:
		return platformSpec{}, fmt.Errorf("arcitecture not implemented yet %v", arch)
	}

	switch {
	case platform == "darwin":
	case platform == "linux":
	case strings.Contains(platform, "nt"):
		platform = "windows"
	default:
		return platformSpec{}, fmt.Errorf("platform not implemented %v", platform)
	}

	return platformSpec{GOOS: platform, GOARCH: arch}, nil
}

func (ssh *conn) Deploy(dir string) error {
	plat, err := ssh.getUname()
	if err != nil {
		if strings.Contains(err.Error(), "Name or service not known") {
			return fmt.Errorf("unknown host. try ip %v", err)
		}
		return fmt.Errorf("failed to get remote platform details \n%v", err)
	}

	if err := ssh.mkScriptsXmlDir(ssh.User); err != nil {
		return err
	}

	fn := strings.Join([]string{"prtg_client_util", plat.GOOS, plat.GOARCH}, "-")

	fnpath := strings.Join([]string{dir, fn}, string(os.PathSeparator))
	target := "/var/prtg/scriptsxml/prtg_client_util"
	fmt.Printf("copying %v to %v %v", fnpath, ssh.Server, target)
	err = ssh.Scp(fnpath, target)
	if err != nil {
		fmt.Printf("%v %v", fnpath, target)
		return fmt.Errorf("failed to scp file %v ", err)
	}

	_, errStr, isTimeout, err := ssh.Run("chmod 755 " + target)
	if (err != nil) || errStr != "" {
		return fmt.Errorf("failed chmoding directory %v", err)
	}
	if !isTimeout {
		err := fmt.Errorf("error: command timeout")
		return err
	}
	return nil
}

func (ssh *conn) chown(file, user, group string) error {
	c := fmt.Sprintf("chown -R %v:%v %v ", user, group, file)
	_, errStr, isTimeout, err := ssh.Run(c)
	if (err != nil) || errStr != "" {
		fmt.Println(errStr)
		return fmt.Errorf("failed chowning %v %v", file, err)
	}
	if !isTimeout {
		err := fmt.Errorf("error: command timeout")
		return err
	}
	return nil
}

func (ssh *conn) mkScriptsXmlDir(user string) error {
	_, errStr, isTimeout, err := ssh.Run("mkdir -p /var/prtg/scriptsxml")
	if (err != nil) || errStr != "" {
		return fmt.Errorf("error executing mkdir %v %v", err, errStr)
	}
	if !isTimeout {
		return fmt.Errorf("error: command timeout")
	}
	if err := ssh.chown("/var/prtg/scriptsxml", user, user); err != nil {
		return err
	}

	return nil
}

func (ssh *conn) CreateUsers(tuser, tpass, juser, jpass string) error {
	jssh := easyssh.MakeConfig{
		User:     juser,
		Server:   ssh.MakeConfig.Proxy.Server,
		Port:     ssh.MakeConfig.Proxy.Port,
		Password: jpass,
		Timeout:  time.Minute,
	}

	if jpass == "" {
		jssh = easyssh.MakeConfig{}
	}

	tssh := easyssh.MakeConfig{
		User:     tuser,
		Server:   ssh.MakeConfig.Server,
		Port:     ssh.MakeConfig.Port,
		Password: tpass,
		Timeout:  time.Minute,
		Proxy: easyssh.DefaultConfig{
			User:     jssh.User,
			Server:   jssh.Server,
			Port:     jssh.Port,
			Password: jssh.Password,
			Timeout:  jssh.Timeout,
		},
	}

	target := &conn{tssh}
	jumphost := &conn{jssh}

	if jumphost.Server != "" {
		err := jumphost.createUser(ssh.MakeConfig.Proxy.User, ssh.MakeConfig.Proxy.Password)
		if err != nil {
			return fmt.Errorf("jumphost: %v", err)
		}
	}

	err := target.createUser(ssh.MakeConfig.User, ssh.MakeConfig.Password)
	if err != nil {
		return fmt.Errorf("target: %v", err)
	}
	return target.mkScriptsXmlDir(ssh.User)
}

func (ssh *conn) createUser(usr, passwd string) error {
	fmt.Printf("creating user %v on %v\n", usr, ssh.Server)

	_, errStr, isTimeout, err := ssh.Run(fmt.Sprintf("/usr/sbin/useradd -m -l %v", usr))
	if (err != nil) || errStr != "" {
		if strings.Contains(errStr, "already exists") {
			return ssh.updateUser(usr, passwd)
		}
		if err != nil {
			if strings.Contains(err.Error(), "service not known") {
				return fmt.Errorf("createUser: ssh connection error, unknown host, try ip")
			}
		}

		return fmt.Errorf("failed createUser %v %v", usr, err)
	}
	if !isTimeout {
		err := fmt.Errorf("error: command timeout")
		return err
	}
	_, errStr, isTimeout, err = ssh.Run(fmt.Sprintf(" echo %v:%v | /usr/sbin/chpasswd", usr, passwd))
	if (err != nil) || errStr != "" {
		return fmt.Errorf("failed setting password on %v \n%v %v", ssh.MakeConfig.Server, err, errStr)
	}
	if !isTimeout {
		err := fmt.Errorf("error: command timeout")
		return err
	}

	_ = ssh.Deploy("")

	return err
}

func (ssh *conn) updateUser(usr, passwd string) error {

	if passwd == "" {
		return fmt.Errorf("don't use blank passwords %v on %v", usr, ssh.MakeConfig.Server)
	}
	_, errStr, isTimeout, err := ssh.Run(fmt.Sprintf(" echo %v:%v | /usr/sbin/chpasswd", usr, passwd))
	if (err != nil) || errStr != "" {
		if !strings.Contains(errStr, "password not changed") {
			fmt.Println(errStr)
			return fmt.Errorf("failed changing password %v on %v  \n%v", usr, ssh.MakeConfig.Server, err)
		}

	}
	if !isTimeout {
		err := fmt.Errorf("error: command timeout")
		return err
	}
	return nil
}
