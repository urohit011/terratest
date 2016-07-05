package ssh

import (
	"golang.org/x/crypto/ssh"
	"fmt"
	"io"
	"net"
	"reflect"
	"log"
)

type SshConnectionOptions struct {
	Username   		string
	Address    		string
	Port       		int
	AuthMethods   		[]ssh.AuthMethod
	Command			string
	JumpHost		*SshConnectionOptions
}

func (options *SshConnectionOptions) ConnectionString() string {
	return fmt.Sprintf("%s:%d", options.Address, options.Port)
}

// A container object for all resources created by an SSH session. The reason we need this is so that we can do a
// single defer in a top-level method that calls the Cleanup method to go through and ensure all of these resources are
// released and cleaned up.
type SshSession struct {
	Options		*SshConnectionOptions
	Client 		*ssh.Client
	Session 	*ssh.Session
	JumpHost	*JumpHostSession
}

func (sshSession *SshSession) Cleanup(logger *log.Logger) {
	if sshSession == nil {
		return
	}


	// Closing the session may result in an EOF error if it's already closed (e.g. due to hitting CTRL + D), so
	// don't report those errors, as there is nothing actually wrong in that case.
	Close(sshSession.Session, logger, io.EOF.Error())
	Close(sshSession.Client, logger)
	sshSession.JumpHost.Cleanup(logger)
}

type JumpHostSession struct {
	JumpHostClient 		*ssh.Client
	HostVirtualConnection 	net.Conn
	HostConnection		ssh.Conn
}

func (jumpHost *JumpHostSession) Cleanup(logger *log.Logger) {
	if jumpHost == nil {
		return
	}

	// Closing a connection may result in an EOF error if it's already closed (e.g. due to hitting CTRL + D), so
	// don't report those errors, as there is nothing actually wrong in that case.
	Close(jumpHost.HostConnection, logger, io.EOF.Error())
	Close(jumpHost.HostVirtualConnection, logger, io.EOF.Error())
	Close(jumpHost.JumpHostClient, logger)
}

type Closeable interface {
	Close() error
}

func Close(closeable Closeable, logger *log.Logger, ignoreErrors ...string) {
	if interfaceIsNil(closeable) {
		return
	}

	if err := closeable.Close(); err != nil && !contains(ignoreErrors, err.Error()) {
		logger.Printf("Error closing %s: %s", closeable, err.Error())
	}
}

func contains(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

// Go is a shitty language. Checking an interface directly against nil does not work, and if you don't know the exact
// types the interface may be ahead of time, the only way to know if you're dealing with nil is to use reflection.
// http://stackoverflow.com/questions/13476349/check-for-nil-and-nil-interface-in-go
func interfaceIsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
