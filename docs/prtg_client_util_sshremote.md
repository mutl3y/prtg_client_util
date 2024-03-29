## prtg_client_util sshremote

run command remotely through ssh tunnel via jumphost / proxy

### Synopsis

run command remotely through ssh tunnel

Functionality is restricted to running prtg_client_util remotely from /var/prtg/scriptsxml.
a copy of the app must be placed in that folder with execute permissions for remote user

Be aware this effectively allows PRTG to perform remote code execution.

Only a basic user account should be created for this, no sudo rights etc.

RSA key authentication is preferred however authentication will fall back to password if supplied



```
prtg_client_util sshremote [flags]
```

### Options

```
  -h, --help               help for sshremote
  -f, --j_KeyFile string   jumphost - private key file
  -i, --j_host string      jumphost - ip
  -p, --j_pass string      jumphost - password (default "prtgUtil")
  -o, --j_port string      jumphost - ssh port (default "22")
  -u, --j_user string      jumphost - user (default "prtgUtil")
  -R, --run string         command to run on remote host (default "ping")
  -F, --t_KeyFile string   target - private key file hint:C:\Users\mark\.ssh\it_rsa
  -I, --t_host string      target - ip (default "localhost")
  -P, --t_pass string      target - password (default "prtgUtil")
  -O, --t_port string      target - ssh port (default "22")
  -U, --t_user string      target - user (default "prtgUtil")
```

### Options inherited from parent commands

```
  -d, --debug              command line output
  -t, --timeout duration   timeout string eg 500ms (default 500ms)
```

### SEE ALSO

* [prtg_client_util](prtg_client_util.md)	 - simple prtg tests for remote nodes

