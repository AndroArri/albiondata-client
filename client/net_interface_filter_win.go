// +build windows

package client

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	IfOperStatusUp            = 1
	IF_TYPE_SOFTWARE_LOOPBACK = 24
)

const hexDigit = "0123456789abcdef"

func adapterAddresses() ([]*windows.IpAdapterAddresses, error) {
	var b []byte
	l := uint32(15000) // recommended initial size
	for {
		b = make([]byte, l)
		err := windows.GetAdaptersAddresses(syscall.AF_UNSPEC, windows.GAA_FLAG_INCLUDE_PREFIX, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])), &l)
		if err == nil {
			if l == 0 {
				return nil, nil
			}
			break
		}
		if err.(syscall.Errno) != syscall.ERROR_BUFFER_OVERFLOW {
			return nil, os.NewSyscallError("getadaptersaddresses", err)
		}
		if l <= uint32(len(b)) {
			return nil, os.NewSyscallError("getadaptersaddresses", err)
		}
	}
	var aas []*windows.IpAdapterAddresses
	for aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])); aa != nil; aa = aa.Next {
		aas = append(aas, aa)
	}
	return aas, nil
}

func bytePtrToString(p *uint8) string {
	a := (*[10000]uint8)(unsafe.Pointer(p))
	i := 0
	for a[i] != 0 {
		i++
	}
	return string(a[:i])
}

func physicalAddrToString(physAddr [8]byte) string {
	if len(physAddr) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(physAddr)*3-1)
	for i, b := range physAddr {
		if i > 0 {
			buf = append(buf, ':')
		}
		buf = append(buf, hexDigit[b>>4])
		buf = append(buf, hexDigit[b&0xF])
	}
	return string(buf)
}

// Gets all physical interfaces based on filter results, ignoring all VM, Loopback and Tunnel interfaces.
func getAllPhysicalInterface() []string {
	aa, _ := adapterAddresses()

	var outInterfaces []string

	for _, pa := range aa {
		mac := physicalAddrToString(pa.PhysicalAddress)
		name := "\\Device\\NPF_" + bytePtrToString(pa.AdapterName)
		var flags uint32 = pa.Flags

		if flags&uint32(IF_TYPE_SOFTWARE_LOOPBACK) == 0 && flags&uint32(IfOperStatusUp) == 1 && isPhysicalInterface(mac) {
			outInterfaces = append(outInterfaces, name)
		}
	}

	return outInterfaces
}
