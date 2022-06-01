// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package ssh

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-o1t/pkg/controller"
	"golang.org/x/crypto/ssh"
)

var log = logging.GetLogger()

const (
	// sshNetconfSubsystem sets the SSH subsystem to NETCONF
	sshNetconfSubsystem = "netconf"
)

var (
	ContextKeyUser = &contextKey{"user"}

	ContextKeySessionID = &contextKey{"session-id"}

	ContextKeyPublicKey = &contextKey{"public-key"}

	ContextKeyServer = &contextKey{"ssh-server"}

	ContextKeyPermissions = &contextKey{"permissions"}
)

type contextKey struct {
	name string
}

type Context interface {
	context.Context
	sync.Locker

	// User returns the username used when establishing the SSH connection.
	User() string

	// SessionID returns the session hash.
	SessionID() string

	// Permissions returns the Permissions object used for this connection.
	Permissions() *ssh.Permissions

	// SetValue allows you to easily write new values into the underlying context.
	SetValue(key, value interface{})
}

type sshContext struct {
	context.Context
	*sync.Mutex
}

func (ctx *sshContext) User() string {
	return ctx.Value(ContextKeyUser).(string)
}

func (ctx *sshContext) SessionID() string {
	return ctx.Value(ContextKeySessionID).(string)
}

func (ctx *sshContext) Permissions() *ssh.Permissions {
	return ctx.Value(ContextKeyPermissions).(*ssh.Permissions)
}

func (ctx *sshContext) SetValue(key, value interface{}) {
	ctx.Context = context.WithValue(ctx.Context, key, value)
}

func fillContext(ctx Context, conn ssh.ConnMetadata) {
	if ctx.Value(ContextKeySessionID) != nil {
		return
	}
	ctx.SetValue(ContextKeySessionID, hex.EncodeToString(conn.SessionID()))
	ctx.SetValue(ContextKeyUser, conn.User())
}

func newContext(srv *sshServer) (*sshContext, context.CancelFunc) {
	innerCtx, cancel := context.WithCancel(context.Background())
	ctx := &sshContext{innerCtx, &sync.Mutex{}}
	ctx.SetValue(ContextKeyServer, srv)
	ctx.SetValue(ContextKeyPermissions, &ssh.Permissions{})
	return ctx, cancel
}

type PublicKeyHandler func(ctx Context, key ssh.PublicKey) bool
type PasswordHandler func(ctx Context, password string) bool
type SubsystemHandler func(ctx Context, srv *sshServer, sshCh ssh.Channel) error

func NetconfHandler(ctx Context, srv *sshServer, sshCh ssh.Channel) error {
	ns := NewNetconfServer(ctx, srv, sshCh)
	err := ns.Serve()
	return err
}

var DefaultSubsystemHandlers = map[string]SubsystemHandler{
	sshNetconfSubsystem: NetconfHandler,
}

type SSHServer interface {
	Start() error
	Stop(ctx context.Context) error
	Handle(context.Context, string, []byte) ([]byte, error)
}

type sshServer struct {
	mu sync.RWMutex

	netconfPort int
	Version     string

	subsystemHandlers map[string]SubsystemHandler
	HostSigners       []ssh.Signer

	PasswordHandler  PasswordHandler
	PublicKeyHandler PublicKeyHandler

	controller controller.O1Controller
}

func (srv *sshServer) addHostKey(key ssh.Signer) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	for i, k := range srv.HostSigners {
		if k.PublicKey().Type() == key.PublicKey().Type() {
			srv.HostSigners[i] = key
			return
		}
	}

	srv.HostSigners = append(srv.HostSigners, key)
}

func (srv *sshServer) createHostKeyFile() error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return err
	}

	srv.addHostKey(signer)

	return nil
}

