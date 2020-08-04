// Copyright 2020 Douyu
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

package jupiter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/douyu/jupiter/pkg/server/xgrpc"

	"github.com/douyu/jupiter/pkg/server"
	. "github.com/smartystreets/goconvey/convey"
)

type testServer struct {
	ServeBlockTime time.Duration
	ServeErr       error

	StopBlockTime time.Duration
	StopErr       error

	GstopBlockTime time.Duration
	GstopErr       error
}

func (s *testServer) Serve() error {
	time.Sleep(s.ServeBlockTime)
	return s.ServeErr
}
func (s *testServer) Stop() error {
	time.Sleep(s.StopBlockTime)
	return s.StopErr
}
func (s *testServer) GracefulStop(ctx context.Context) error {
	time.Sleep(s.GstopBlockTime)
	return s.GstopErr
}
func (s *testServer) Info() *server.ServiceInfo {
	return &server.ServiceInfo{}
}
func TestApplication_New(t *testing.T) {
	Convey("test application run serve", t, func(c C) {
		_, err := New()
		So(err, ShouldBeNil)
	})
}
func TestApplication_Run_1(t *testing.T) {
	Convey("test application run serve", t, func(c C) {
		srv := &testServer{
			ServeErr: errors.New("when server call serve error"),
		}
		app := &Application{}
		app.initialize()
		err := app.Serve(srv)
		So(err, ShouldBeNil)
		go func() {
			// make sure Serve() is called
			time.Sleep(time.Millisecond * 100)
			err := app.Stop()
			c.So(err, ShouldBeNil)
		}()
		err = app.Run()
		So(err, ShouldEqual, srv.ServeErr)
	})
	Convey("test application run serve block", t, func(c C) {
		srv := &testServer{
			ServeBlockTime: time.Second,
			ServeErr:       errors.New("when server call serve error"),
		}
		app := &Application{}
		app.initialize()
		err := app.Serve(srv)
		So(err, ShouldBeNil)
		go func() {
			// make sure Serve() is called
			time.Sleep(time.Millisecond * 100)
			err := app.Stop()
			c.So(err, ShouldBeNil)
		}()
		err = app.Run()
		So(err, ShouldEqual, srv.ServeErr)
	})
	Convey("test application run stop", t, func(c C) {
		srv := &testServer{
			ServeBlockTime: time.Second * 2,
			StopBlockTime:  time.Second,
			StopErr:        errors.New("when server call stop error"),
		}
		app := &Application{}
		app.initialize()
		err := app.Serve(srv)
		So(err, ShouldBeNil)
		go func() {
			// make sure Serve() is called
			time.Sleep(time.Millisecond * 200)
			err := app.Stop()
			c.So(err, ShouldBeNil)
		}()
		err = app.Run()
		So(err, ShouldEqual, srv.StopErr)
	})
}

func TestApplication_initialize(t *testing.T) {
	Convey("test application initialize", t, func() {
		app := &Application{}
		app.initialize()
		So(app.servers, ShouldNotBeNil)
		So(app.workers, ShouldNotBeNil)
		So(app.logger, ShouldNotBeNil)
		So(app.cycle, ShouldNotBeNil)
	})
}

func TestApplication_Startup(t *testing.T) {
	Convey("test application startup error", t, func() {
		app := &Application{}
		startUpErr := errors.New("throw startup error")
		err := app.Startup(func() error {
			return startUpErr
		})
		So(err, ShouldEqual, startUpErr)
	})

	Convey("test application startup nil", t, func() {
		app := &Application{}
		err := app.Startup(func() error {
			return nil
		})
		So(err, ShouldBeNil)
	})
}

type stopInfo struct {
	state bool
}

func (info *stopInfo) Stop() error {
	info.state = true
	return nil
}

func TestApplication_BeforeStop(t *testing.T) {
	Convey("test application before stop", t, func(c C) {
		si := &stopInfo{}
		app := &Application{}
		app.initialize()
		app.RegisterHooks(StageBeforeStop, si.Stop)
		go func(si *stopInfo) {
			time.Sleep(time.Microsecond * 100)
			err := app.Stop()
			c.So(err, ShouldBeNil)
			c.So(si.state, ShouldEqual, true)
		}(si)
		err := app.Run()
		c.So(err, ShouldBeNil)
	})
}
func TestApplication_EmptyRun(t *testing.T) {
	Convey("test application empty run", t, func(c C) {
		app := &Application{}
		app.initialize()
		go func() {
			app.cycle.DoneAndClose()
		}()
		err := app.Run()
		c.So(err, ShouldBeNil)
	})
}

func TestApplication_AfterStop(t *testing.T) {
	Convey("test application after stop", t, func() {
		si := &stopInfo{}
		app := &Application{}
		app.initialize()
		app.RegisterHooks(StageAfterStop, si.Stop)
		go func() {
			app.Stop()
		}()
		err := app.Run()
		So(err, ShouldBeNil)
		So(si.state, ShouldEqual, true)
	})
}

func TestApplication_Serve(t *testing.T) {
	Convey("test application serve throw wrong ip", t, func(c C) {
		app := &Application{}
		grpcConfig := xgrpc.DefaultConfig()
		grpcConfig.Port = 0
		app.initialize()
		err := app.Serve(grpcConfig.Build())
		So(err, ShouldBeNil)
		go func() {
			// make sure Serve() is called
			time.Sleep(time.Millisecond * 1500)
			err = app.Stop()
			c.So(err, ShouldBeNil)
		}()
		err = app.Run()
		// So(err, ShouldEqual, grpc.ErrServerStopped)
		So(err, ShouldBeNil)
	})
}

type testWorker struct {
	RunErr  error
	StopErr error
}

func (t *testWorker) Run() error {
	return t.RunErr
}
func (t *testWorker) Stop() error {
	return t.StopErr
}
func Test_Unit_Application_Schedule(t *testing.T) {
	Convey("test unit Application.Schedule", t, func(c C) {
		w := &testWorker{}
		app := &Application{}
		err := app.Schedule(w)
		c.So(err, ShouldBeNil)
	})
}
func Test_Unit_Application_Stop(t *testing.T) {
	Convey("test unit Application.Stop", t, func(c C) {
		app := &Application{}
		app.initialize()
		err := app.Stop()
		c.So(err, ShouldBeNil)
	})
}

func Test_Unit_Application_GracefulStop(t *testing.T) {
	Convey("test unit Application.GracefulStop", t, func(c C) {
		app := &Application{}
		app.initialize()
		err := app.GracefulStop(context.TODO())
		c.So(err, ShouldBeNil)
	})
}
func Test_Unit_Application_startServers(t *testing.T) {
	Convey("test unit Application.startServers", t, func(c C) {
		app := &Application{}
		app.initialize()
		err := app.startServers()
		c.So(err, ShouldBeNil)
		go func() {
			time.Sleep(time.Microsecond * 100)
			app.Stop()
		}()
	})
}

type testJobRunner struct{}

func (t *testJobRunner) Run() {}

func Test_Unit_Application_Job(t *testing.T) {
	j := &testJobRunner{}
	app := &Application{}
	app.initialize()
	app.Job(j)
}