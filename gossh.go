package gossh

import (
    "fmt"
    "io/ioutil"
    "net"

    "golang.org/x/crypto/ssh"
)

// HostConfig holds the host address and optional specific credentials
type HostConfig struct {
    Address    string
    User       string
    PrivateKey string // Path to private key file (for key-based auth)
    Password   string // For password-based auth
}

// Client struct to hold the configuration
type Client struct {
    Config     *ssh.ClientConfig
    JumpHosts  []HostConfig
    TargetHost HostConfig
}

// createClientConfig creates an SSH client configuration
func createClientConfig(user, privateKeyPath, password string) (*ssh.ClientConfig, error) {
    var authMethods []ssh.AuthMethod

    if privateKeyPath != "" {
        key, err := ioutil.ReadFile(privateKeyPath)
        if err != nil {
            return nil, fmt.Errorf("unable to read private key: %w", err)
        }

        signer, err := ssh.ParsePrivateKey(key)
        if err != nil {
            return nil, fmt.Errorf("unable to parse private key: %w", err)
        }

        authMethods = append(authMethods, ssh.PublicKeys(signer))
    }

    if password != "" {
        authMethods = append(authMethods, ssh.Password(password))
    }

    if len(authMethods) == 0 {
        return nil, fmt.Errorf("no authentication method provided")
    }

    return &ssh.ClientConfig{
        User:            user,
        Auth:            authMethods,
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }, nil
}

// getClientConfig returns the specific client config for a host, or the default if none is specified
func (client *Client) getClientConfig(host HostConfig) (*ssh.ClientConfig, error) {
    // Use specific credentials if provided
    if host.User != "" && (host.PrivateKey != "" || host.Password != "") {
        return createClientConfig(host.User, host.PrivateKey, host.Password)
    }
    // Fallback to default configuration
    return client.Config, nil
}

// Connect handles the connection through jump hosts to the target host
func (client *Client) Connect() (*ssh.Client, error) {
    var currentClient *ssh.Client
    var err error

    for _, host := range client.JumpHosts {
        config, err := client.getClientConfig(host)
        if err != nil {
            return nil, err
        }

        address := host.Address + ":22"
        if currentClient == nil {
            currentClient, err = ssh.Dial("tcp", address, config)
        } else {
            var conn net.Conn
            conn, err = currentClient.Dial("tcp", address)
            if err != nil {
                return nil, fmt.Errorf("failed to dial from current host to %s: %w", address, err)
            }

            ncc, chans, reqs, err := ssh.NewClientConn(conn, address, config)
            if err != nil {
                return nil, fmt.Errorf("failed to create client connection to %s: %w", address, err)
            }
            currentClient = ssh.NewClient(ncc, chans, reqs)
        }

        if err != nil {
            return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
        }
    }

    targetConfig, err := client.getClientConfig(client.TargetHost)
    if err != nil {
        return nil, err
    }

    var targetClient *ssh.Client
    if currentClient == nil {
        // Direct connection to the target host
        targetClient, err = ssh.Dial("tcp", client.TargetHost.Address+":22", targetConfig)
        if err != nil {
            return nil, fmt.Errorf("failed to dial to target host: %w", err)
        }
    } else {
        // Connect to the target host through the last jump host
        targetConn, err := currentClient.Dial("tcp", client.TargetHost.Address+":22")
        if err != nil {
            return nil, fmt.Errorf("failed to dial to target host: %w", err)
        }

        ncc, chans, reqs, err := ssh.NewClientConn(targetConn, client.TargetHost.Address+":22", targetConfig)
        if err != nil {
            return nil, fmt.Errorf("failed to create client connection to target host: %w", err)
        }
        targetClient = ssh.NewClient(ncc, chans, reqs)
    }

    return targetClient, nil
}

// ExecuteCommand executes a command on the target host
func (client *Client) ExecuteCommand(command string) (string, error) {
    sshClient, err := client.Connect()
    if err != nil {
        return "", err
    }
    defer sshClient.Close()

    session, err := sshClient.NewSession()
    if err != nil {
        return "", fmt.Errorf("failed to create session on target host: %w", err)
    }
    defer session.Close()

    output, err := session.CombinedOutput(command)
    if err != nil {
        return "", fmt.Errorf("failed to execute command: %w", err)
    }

    return string(output), nil
}

// SSHClient returns vanilla SSH client
func (client *Client) SSHClient() (*ssh.Client, error) {
    sshClient, err := client.Connect()
    if err != nil {
        return nil, err
    }
    defer sshClient.Close()

    return sshClient, nil
}

// SSHSession returns vanilla SSH session
func (client *Client) SSHSession() (*ssh.Session, error) {
    sshClient, err := client.Connect()
    if err != nil {
        return nil, err
    }
    defer sshClient.Close()

    session, err := sshClient.NewSession()
    if err != nil {
        return nil, fmt.Errorf("failed to create session on target host: %w", err)
    }
    defer session.Close()

    return session, nil
}

// NewJumpClient creates a new SSH client with the given configuration
func NewJumpClient(targetHost HostConfig, jumpHosts []HostConfig) (*Client, error) {
    // Create client with the target host's configuration
    config, err := createClientConfig(targetHost.User, targetHost.PrivateKey, targetHost.Password)
    if err != nil {
        return nil, err
    }

    return &Client{
        Config:     config,
        JumpHosts:  jumpHosts,
        TargetHost: targetHost,
    }, nil
}

// NewDirectClient creates an SSH client for a direct connection to the target host
func NewDirectClient(targetHost HostConfig) (*Client, error) {
    config, err := createClientConfig(targetHost.User, targetHost.PrivateKey, targetHost.Password)
    if err != nil {
        return nil, err
    }

    return &Client{
        Config:     config,
        JumpHosts:  []HostConfig{}, // No jump hosts
        TargetHost: targetHost,
    }, nil
}

// NewClient creates an SSH client based on the presence of jump hosts
func NewClient(targetHost HostConfig, jumpHosts []HostConfig) (*Client, error) {
    if jumpHosts != nil {
        return NewJumpClient(targetHost, jumpHosts)
    }
    return NewDirectClient(targetHost)
}
