package gossh

import (
    "fmt"
    "io"
    "os"
    "path"
)

// SCPUpload streams data from an io.Reader to a remote file using SCP protocol
func (client *Client) SCPUpload(reader io.Reader, size int64, remoteFilePath string) error {
    sshClient, err := client.Connect()
    if err != nil {
        return err
    }
    defer sshClient.Close()

    // Start an SSH session
    session, err := sshClient.NewSession()
    if err != nil {
        return fmt.Errorf("failed to create session on target host: %w", err)
    }
    defer session.Close()

    // Set up SCP command on remote host
    scpCmd := fmt.Sprintf("scp -t %s", remoteFilePath)
    stdinPipe, err := session.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to open stdin pipe: %w", err)
    }
    defer stdinPipe.Close()

    // Start the remote SCP process
    if err := session.Start(scpCmd); err != nil {
        return fmt.Errorf("failed to start SCP command on remote host: %w", err)
    }

    // Send file permissions (0644), size, and name
    // Using default permissions here as we don't have the actual file's metadata
    fmt.Fprintf(stdinPipe, "C0644 %d %s\n", size, path.Base(remoteFilePath))

    // Stream the data
    _, err = io.Copy(stdinPipe, reader)
    if err != nil {
        return fmt.Errorf("failed to copy data to remote host: %w", err)
    }
    fmt.Fprint(stdinPipe, "\x00") // Send transfer completion signal
    stdinPipe.Close()             // Close the pipe

    // Wait for SCP command to complete
    if err := session.Wait(); err != nil {
        return fmt.Errorf("SCP command finished with error: %w", err)
    }

    return nil
}

// SCPUploadFile uploads a local file to the remote server using SCP
func (client *Client) SCPUploadFile(localFilePath, remoteFilePath string) error {
    // Open the local file
    file, err := os.Open(localFilePath)
    if err != nil {
        return fmt.Errorf("failed to open local file: %w", err)
    }
    defer file.Close()

    // Get file info for size
    fileInfo, err := file.Stat()
    if err != nil {
        return fmt.Errorf("failed to get file info: %w", err)
    }

    // Call SCPUpload with the file's reader and size
    return client.SCPUpload(file, fileInfo.Size(), remoteFilePath)
}
