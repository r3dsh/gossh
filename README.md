# GO SSH

**Yet another golang.org/x/crypto/ssh wrapper.**

While trivial using modern SSH Client, I couldn't find any ready to use, nicely wrapped ssh library that supports jumping through multiple hosts.

```go
package main

import "github.com/r3dsh/gossh"

func main() {
    // Example usage
    jumpHosts := []gossh.HostConfig{
        {Address: "10.5.50.80", User: "midrange", PrivateKey: "conf/keys/id_rsa"},
        {Address: "10.5.50.79", User: "midrange", PrivateKey: "conf/keys/id_rsa"}, // Will use default credentials
    }
    targetHost := gossh.HostConfig{Address: "10.5.50.76", User: "midrange", PrivateKey: "conf/keys/id_rsa"}

    client, err := gossh.NewClient(targetHost, jumpHosts)
    if err != nil {
        log.Fatalf("Failed to create SSH client: %v", err)
    }
    // direct connection, withtout jumphosts
    // mazeClient, err := gossh.NewClient(gossh.HostConfig{Address: "10.5.50.77", User: "x", Password: "<PASSWORD>"}, nil)
    // if err != nil {
    //     log.Fatalf("Failed to create SSH client: %v", err)
    // }

    output, err := client.ExecuteCommand("whoami")
    if err != nil {
        log.Fatalf("Failed to execute command: %v", err)
    }

    fmt.Printf("Output from the target host: %s", output)
}
```

This is basically GO implementation of:
```shell
ssh -J midrange@10.5.50.80,midrange@10.5.50.79 midrange@10.5.50.76 whoami
```

If you work with docker programmatically, this one might be interesting:
```go
func main() {
    // Example usage
    jumpHosts := []gossh.HostConfig{
        {Address: "10.5.50.80", User: "midrange", PrivateKey: "conf/keys/id_rsa"},
        {Address: "10.5.50.79", User: "midrange", PrivateKey: "conf/keys/id_rsa"}, // Will use default credentials
    }
    targetHost := gossh.HostConfig{Address: "10.5.50.76", User: "midrange", PrivateKey: "conf/keys/id_rsa"}
    
    client, err := gossh.NewClient(targetHost, jumpHosts)
        if err != nil {
        log.Fatalf("Failed to create SSH client: %v", err)
    }
    
    // Forward remote Unix socket to local port
    go client.ProxyRemoteUnixSocketToLocal("9090", "/var/run/docker.sock")
    
    // Block forever (or until your application has a reason to exit)
    select {}
}
```

Or, when working programmatically with kubernetes clusters:
```go
func main() {
    // Example usage
    jumpHosts := []gossh.HostConfig{
        {Address: "10.5.50.80", User: "midrange", PrivateKey: "conf/keys/id_rsa"},
        {Address: "10.5.50.79", User: "midrange", PrivateKey: "conf/keys/id_rsa"}, // Will use default credentials
    }
    targetHost := gossh.HostConfig{Address: "10.5.50.76", User: "midrange", PrivateKey: "conf/keys/id_rsa"}
    
    client, err := gossh.NewClient(targetHost, jumpHosts)
        if err != nil {
        log.Fatalf("Failed to create SSH client: %v", err)
    }
    
    // Forward remote host:port to local port
    go client.ProxyRemotePortToLocal("8443", "localhost", "6443")
    
    // Block forever (or until your application has a reason to exit)
    select {}
}
```

If you don't like my syntactic sugar, you can get vanilla SSH client and/or session on target host using:
```go
    // vanilla SSH client
    client, err := client.SSHClient()
    
    // vanilla SSH client session
    client, err := client.SSHSession()
```

## DISCLAIMER: This my is first, one hour iteration on the library.
