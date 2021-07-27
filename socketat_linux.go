package socketat

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

// SocketAt creates a socket in the namespace passed as argument.
// ref: https://lore.kernel.org/patchwork/patch/217025/
func SocketAt(domain, typ, proto, ns int) (int, error) {
	var fn nsFunc
	fn = func() (interface{}, error) { return syscall.Socket(domain, typ, proto) }

	obj, err := DoAtNS(ns, fn)
	if err != nil {
		return -1, err
	}

	fd, ok := obj.(int)
	if !ok {
		return -1, err
	}
	return fd, nil
}

// ListenAt is like net.Listen but it creates a Listener inside the
// namespace passed as argument. The new connections accepted are
// still confined to the namespace, but the user doesn't have to worry about
// the problems of golang and goroutines.
func ListenAt(network, address string, ns int) (net.Listener, error) {
	var fn nsFunc
	fn = func() (interface{}, error) { return net.Listen(network, address) }

	obj, err := DoAtNS(ns, fn)
	if err != nil {
		return nil, err
	}

	ln, ok := obj.(net.Listener)
	if !ok {
		return nil, err
	}
	return ln, nil
}

// DialAt is like net.Dial but the connections is created inside the
// namespace passed as argument. The connection returned can be handled
// doesn't is goroutine safe and doesn't have the problems of golang
// with linux namespaces.
func DialAt(network, address string, ns int) (conn net.Conn, err error) {
	var fn nsFunc
	fn = func() (interface{}, error) {
		// dial inside the namespace and create the connection
		// it can't run goroutines or those will escape the namespace
		// https://github.com/golang/go/issues/44922
		var d net.Dialer
		// disable happy eyeballs beecause it creates goroutines
		d.FallbackDelay = -1
		// prefer go resolver to avoid issues with CGO
		r := net.DefaultResolver
		r.PreferGo = true
		r.Dial = d.DialContext
		d.Resolver = r
		return d.Dial(network, address)
	}

	obj, err := DoAtNS(ns, fn)
	if err != nil {
		return nil, err
	}

	conn, ok := obj.(net.Conn)
	if !ok {
		return nil, err
	}
	return conn, nil

}

// get current namespace assume runtime.LockOSThread() is held
// /proc/self/ns/net returns the namespace of the main thread, not
// of whatever thread this goroutine is running on.  Make sure we
// use the thread's net namespace since the thread is switching around
// https://github.com/containernetworking/plugins/blob/master/pkg/ns/ns_linux.go
func getCurrentNS() (int, error) {
	origin, err := os.Open(fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), unix.Gettid()))
	if err != nil {
		return -1, err
	}
	return int(origin.Fd()), err
}

type nsFunc func() (interface{}, error)

// DoAtNS execute a function inside an specific namespace
// goroutines spawned inside linnux namespace can escape
// the namespace, fn() should not spawn any goroutine inside
// https://www.weave.works/blog/linux-namespaces-golang-followup
func DoAtNS(ns int, fn nsFunc) (obj interface{}, err error) {
	// lock the thread so we don't switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// get current namespace
	origin, err := getCurrentNS()
	if err != nil {
		return nil, err
	}

	// enter the new namespace
	err = unix.Setns(ns, syscall.CLONE_NEWNET)
	if err != nil {
		return nil, err
	}
	// always come back to the original namespace
	defer func() {
		if e := unix.Setns(origin, syscall.CLONE_NEWNET); e != nil {
			if err != nil {
				err = fmt.Errorf("Error returning to original namespace %v: %w", e, err)
			} else {
				err = e
			}
		}
	}()

	// execute the function inside the namespace
	// it should not have any goroutine or it will
	// escape the namespace
	obj, err = fn()
	return
}
