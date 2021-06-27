package socketat

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const nsMax = 100

func TestNoLookup(t *testing.T) {

	// Save the current network namespace
	origns, _ := netns.Get()
	defer origns.Close()
	defer netns.Set(origns)

	// Create network namespaces
	nsArray := []netns.NsHandle{}
	errs := make(chan error, 2*nsMax)
	for i := 0; i < nsMax; i++ {
		ns := createNamespace(t)
		defer ns.Close()
		nsArray = append(nsArray, ns)
	}

	// Create listeners in all namespaces
	for _, newns := range nsArray {
		newns := newns
		listener, err := ListenAt("tcp", "localhost:8080", int(newns))
		if err != nil {
			t.Fatalf("Error listening to address localhost:8008")
		}
		defer listener.Close()
		go func() {
			conn1, err := listener.Accept()
			if err != nil {
				errs <- err
			}
			buf := make([]byte, 1024)
			n, err := conn1.Read(buf)
			if err != nil {
				errs <- err
			}

			if string(buf[:n]) != fmt.Sprintf("SYN-NS%d", int(newns)) {
				errs <- fmt.Errorf("Received %s expected SYN-NS%d", string(buf), int(newns))
			}
			conn1.Write([]byte(fmt.Sprintf("ACK-NS%d", int(newns))))
			conn1.Close()
			errs <- nil
		}()
	}
	time.Sleep(500 * time.Millisecond)

	// Dial in all namespaces
	for _, newns := range nsArray {
		newns := newns
		go func() {
			conn, err := DialAt("tcp", "localhost:8080", int(newns))
			if err != nil {
				errs <- err
			}
			defer conn.Close()
			conn.Write([]byte(fmt.Sprintf("SYN-NS%d", int(newns))))
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				errs <- err
			}
			if string(buf[:n]) != fmt.Sprintf("ACK-NS%d", int(newns)) {
				errs <- fmt.Errorf("Received %s expected ACK-NS%d", string(buf), int(newns))
			}
			errs <- nil
		}()
	}
	// wait for all NS to finish
	for i := 0; i < 2*nsMax; i++ {
		err := <-errs
		if err != nil {
			t.Fatal(err)
		}
	}

}

func createNamespace(t *testing.T) netns.NsHandle {
	// save current namespace and return on exit
	origns, _ := netns.Get()
	defer netns.Set(origns)
	// Create a new network namespace
	newns, _ := netns.New()
	// set up the network inside the namespace
	link, err := netlink.LinkByName("lo")
	if err != nil {
		t.Fatalf("Failed to find \"lo\" in new netns: %v", err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		t.Fatalf("Failed to bring up \"lo\" in new netns: %v", err)
	}
	// Check that we are inside the network namespace
	ifaces, _ := net.Interfaces()
	if (len(ifaces) != 1) && (ifaces[0].Name != "lo") {
		t.Fatalf("We are not inside the namespace")
	}
	return newns

}
