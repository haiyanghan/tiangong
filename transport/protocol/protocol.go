package protocol

type Protocol = byte

const (
	Unknown Protocol = iota
	TCP
	UDP
	ICMP
	HTTP
	HTTPS
	WS
)

var ()

func ProtocolToStr(p Protocol) string {
	switch p {
	case TCP:
		return "TCP"
	case UDP:
		return "UDP"
	case ICMP:
		return "ICMP"
	default:
		return "Unknown"
	}
}
