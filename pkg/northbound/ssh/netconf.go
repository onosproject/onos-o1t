// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package ssh

import (
	"context"
	"io"
	"time"
)

var (
	netconfTimeout = 1
)

type NetconfServer interface {
	Serve() error
}

type netconfSubsystem struct {
	ctx Context
	srv SSHServer
	*serverConn
}

func Hello(n *netconfSubsystem) error {
	helloRequest := "<request-hello"

	netconfCtx, cancel := context.WithTimeout(context.Background(), time.Duration(netconfTimeout)*time.Second)
	defer cancel()

	hello, err := n.srv.Handle(netconfCtx, []byte(helloRequest))
	if err != nil {
		log.Errorf("error create hello request: %v", err)
		return err
	}

	err = n.serverConn.send(hello)
	if err != nil {
		log.Errorf("error send hello request: %v", err)
		return err
	}

	return nil
}

func (n *netconfSubsystem) Serve() error {

	log.Info("starting netconf subsystem")

	err := Hello(n)
	if err != nil {
		log.Errorf("error netconf hello: %v", err)
		return err
	}

	for {

		data, err := n.serverConn.receive()

		if err != nil {
			log.Debugf("handler read error: %s", err)
			return err
		}

		netconfCtx, cancel := context.WithTimeout(context.Background(), time.Duration(netconfTimeout)*time.Second)
		defer cancel()

		reply, err := n.srv.Handle(netconfCtx, data)
		if err != nil {
			log.Infof("Serve decode error: %s", err)
			return err
		}

		if reply != nil {
			err = n.serverConn.send(reply)
			if err != nil {
				log.Debugf("handler write error: %s", err)
				break
			}
		}

	}

	log.Info("finishing netconf subsystem")
	return nil
}

func NewNetconfServer(ctx Context, srv SSHServer, rwc io.ReadWriteCloser) NetconfServer {
	svrConn := &serverConn{
		conn: conn{
			Reader:      rwc,
			WriteCloser: rwc,
		},
	}

	ns := &netconfSubsystem{
		ctx:        ctx,
		srv:        srv,
		serverConn: svrConn,
	}
	return ns
}
