// Copyright 2018-2020 Burak Sezer
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transport

import (
	"context"
	"github.com/buraksezer/olric/internal/flog"
	"github.com/buraksezer/olric/internal/protocol"
	"log"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
)

func dispatcher(w, _ protocol.EncodeDecoder) {
	w.SetStatus(protocol.StatusOK)
}

func getRandomAddr() (string, int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return "", 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", 0, err
	}
	defer l.Close()
	// Now, parse the obtained address
	host, rawPort, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

func newServer() (*Server, error){
	bindAddr, bindPort, err := getRandomAddr()
	if err != nil {
		return nil, err
	}
	l := log.New(os.Stdout, "transport-test", log.LstdFlags)
	fl := flog.New(l)
	s := NewServer(bindAddr, bindPort, time.Second, fl)
	s.SetDispatcher(dispatcher)
	return s, nil
}

func TestServer_ListenAndServe(t *testing.T) {
	s , err := newServer()
	if err != nil {
		t.Fatalf("Expected nil. Got: %v", err)
	}
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			t.Fatalf("Expected nil. Got: %v", err)
		}
	}()
	defer func() {
		err = s.Shutdown(context.TODO())
		if err != nil {
			t.Fatalf("Expected nil. Got: %v", err)
		}
	}()
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("StartCh could not be closed")
	case <-s.StartCh:
		return
	}
}

func TestServer_ProcessConn(t *testing.T) {
	s , err := newServer()
	if err != nil {
		t.Fatalf("Expected nil. Got: %v", err)
	}
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			t.Fatalf("Expected nil. Got: %v", err)
		}
	}()
	defer func() {
		err = s.Shutdown(context.TODO())
		if err != nil {
			t.Fatalf("Expected nil. Got: %v", err)
		}
	}()

	<-s.StartCh
	cc := &ClientConfig{
		Addrs:   []string{s.listener.Addr().String()},
		MaxConn: 10,
	}
	c := NewClient(cc)

	t.Run("process DMapMessage", func(t *testing.T) {
		req := protocol.NewDMapMessage(protocol.OpPut)
		resp, err := c.Request(req)
		if err != nil {
			t.Fatalf("Expected nil. Got: %v", err)
		}
		if resp.Status() != protocol.StatusOK {
			t.Fatalf("Expected status: %v. Got: %v", protocol.StatusOK, resp.Status())
		}
	})

	t.Run("process StreamMessage", func(t *testing.T) {
		req := protocol.NewStreamMessage(protocol.OpCreateStream)
		resp, err := c.Request(req)
		if err != nil {
			t.Fatalf("Expected nil. Got: %v", err)
		}
		if resp.Status() != protocol.StatusOK {
			t.Fatalf("Expected status: %v. Got: %v", protocol.StatusOK, resp.Status())
		}
	})
}