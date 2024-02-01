package main

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/haiyanghan/tiangong"
	"github.com/haiyanghan/tiangong/common"
	"github.com/haiyanghan/tiangong/common/buf"
	"github.com/haiyanghan/tiangong/common/conf"
	"github.com/haiyanghan/tiangong/common/errors"
	"github.com/haiyanghan/tiangong/common/log"
	"github.com/haiyanghan/tiangong/common/net"
	"github.com/haiyanghan/tiangong/kernel/proxy"
	"github.com/haiyanghan/tiangong/transport/protocol"
)

var (
	ConnTimeout      = 30 * time.Second
	HandshakeTimeout = ConnTimeout
	Config           = config{}
)

type Processor struct {
	ctx         context.Context
	client      net.TcpClient
	proxyServer net.TcpServer
	incrementer common.Incrementer
	outConn     net.Conn
}

func NewProcessor(cp string) Processor {
	conf.LoadConfig(cp, &Config, conf.EmptyDefaultValueFunc)

	p := Processor{
		incrementer: common.Incrementer{Range: common.Range{0, math.MaxUint32}},
	}

	ctx := common.SetProcess(context.Background(), &p)
	p.ctx = ctx
	p.client = net.NewTcpClient(Config.ServerHost, Config.ServerPort, ctx)
	p.proxyServer = net.NewTcpServer(Config.ProxyHost, Config.ProxyPort, ctx)
	return p
}

func GetProxyAddr() string {
	return fmt.Sprintf("%s:%d", Config.ProxyHost, Config.ProxyPort)
}

func (p *Processor) Start() error {
	p.proxyServer.ListenTCP(StartListener)
	if err := p.client.Connect(ConnSuccess); err != nil {
		log.Warn("Connect targte server error retry...", nil)
		go common.Retry(func() error {
			return p.client.Connect(ConnSuccess)
		}).Run(3*time.Second, -1)
		return nil
	}

	return nil
}

func (p *Processor) WriteToRemote(proto protocol.Protocol, bytes buf.Buffer) error {
	l := bytes.Len()
	if l > math.MaxUint16 {
		return errors.NewError("packet length exceeding maximum limit", nil)
	}
	h := protocol.PacketHeader{
		Len:      uint16(l),
		Rid:      uint32(p.incrementer.Next()),
		Protocol: byte(proto),
	}

	buffer := buf.NewBuffer(protocol.PacketHeaderLen + l)
	defer buffer.Release()

	// write packet header
	_ = h.WriteTo(buffer)
	// write body
	_, _ = buffer.Write(bytes, l)
	return p.client.Write(buffer)
}

func ConnSuccess(ctx context.Context, conn net.Conn) error {
	closeFunc := conn.Close
	// handshake
	if err := handshake(conn, Config.Token, net.ParseIp(Config.Subhost)); err != nil {
		_ = closeFunc()
		return err
	}
	log.Info("Connect target server success [%s]", conn.RemoteAddr())

	p := common.GetProcess(ctx).(*Processor)
	p.outConn = conn

	buffer := buf.NewRingBuffer()
	defer buffer.Release()

	if err := proxy.SetProxy(GetProxyAddr(), []string{"192.168.*.*"}); err != nil {
		log.Warn("Set systemproxy error, %s", err.Error())
		return nil
	}

	return nil
}

func StartListener(ctx context.Context, conn net.Conn) error {
	p := ctx.Value(common.ProcessKey).(*Processor)

	go func() {
		buffer := buf.NewRingBuffer()
		defer buffer.Release()
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				runtime.Goexit()
			default:
				if n, err := buffer.Write(conn, buffer.Cap()); err != nil || n <= 0 {
					log.Warn("Connect closed, %s", err.Error())
					return
				}
				bytes, _ := buf.ReadAll(buffer)
				if err := p.WriteToRemote(protocol.TCP, buf.Wrap(bytes)); err != nil {
					log.Error("Write to remote server error", err)
				}
			}
		}
	}()

	return nil
}

func handshake(conn net.Conn, token string, subHost net.IpAddress) error {
	timeout := time.Now().Add(HandshakeTimeout)
	buffer := buf.NewBuffer(256)
	ctx, cancel := context.WithTimeout(context.Background(), HandshakeTimeout)
	defer cancel()
	defer buffer.Release()

	{
		authBody := protocol.SessionAuth{
			Token:   token,
			SubHost: subHost[:],
		}
		header := protocol.NewAuthHeader(tiangong.VersionByte(), protocol.AuthSession)
		header.AppendBody(&authBody)
		if err := header.WriteTo(buffer); err != nil {
			return err
		}
		if err := conn.SetWriteDeadline(timeout); err != nil {
			return errors.NewError("SetWriteDeadline error", err)
		}

		if err := conn.ReadFrom(buffer); err != nil {
			return err
		}
		_ = buffer.Clear()
	}
	select {
	case <-ctx.Done():
		return errors.NewError("Handshake Timeout", nil)
	default:
		if err := conn.SetReadDeadline(timeout); err != nil {
			return errors.NewError("SetReadDeadline error", err)
		}
		if n, err := buffer.Write(conn, protocol.AuthResponseLen); err != nil {
			return errors.NewError("", err)
		} else if n < protocol.AuthResponseLen {
			return errors.NewError(fmt.Sprintf("Auth response body too short, require %d bytes, Actual return %d bytes", protocol.AuthResponseLen, n), err)
		}
		response := protocol.AuthResponse{}
		if err := response.ReadFrom(buffer); err != nil || response.Status != protocol.AuthSuccess {
			return errors.NewError("handshake fail", err)
		}
	}

	return nil
}
