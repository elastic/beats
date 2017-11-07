// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package simulator

import (
	"os"
	"path"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/vic/pkg/vsphere/simulator/esx"
)

type FileManager struct {
	mo.FileManager
}

func NewFileManager(ref types.ManagedObjectReference) object.Reference {
	m := &FileManager{}
	m.Self = ref
	return m
}

func (f *FileManager) findDatastore(dc *types.ManagedObjectReference, name string) (*Datastore, types.BaseMethodFault) {
	if dc == nil {
		dc = &esx.Datacenter.Self
	}

	folder := Map.Get(Map.Get(*dc).(*mo.Datacenter).DatastoreFolder).(*Folder)

	ds := Map.FindByName(name, folder.ChildEntity)
	if ds == nil {
		return nil, &types.InvalidDatastore{Name: name}
	}

	return ds.(*Datastore), nil
}

func (f *FileManager) fault(name string, err error, fault types.BaseFileFault) types.BaseMethodFault {
	switch {
	case os.IsNotExist(err):
		fault = new(types.FileNotFound)
	}

	fault.GetFileFault().File = name

	return fault.(types.BaseMethodFault)
}

type deleteDatastoreFileTask struct {
	*FileManager

	req *types.DeleteDatastoreFile_Task
}

func (s *deleteDatastoreFileTask) Run(Task *Task) (types.AnyType, types.BaseMethodFault) {
	p, fault := parseDatastorePath(s.req.Name)
	if fault != nil {
		return nil, fault
	}

	ds, fault := s.findDatastore(s.req.Datacenter, p.Datastore)
	if fault != nil {
		return nil, fault
	}

	dir := ds.Info.GetDatastoreInfo().Url
	file := path.Join(dir, p.Path)

	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, s.fault(file, err, new(types.CannotDeleteFile))
		}
	}

	err = os.RemoveAll(file)
	if err != nil {
		return nil, s.fault(file, err, new(types.CannotDeleteFile))
	}

	return nil, nil
}

func (f *FileManager) DeleteDatastoreFileTask(d *types.DeleteDatastoreFile_Task) soap.HasFault {
	task := NewTask(&deleteDatastoreFileTask{f, d})

	task.Run()

	return &methods.DeleteDatastoreFile_TaskBody{
		Res: &types.DeleteDatastoreFile_TaskResponse{
			Returnval: task.Self,
		},
	}
}

func (f *FileManager) MakeDirectory(r *types.MakeDirectory) soap.HasFault {
	body := &methods.MakeDirectoryBody{}

	p, fault := parseDatastorePath(r.Name)
	if fault != nil {
		body.Fault_ = Fault("", fault)
		return body
	}

	ds, fault := f.findDatastore(r.Datacenter, p.Datastore)
	if fault != nil {
		body.Fault_ = Fault("", fault)
		return body
	}

	name := path.Join(ds.Info.GetDatastoreInfo().Url, p.Path)

	mkdir := os.Mkdir

	if isTrue(r.CreateParentDirectories) {
		mkdir = os.MkdirAll
	}

	err := mkdir(name, 0700)
	if err != nil {
		fault = f.fault(r.Name, err, new(types.CannotCreateFile))
		body.Fault_ = Fault(err.Error(), fault)
		return body
	}

	return body
}

type moveDatastoreFileTask struct {
	*FileManager

	req *types.MoveDatastoreFile_Task
}

func (s *moveDatastoreFileTask) Run(Task *Task) (types.AnyType, types.BaseMethodFault) {
	src, fault := parseDatastorePath(s.req.SourceName)
	if fault != nil {
		return nil, fault
	}

	srcDs, fault := s.findDatastore(s.req.SourceDatacenter, src.Datastore)
	if fault != nil {
		return nil, fault
	}

	srcDir := srcDs.Info.GetDatastoreInfo().Url
	srcFile := path.Join(srcDir, src.Path)

	dst, fault := parseDatastorePath(s.req.DestinationName)
	if fault != nil {
		return nil, fault
	}

	dstDs, fault := s.findDatastore(s.req.DestinationDatacenter, dst.Datastore)
	if fault != nil {
		return nil, fault
	}

	dstDir := dstDs.Info.GetDatastoreInfo().Url
	dstFile := path.Join(dstDir, dst.Path)

	if !isTrue(s.req.Force) {
		_, err := os.Stat(dstFile)
		if err == nil {
			return nil, &types.FileAlreadyExists{
				FileFault: types.FileFault{
					File: dstFile,
				},
			}
		}
	}

	err := os.Rename(srcFile, dstFile)
	if err != nil {
		return nil, s.fault(srcFile, err, new(types.CannotAccessFile))
	}

	return nil, nil
}

func (f *FileManager) MoveDatastoreFileTask(d *types.MoveDatastoreFile_Task) soap.HasFault {
	task := NewTask(&moveDatastoreFileTask{f, d})

	task.Run()

	return &methods.MoveDatastoreFile_TaskBody{
		Res: &types.MoveDatastoreFile_TaskResponse{
			Returnval: task.Self,
		},
	}
}
