// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one

// or more contributor license agreements. Licensed under the Elastic License;

// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"errors"
	"reflect"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func TestHook_Name(t *testing.T) {
	tests := []struct {
		name string
		h    *Hook
		want string
	}{
		{
			name: "TestCase1",
			h:    NewHook("TestCase1", func(socket *string, log *logger.Logger) error { return nil }),
			want: "TestCase1",
		},
		{
			name: "TestCase2",
			h:    NewHook("TestCase2", func(socket *string, log *logger.Logger) error { return nil }),
			want: "TestCase2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.Name(); got != tt.want {
				t.Errorf("Hook.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHook_Execute(t *testing.T) {
	socket := ""
	log := logger.New(os.Stderr, false)
	type args struct {
		socket *string
		log    *logger.Logger
	}
	tests := []struct {
		name    string
		h       *Hook
		args    args
		wantErr bool
	}{
		{
			name: "TestCase1",
			h:    NewHook("TestCase1", func(socket *string, log *logger.Logger) error { return errors.New("error") }),
			args: args{
				socket: &socket,
				log:    log,
			},
			wantErr: true,
		},
		{
			name: "TestCase2",
			h:    NewHook("TestCase2", func(socket *string, log *logger.Logger) error { return nil }),
			args: args{
				socket: &socket,
				log:    log,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.Execute(tt.args.socket, tt.args.log); (err != nil) != tt.wantErr {
				t.Errorf("Hook.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewHook(t *testing.T) {
	type args struct {
		hookName string
		hookFunc HookFunc
	}
	tests := []struct {
		name string
		args args
		want *Hook
	}{
		{
			name: "TestCase1",
			args: args{
				hookName: "TestCase1",
				hookFunc: func(socket *string, log *logger.Logger) error { return nil },
			},
			want: NewHook("TestCase1", func(socket *string, log *logger.Logger) error { return nil }),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHook(tt.args.hookName, tt.args.hookFunc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHookManager(t *testing.T) {

	socket := ""
	log := logger.New(os.Stderr, false)

	hookFunc1Completed := false
	hookFunc2Completed := false
	hookFunc1 := func(socket *string, log *logger.Logger) error { hookFunc1Completed = true; return nil }
	hookFunc2 := func(socket *string, log *logger.Logger) error { hookFunc2Completed = true; return nil }

	hm := NewHookManager(HookTypePost)
	hm.Register(NewHook("TestCase1", hookFunc1))
	hm.Register(NewHook("TestCase2", hookFunc2))

	hm.Execute(&socket, log)

	if !hookFunc1Completed {
		t.Errorf("hookFunc1Completed = %v, want %v", hookFunc1Completed, true)
	}
	if !hookFunc2Completed {
		t.Errorf("hookFunc2Completed = %v, want %v", hookFunc2Completed, true)
	}
}
