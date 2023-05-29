// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pkg

import (
	"errors"
	"fmt"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/package/schema"
)

// Requires the Google flatbuffer compiler and Elastic go-licenser.
//go:generate flatc --go schema.fbs
//go:generate go-licenser -license=Elastic schema

var bufferPool sync.Pool

func init() {
	bufferPool.New = func() interface{} {
		return flatbuffers.NewBuilder(1024)
	}
}

// fbGetBuilder returns a Builder that can be used for encoding data. The builder
// should be put back into the pool by invoking the put function after the encoded bytes
// are no longer in used (i.e. a copy of b.FinishedBytes() has been made).
func fbGetBuilder() (b *flatbuffers.Builder, put func()) {
	b = bufferPool.Get().(*flatbuffers.Builder)
	b.Reset()
	return b, func() { bufferPool.Put(b) }
}

// encodePackages encodes an array of packages by creating a vector of packages and tracking offsets. It uses the
// func fbEncodePackage to encode individual packages, and returns a []byte containing the encoded data.
func encodePackages(builder *flatbuffers.Builder, packages []*Package) []byte {
	offsets := make([]flatbuffers.UOffsetT, len(packages))

	for i, p := range packages {
		offsets[i] = fbEncodePackage(builder, p)
	}
	schema.PackageContainerStartPackagesVector(builder, len(offsets))
	for _, offset := range offsets {
		builder.PrependUOffsetT(offset)
	}
	packageContainerVector := builder.EndVector(len(offsets))
	schema.PackageContainerStart(builder)
	schema.PackageContainerAddPackages(builder, packageContainerVector)
	root := schema.PackageContainerEnd(builder)
	builder.Finish(root)
	return builder.FinishedBytes()
}

// fbEncodePackage encodes the given Package to a flatbuffer. The returned bytes
// are a pointer into the Builder's memory.
func fbEncodePackage(b *flatbuffers.Builder, p *Package) flatbuffers.UOffsetT {
	if p == nil {
		return 0
	}

	return fbWritePackage(b, p)
}

func fbWritePackage(b *flatbuffers.Builder, p *Package) flatbuffers.UOffsetT {
	if p == nil {
		return 0
	}

	var packageNameOffset,
		packageVersionOffset,
		packageReleaseOffset,
		packageArchOffset,
		packageLicenseOffset,
		packageSummaryOffset,
		packageURLOffset,
		packageTypeOffset,
		packageErrorOffset flatbuffers.UOffsetT

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
	if p.error != nil {
		packageErrorOffset = b.CreateString(p.error.Error())
	}

	schema.PackageStart(b)
	schema.PackageAddInstalltime(b, uint64(p.InstallTime.Unix()))
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
	if packageErrorOffset > 0 {
		schema.PackageAddError(b, packageErrorOffset)
	}

	return schema.PackageEnd(b)
}

// decodePackagesFromContainer accepts a flatbuffer encoded byte slice, and decodes
// each package from the container vector with the help of fbDecodePackage.
// It returns an array of package objects.
func decodePackagesFromContainer(data []byte) ([]*Package, error) {
	var packages []*Package
	container := schema.GetRootAsPackageContainer(data, 0)
	for i := 0; i < container.PackagesLength(); i++ {
		sPkg := schema.Package{}
		done := container.Packages(&sPkg, i)
		if !done {
			return nil, fmt.Errorf("failed to load package at container vector position: %d", i)
		} else {
			p := fbDecodePackage(&sPkg)
			packages = append(packages, p)
		}
	}
	return packages, nil
}

// fbDecodePackage decodes flatbuffer package data and copies it into a Package
// object that is returned.
func fbDecodePackage(p *schema.Package) *Package {
	var err error
	if string(p.Error()) != "" {
		err = errors.New(string(p.Error()))
	}

	return &Package{
		Name:        string(p.Name()),
		Version:     string(p.Version()),
		Release:     string(p.Release()),
		Arch:        string(p.Arch()),
		License:     string(p.License()),
		InstallTime: time.Unix(int64(p.Installtime()), 0).UTC(),
		Size:        p.Size(),
		Summary:     string(p.Summary()),
		URL:         string(p.Url()),
		Type:        string(p.Type()),
		error:       err,
	}
}
