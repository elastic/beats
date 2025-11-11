// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package parsers

import (
	"reflect"
	"testing"
)

func TestLookupApplicationId(t *testing.T) {
	type args struct {
		appId string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Firefox",
			args: args{appId: "3476342aab319002"},
			want: "Mozilla Firefox",
		},
		{
			name: "Notepad++",
			args: args{appId: "ea5af8ce5aeb5617"},
			want: "Notepad++",
		},
		{
			name: "Visual Studio",
			args: args{appId: "acb8cd11364e2de8"},
			want: "VisualStudio",
		},
		{
			name: "Microsoft Edge",
			args: args{appId: "ccba5a5986c77e43"},
			want: "Microsoft Edge (Chromium)",
		},
		{
			name: "Slack",
			args: args{appId: "8bce06a9e923e1f9"},
			want: "Slack 4.10.3",
		},
		{
			name: "WinRAR",
			args: args{appId: "ad57bd0f4825cce"},
			want: "WinRAR 6.01 Russian 64 bit",
		},
		{
			name: "PowerShell 7",
			args: args{appId: "3c3871276e149215"},
			want: "PowerShell 7",
		},
		{
			name: "Microsoft Teams",
			args: args{appId: "a55ed4fbb973aefb"},
			want: "Microsoft Teams",
		},
		{
			name: "Adobe Photoshop CS6",
			args: args{appId: "177aeb41deb606ae"},
			want: "Adobe Photoshop CS6 (64 Bit)",
		},
		{
			name: "Unknown AppId",
			args: args{appId: "unknown123456789"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LookupApplicationId(tt.args.appId); got != tt.want {
				t.Errorf("LookupApplicationId() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewApplicationId(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want ApplicationId
	}{
		{name: "TestNewApplicationId", args: args{id: "3476342aab319002"}, want: ApplicationId{Id: "3476342aab319002", Name: "Mozilla Firefox"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewApplicationId(tt.args.id); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewApplicationId() = %v, want %v", got, tt.want)
			}
		})
	}
}
