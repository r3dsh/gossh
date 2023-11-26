package gossh

import (
    "fmt"
    "io"
    "log"
    "net"
)

// ProxyRemotePortToLocal sets up port forwarding from a remote host:port to a local port
func (client *Client) ProxyRemotePortToLocal(localPort string, remoteHost string, remotePort string) error {
    sshClient, err := client.Connect()
    if err != nil {
        return err
    }
    defer sshClient.Close()

    // Start local listener on the specified port
    localListener, err := net.Listen("tcp", "localhost:"+localPort)
    if err != nil {
        return fmt.Errorf("failed to set up local listener: %w", err)
    }
    defer localListener.Close()

    for {
        localConn, err := localListener.Accept()
        if err != nil {
            return fmt.Errorf("failed to accept local connection: %w", err)
        }

        // Handle the connection in a new goroutine
        go func() {
            defer localConn.Close()

            // Connect to remote host
            remoteConn, err := sshClient.Dial("tcp", remoteHost+":"+remotePort)
            if err != nil {
                log.Printf("failed to connect to remote host: %v", err)
                return
            }
            defer remoteConn.Close()

            // Copy data between local and remote
            go io.Copy(remoteConn, localConn)
            io.Copy(localConn, remoteConn)
        }()
    }
}

// ProxyRemoteUnixSocketToLocal forwards a Unix socket over SSH to a local TCP port
func (client *Client) ProxyRemoteUnixSocketToLocal(localPort string, remoteSocketPath string) error {
    sshClient, err := client.Connect()
    if err != nil {
        return err
    }
    defer sshClient.Close()

    // Start local listener on the specified port
    localListener, err := net.Listen("tcp", "localhost:"+localPort)
    if err != nil {
        return fmt.Errorf("failed to set up local listener: %w", err)
    }
    defer localListener.Close()

    for {
        localConn, err := localListener.Accept()
        if err != nil {
            return fmt.Errorf("failed to accept local connection: %w", err)
        }

        // Handle the connection in a new goroutine
        go func() {
            defer localConn.Close()

            // Connect to remote Unix socket
            remoteConn, err := sshClient.Dial("unix", remoteSocketPath)
            if err != nil {
                log.Printf("failed to connect to remote Unix socket: %v", err)
                return
            }
            defer remoteConn.Close()

            // Copy data between local and remote
            go io.Copy(remoteConn, localConn)
            io.Copy(localConn, remoteConn)
        }()
    }
}
