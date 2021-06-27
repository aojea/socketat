package socketat

import (
	"log"
	"net"
	"os"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

type SocketAt struct {
	Namespace int
}

func New(namespace int) *SocketAt {
	return &SocketAt{
		Namespace: namespace,
	}
}

func (s SocketAt) Socket(domain, typ, proto int) (int, error) {
	// get current namespace
	origin, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return -1, err
	}
	// lock the thread so we don't switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// enter the new namespace
	err = unix.Setns(int(s.Namespace), syscall.CLONE_NEWNET)
	if err != nil {
		return -1, err
	}
	// always come back to the original namespace
	defer func() {
		if e := unix.Setns(int(origin.Fd()), syscall.CLONE_NEWNET); e != nil {
			log.Printf("failed to recover netns: %+v", e)
		}
	}()
	// open the socket in the new namespace and return its file descriptor
	return syscall.Socket(domain, typ, proto)
}

func ListenAt(network, address string, ns int) (net.Listener, error) {
	// get current namespace
	origin, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return nil, err
	}
	// lock the thread so we don't switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// enter the new namespace
	err = unix.Setns(int(ns), syscall.CLONE_NEWNET)
	if err != nil {
		return nil, err
	}
	// always come back to the original namespace
	defer func() {
		if e := unix.Setns(int(origin.Fd()), syscall.CLONE_NEWNET); e != nil {
			log.Printf("failed to recover netns: %+v", e)
		}
	}()
	return net.Listen(network, address)
}

func DialAt(network, address string, ns int) (net.Conn, error) {
	// get current namespace
	origin, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return nil, err
	}
	// lock the thread so we don't switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// enter the new namespace
	err = unix.Setns(int(ns), syscall.CLONE_NEWNET)
	if err != nil {
		return nil, err
	}
	// always come back to the original namespace
	defer func() {
		if e := unix.Setns(int(origin.Fd()), syscall.CLONE_NEWNET); e != nil {
			log.Printf("failed to recover netns: %+v", e)
		}
	}()
	var d net.Dialer
	d.FallbackDelay = -1
	return d.Dial(network, address)
}
