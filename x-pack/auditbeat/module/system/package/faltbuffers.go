// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pkg

import (
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/package/schema"
	flatbuffers "github.com/google/flatbuffers/go"
)

// Requires the Google flatbuffer compiler and Elastic go-licenser.
//go:generate flatc --go schema.fbs
//go:generate go-licenser schema

var bufferPool sync.Pool

func init() {
	bufferPool.New = func() interface{} {
		return flatbuffers.NewBuilder(1024)
	}
}

// fbGetBuilder returns a Builder that can be used for encoding data. The builder
// should be released by invoking the release function after the encoded bytes
// are no longer in used (i.e. a copy of b.FinishedBytes() has been made).
func fbGetBuilder() (b *flatbuffers.Builder, release func()) {
	b = bufferPool.Get().(*flatbuffers.Builder)
	b.Reset()
	return b, func() { bufferPool.Put(b) }
}

// fbEncodePackage encodes the given Package to a flatbuffer. The returned bytes
// are a pointer into the Builder's memory.
func fbEncodePackage(b *flatbuffers.Builder, p *Package) []byte {
	if p == nil {
		return nil
	}

	offset := fbWritePackage(b, p)
	b.Finish(offset)
	return b.FinishedBytes()
}

func fbWritePackage(b *flatbuffers.Builder, p *Package) flatbuffers.UOffsetT {
	if p == nil {
		return 0
	}

	var packageNameOffset flatbuffers.UOffsetT
	var packageVersionOffset flatbuffers.UOffsetT
	var packageReleaseOffset flatbuffers.UOffsetT
	var packageArchOffset flatbuffers.UOffsetT
	var packageLicenseOffset flatbuffers.UOffsetT
	var packageSummaryOffset flatbuffers.UOffsetT
	var packageURLOffset flatbuffers.UOffsetT
	var packageTypeOffset flatbuffers.UOffsetT

	if p.Name != "" {
		packageNameOffset = b.CreateString(p.Name)
	}
	if p.Version != "" {
		packageVersionOffset = b.CreateString(p.Version)
	}
	if p.Release != "" {
		packageReleaseOffset = b.CreateString(p.Release)
	}
	if p.Arch != "" {
		packageArchOffset = b.CreateString(p.Arch)
	}
	if p.License != "" {
		packageLicenseOffset = b.CreateString(p.License)
	}
	if p.Summary != "" {
		packageSummaryOffset = b.CreateString(p.Summary)
	}
	if p.URL != "" {
		packageURLOffset = b.CreateString(p.URL)
	}
	if p.Type != "" {
		packageTypeOffset = b.CreateString(p.Type)
	}

	schema.PackageStart(b)
	schema.PackageAddInstalltime(b, uint64(p.InstallTime.UnixNano()))
	schema.PackageAddSize(b, p.Size)

	if packageNameOffset > 0 {
		schema.PackageAddName(b, packageNameOffset)
	}
	if packageVersionOffset > 0 {
		schema.PackageAddVersion(b, packageVersionOffset)
	}
	if packageReleaseOffset > 0 {
		schema.PackageAddRelease(b, packageReleaseOffset)
	}
	if packageArchOffset > 0 {
		schema.PackageAddArch(b, packageArchOffset)
	}
	if packageLicenseOffset > 0 {
		schema.PackageAddLicense(b, packageLicenseOffset)
	}
	if packageSummaryOffset > 0 {
		schema.PackageAddSummary(b, packageSummaryOffset)
	}
	if packageURLOffset > 0 {
		schema.PackageAddUrl(b, packageURLOffset)
	}
	if packageTypeOffset > 0 {
		schema.PackageAddType(b, packageTypeOffset)
	}

	return schema.PackageEnd(b)
}

// fbDecodePackage decodes flatbuffer event data and copies it into a Package
// object that is returned.
func fbDecodePackage(buf []byte, offset int) *Package {
	p := schema.GetRootAsPackage(buf[offset:], 0)

	rtnPkg := &Package{
		Name:        string(p.Name()),
		Version:     string(p.Version()),
		Release:     string(p.Release()),
		Arch:        string(p.Arch()),
		License:     string(p.License()),
		InstallTime: time.Unix(int64(p.Installtime()), 0),
		Size:        p.Size(),
		Summary:     string(p.Summary()),
		URL:         string(p.Url()),
		Type:        string(p.Type()),
	}

	return rtnPkg
}

// fbIsPackageTimestampBefore returns true if the package's timestamp is before
// the given ts. This convenience function allows you to compare the package's
// timestamp without fully decoding and copying the flatbuffer package data.
func fbIsPackageTimestampBefore(buf []byte, ts time.Time) bool {
	p := schema.GetRootAsPackage(buf, 0)
	packageInstallTime := time.Unix(0, int64(p.Installtime()))
	return packageInstallTime.Before(ts)
}
