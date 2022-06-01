// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package o1tclient

import (
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/openshift-telco/go-netconf-client/netconf/message"

	"golang.org/x/crypto/ssh"
)

type Controller interface {
	GetConfig() (string, error)
	EditConfig() error
	CloseSession() error
	EndSession() error
}

type controller struct {
	onosO1TAddress string
	session        *netconf.Session
}

func NewController(onosO1TAddress string) (Controller, error) {
	log.Info("Init")

	session, err := createSession(onosO1TAddress)
	if err != nil {
		return nil, err
	}

	log.Info("Starting Controller")
	return &controller{
		onosO1TAddress: onosO1TAddress,
		session:        session,
	}, nil
}

func generateSigner() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func DialSSH(target string, config *ssh.ClientConfig) (*netconf.Session, error) {
	var t netconf.TransportSSH
	t.SetVersion("v1.1")
	err := t.Dial(target, config)
	if err != nil {
		err := t.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}
	return netconf.NewSession(&t), nil
}

func createSession(onosO1TAddress string) (*netconf.Session, error) {
	log.Info("Creating Session")

	signer, err := generateSigner()
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		// User:            "admin",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	log.Info("Dialing SSH")
	s, err := DialSSH(onosO1TAddress, sshConfig)
	if err != nil {
		log.Info("Netconf dial error", err)

		return nil, err
	}

	log.Info("Netconf Session Hello")
	capabilities := netconf.DefaultCapabilities
	err = s.SendHello(&message.Hello{Capabilities: capabilities})
	if err != nil {
		return nil, err
	}

	s.Transport.SetVersion("v1.1")
	return s, nil
}

func (c *controller) GetConfig() (string, error) {
	log.Info("Get Config")

	g := new(GetConfig)
	g.Source = &Datastore{Running: ""}
	g.Filter = &Filter{
		Type:   "xpath",
		Data:   "<root />",
		Select: "/",
		XMLNS:  "http://opennetworking.org/kpimon:ric:1.0.0",
	}
	g.MessageID = "1"
	err := c.session.AsyncRPC(g, defaultLogRpcReplyCallback(g.MessageID))
	time.Sleep(200 * time.Millisecond)
	return "", err
}

func (c *controller) EditConfig() error {
	log.Info("Edit Config")
	data := `<report_period xmlns="http://opennetworking.org/kpimon:ric:1.0.0">
				<interval>5000</interval>
			 </report_period>`

	e := new(EditConfig)
	e.Target = &Datastore{Running: ""}
	e.DefaultOperation = "merge"
	e.Config = &config{Config: data}
	e.MessageID = "2"

	err := c.session.AsyncRPC(e, defaultLogRpcReplyCallback(e.MessageID))
	time.Sleep(200 * time.Millisecond)
	return err
}

func (c *controller) CloseSession() error {
	log.Info("Close Session")
	e := new(CloseSession)
	e.MessageID = NewUUID()
	e.CloseSession = ""

	err := c.session.AsyncRPC(e, defaultLogRpcReplyCallback(e.MessageID))
	time.Sleep(200 * time.Millisecond)
	return err
}

func (c *controller) EndSession() error {
	log.Info("End")
	err := c.session.Close()
	return err
}

func defaultLogRpcReplyCallback(eventId string) netconf.Callback {
	return func(event netconf.Event) {
		reply := event.RPCReply()
		if reply == nil {
			log.Info("Failed to execute RPC")
		}
		if event.EventID() == eventId {
			log.Info("Successfully executed RPC")
			log.Info(reply.RawReply)
		}
	}
}
