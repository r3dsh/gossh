package gossh

import (
    "bufio"
    "fmt"
    "io"

    "golang.org/x/crypto/ssh"
)

// CombinedOutputHandler is for combined stdout and stderr lines
type CombinedOutputHandler func(line string)

// SeparateOutputHandler is for separate stdout and stderr lines
type SeparateOutputHandler func(stdoutLine string, stderrLine string)

// StreamCommand executes a command on the target host and handles output based on the provided handlers
func (client *Client) StreamCommand(command string, combinedHandler CombinedOutputHandler, separateHandler SeparateOutputHandler) error {
    sshClient, err := client.Connect()
    if err != nil {
        return err
    }
    defer sshClient.Close()

    session, err := sshClient.NewSession()
    if err != nil {
        return fmt.Errorf("failed to create session on target host: %w", err)
    }
    defer session.Close()

    if separateHandler != nil {
        // Stream separate stdout and stderr using the separateHandler
        return streamSeparateOutput(session, command, separateHandler)
    } else if combinedHandler != nil {
        // Stream combined output using the combinedHandler
        return streamCombinedOutput(session, command, combinedHandler)
    }

    return fmt.Errorf("no valid handler provided")
}

// streamCombinedOutput streams combined stdout and stderr output
func streamCombinedOutput(session *ssh.Session, command string, handler CombinedOutputHandler) error {
    stdoutPipe, err := session.StdoutPipe()
    if err != nil {
        return fmt.Errorf("unable to setup stdout for session: %w", err)
    }
    stderrPipe, err := session.StderrPipe()
    if err != nil {
        return fmt.Errorf("unable to setup stderr for session: %w", err)
    }

    if err := session.Start(command); err != nil {
        return fmt.Errorf("failed to start command: %w", err)
    }

    // Stream both stdout and stderr to the same handler
    go handleStream(stdoutPipe, handler)
    go handleStream(stderrPipe, handler)

    if err := session.Wait(); err != nil {
        return fmt.Errorf("command finished with error: %w", err)
    }

    return nil
}

// streamSeparateOutput streams separate stdout and stderr output
func streamSeparateOutput(session *ssh.Session, command string, handler SeparateOutputHandler) error {
    stdoutPipe, err := session.StdoutPipe()
    if err != nil {
        return fmt.Errorf("unable to setup stdout for session: %w", err)
    }
    stderrPipe, err := session.StderrPipe()
    if err != nil {
        return fmt.Errorf("unable to setup stderr for session: %w", err)
    }

    if err := session.Start(command); err != nil {
        return fmt.Errorf("failed to start command: %w", err)
    }

    go handleStream(stdoutPipe, func(line string) { handler(line, "") })
    go handleStream(stderrPipe, func(line string) { handler("", line) })

    if err := session.Wait(); err != nil {
        return fmt.Errorf("command finished with error: %w", err)
    }

    return nil
}

// handleStream reads from the given reader and calls the handler function for each line
func handleStream(reader io.Reader, handler CombinedOutputHandler) {
    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
        handler(scanner.Text())
    }
}

// SendStringToFile sends the provided data to a file on the remote server
func (client *Client) SendStringToFile(data, remoteFilePath string) error {
    sshClient, err := client.Connect()
    if err != nil {
        return err
    }
    defer sshClient.Close()

    session, err := sshClient.NewSession()
    if err != nil {
        return fmt.Errorf("failed to create session on target host: %w", err)
    }
    defer session.Close()

    // Using 'cat' to write the data to the remote file
    cmd := fmt.Sprintf("cat > %s", remoteFilePath)
    stdinPipe, err := session.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to open stdin pipe: %w", err)
    }

    if err := session.Start(cmd); err != nil {
        return fmt.Errorf("failed to start command on remote host: %w", err)
    }

    // Write the data to the stdin of the 'cat' command
    _, err = stdinPipe.Write([]byte(data))
    if err != nil {
        return fmt.Errorf("failed to write to stdin pipe: %w", err)
    }
    stdinPipe.Close()

    // Wait for the command to finish
    if err := session.Wait(); err != nil {
        return fmt.Errorf("command finished with error: %w", err)
    }

    return nil
}

// StreamToRemoteFile streams data from an io.Reader to a file on the remote server
// use it for small text files only.
func (client *Client) StreamToRemoteFile(reader io.Reader, remoteFilePath string) error {
    sshClient, err := client.Connect()
    if err != nil {
        return err
    }
    defer sshClient.Close()

    session, err := sshClient.NewSession()
    if err != nil {
        return fmt.Errorf("failed to create session on target host: %w", err)
    }
    defer session.Close()

    // Using 'cat' to write the data to the remote file
    cmd := fmt.Sprintf("cat > %s", remoteFilePath)
    stdinPipe, err := session.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to open stdin pipe: %w", err)
    }

    if err := session.Start(cmd); err != nil {
        return fmt.Errorf("failed to start command on remote host: %w", err)
    }

    // Stream the data to the stdin of the 'cat' command
    _, err = io.Copy(stdinPipe, reader)
    stdinPipe.Close()
    if err != nil {
        return fmt.Errorf("failed to stream data to stdin pipe: %w", err)
    }

    // Wait for the command to finish
    if err := session.Wait(); err != nil {
        return fmt.Errorf("command finished with error: %w", err)
    }

    return nil
}