func NewSSHServer(netconfPort int, o1tControl controller.O1Controller) (SSHServer, error) {
	srv := &sshServer{
		netconfPort: netconfPort,
	}
	srv.subsystemHandlers = DefaultSubsystemHandlers
	srv.controller = o1tControl

	srv.PublicKeyHandler = func(ctx Context, key ssh.PublicKey) bool {
		return true //TODO For now allow all keys, later use ssh.KeysEqual() to compare against known keys
	}

	err := srv.createHostKeyFile()
	if err != nil {
		return nil, err
	}

	return srv, nil
}

func (srv *sshServer) Handle(ctx context.Context, sessionID string, request []byte) ([]byte, error) {
	reply, err := srv.controller.Handler(ctx, sessionID, request)
	return reply, err
}

func (srv *sshServer) Stop(ctx context.Context) error {
	return nil
}

func (srv *sshServer) config(ctx Context) *ssh.ServerConfig {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	config := &ssh.ServerConfig{}

	for _, signer := range srv.HostSigners {
		config.AddHostKey(signer)
	}

	if srv.PasswordHandler == nil && srv.PublicKeyHandler == nil {
		config.NoClientAuth = true
	}
	if srv.Version != "" {
		config.ServerVersion = "SSH-2.0-" + srv.Version
	}
	if srv.PasswordHandler != nil {
		config.PasswordCallback = func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			fillContext(ctx, conn)
			if ok := srv.PasswordHandler(ctx, string(password)); !ok {
				return ctx.Permissions(), fmt.Errorf("permission denied")
			}
			return ctx.Permissions(), nil
		}
	}
	if srv.PublicKeyHandler != nil {
		config.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			fillContext(ctx, conn)
			if ok := srv.PublicKeyHandler(ctx, key); !ok {
				return ctx.Permissions(), fmt.Errorf("permission denied")
			}
			ctx.SetValue(ContextKeyPublicKey, key)
			return ctx.Permissions(), nil
		}
	}
	return config
}

func (srv *sshServer) Start() error {
	address := "0.0.0.0:" + strconv.Itoa(srv.netconfPort)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Error(err)
		return err

	}

	log.Infof("Netconf SSH server listening on %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error(err)
			return err
		}

		ctx, _ := newContext(srv)
		config := srv.config(ctx)

		srvConn, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
			log.Error(err)
			return err
		}

		fillContext(ctx, srvConn)
		go ssh.DiscardRequests(reqs)
		go srv.handleServerConn(ctx, chans)
	}
}

func (srv *sshServer) handleServerConn(ctx Context, chans <-chan ssh.NewChannel) {
	log.Info("Handling Connection")

	for newChan := range chans {
		log.Infof("Handling Channel %s", newChan.ChannelType())

		if newChan.ChannelType() != "session" {
			err := newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			if err != nil {
				log.Warn(err)
			}
			continue
		}

		ch, reqs, err := newChan.Accept()
		if err != nil {
			log.Infof("Handling Channel error %s", err)
			continue
		}

		go func(sshCh ssh.Channel, in <-chan *ssh.Request) {
			defer sshCh.Close()

			for req := range in {
				log.Infof("Handling Request %s", req.Type)

				switch req.Type {

				case "subsystem":
					var payload = struct{ Value string }{}
					err = ssh.Unmarshal(req.Payload, &payload)
					if err != nil {
						log.Warn(err)
						return
					}

					handler, ok := srv.subsystemHandlers[payload.Value]
					if !ok {
						err = req.Reply(false, nil)
						if err != nil {
							log.Warn(err)
							return
						}
						continue
					}

					log.Infof("Handling subsystem %s", payload.Value)
					go func() {
						defer sshCh.Close()
						err := handler(ctx, srv, sshCh)
						if err != nil {
							log.Warn(err)
							return
						}
					}()

					err = req.Reply(true, nil)
					if err != nil {
						log.Warn(err)
						return
					}

				default:
					err = req.Reply(false, nil)
					if err != nil {
						log.Warn(err)
						return
					}

					continue
				}
			}
		}(ch, reqs)
	}
}
