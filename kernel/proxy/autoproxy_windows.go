package proxy

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/haiyanghan/tiangong/common"
	"github.com/haiyanghan/tiangong/common/errors"
)

const (

	//  Options used in INTERNET_PER_CONN_OPTION struct
	INTERNET_PER_CONN_FLAGS          = 1 // Sets or retrieves the connection type. The Value member will contain one or more of the values from PerConnFlags
	INTERNET_PER_CONN_PROXY_SERVER   = 2 // Sets or retrieves a string containing the proxy servers.
	INTERNET_PER_CONN_PROXY_BYPASS   = 3 // Sets or retrieves a string containing the URLs that do not use the proxy server.
	INTERNET_PER_CONN_AUTOCONFIG_URL = 4 // Sets or retrieves a string containing the URL to the automatic configuration script.

	INTERNET_OPTION_REFRESH               = 37
	INTERNET_OPTION_PROXY                 = 38
	INTERNET_OPTION_SETTINGS_CHANGED      = 39
	INTERNET_OPTION_PER_CONNECTION_OPTION = 75

	// INTERNET_PER_CONN_FLAGS values:
	PROXY_TYPE_DIRECT         = 0x00000001 // direct to net
	PROXY_TYPE_PROXY          = 0x00000002 // via named proxy
	PROXY_TYPE_AUTO_PROXY_URL = 0x00000004 // autoproxy URL
	PROXY_TYPE_AUTO_DETECT    = 0x00000008 // use autoproxy detection

)

var (
	callWindowsFunc = func(proc, dwOption, lpBuffer, dwBufferLength uintptr) error {
		r1, _, err := syscall.Syscall6(proc, 4, 0, dwOption, lpBuffer, dwBufferLength, 0, 0)
		if r1 != 1 {
			return err
		}
		return nil
	}
)

type INTERNET_PER_CONN_OPTION struct {
	dwOption uint32
	dwValue  uint64 // 注意 32位 和 64位 struct 和 union 内存对齐ffffff
}

type INTERNET_PER_CONN_OPTION_LIST struct {
	dwSize        uint32
	pszConnection *uint16
	dwOptionCount uint32
	dwOptionError uint32
	pOptions      uintptr
}

func SetProxy(proxy string, ignores []string) error {
	winInet, err := syscall.LoadLibrary("Wininet.dll")
	defer syscall.Close(winInet)

	if err != nil {
		return errors.NewError("loadLibrary Wininet.dll error, ", err)
	}
	InternetSetOption, err := syscall.GetProcAddress(winInet, "InternetSetOption")
	if err != nil {
		return errors.NewError("getProcAddress InternetSetOptionW.dll error, ", err)
	}

	options := buildOptions(proxy, ignores)

	list := INTERNET_PER_CONN_OPTION_LIST{}
	list.dwSize = uint32(unsafe.Sizeof(list))
	list.pszConnection = nil
	list.dwOptionCount = uint32(len(options))
	list.dwOptionError = 0
	list.pOptions = uintptr(unsafe.Pointer(&options[0]))

	err = callWindowsFunc(InternetSetOption, INTERNET_OPTION_PER_CONNECTION_OPTION, uintptr(unsafe.Pointer(&list)), uintptr(unsafe.Sizeof(list)))
	if err != nil {
		return errors.NewError("call InternetSetOption ErrorCode: %s", err)
	}
	_ = callWindowsFunc(InternetSetOption, INTERNET_OPTION_SETTINGS_CHANGED, 0, 0)
	_ = callWindowsFunc(InternetSetOption, INTERNET_OPTION_REFRESH, 0, 0)
	return nil
}

func QuerySystemProxy() error {
	winInet, err := syscall.LoadLibrary("Wininet.dll")
	defer syscall.Close(winInet)

	if err != nil {
		return errors.NewError("loadLibrary Wininet.dll error, ", err)
	}
	InternetQueryOption, err := syscall.GetProcAddress(winInet, "InternetQueryOption")
	if err != nil {
		return errors.NewError("getProcAddress InternetSetOptionW.dll error, ", err)
	}
	length := 4 * 1024
	buffer := make([]byte, length)
	if callWindowsFunc(InternetQueryOption, INTERNET_OPTION_PROXY, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&length))); err != nil {
		return err
	}
	return nil
}

func buildOptions(proxy string, ignores []string) []INTERNET_PER_CONN_OPTION {
	if common.IsEmpty(proxy) {
		// Reset
		return []INTERNET_PER_CONN_OPTION{
			{INTERNET_PER_CONN_FLAGS, PROXY_TYPE_DIRECT},
		}
	} else {
		options := make([]INTERNET_PER_CONN_OPTION, 3)
		options[0].dwOption = INTERNET_PER_CONN_FLAGS
		options[0].dwValue = PROXY_TYPE_PROXY
		options[1].dwOption = INTERNET_PER_CONN_PROXY_SERVER
		options[1].dwValue = uint64(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(proxy))))
		if len(ignores) > 0 {
			options[2].dwOption = INTERNET_PER_CONN_PROXY_BYPASS
			options[2].dwValue = uint64(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(strings.Join(ignores, ";")))))
		}
		return options
	}
}

func ResetProxy() error {
	return SetProxy("", []string{})
}
