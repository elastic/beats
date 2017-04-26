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
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var Map = NewRegistry()

type Registry struct {
	m       sync.Mutex
	objects map[types.ManagedObjectReference]mo.Reference
	counter int
}

func NewRegistry() *Registry {
	r := &Registry{
		objects: make(map[types.ManagedObjectReference]mo.Reference),
	}

	return r
}

func TypeName(item mo.Reference) string {
	return reflect.TypeOf(item).Elem().Name()
}

// newReference returns a new MOR, where Type defaults to type of the given item
// and Value defaults to a unique id for the given type.
func (r *Registry) newReference(item mo.Reference) types.ManagedObjectReference {
	ref := item.Reference()

	if ref.Type == "" {
		ref.Type = TypeName(item)
	}

	if ref.Value == "" {
		r.counter++
		ref.Value = fmt.Sprintf("%s-%d", strings.ToLower(ref.Type), r.counter)
	}

	return ref
}

// NewEntity sets Entity().Self with a new, unique Value.
// Useful for creating object instances from templates.
func (r *Registry) NewEntity(item mo.Entity) mo.Entity {
	e := item.Entity()
	e.Self.Value = ""
	e.Self = r.newReference(item)
	return item
}

func (r *Registry) PutEntity(parent mo.Entity, item mo.Entity) mo.Entity {
	e := item.Entity()

	if parent != nil {
		e.Parent = &parent.Entity().Self
	}

	r.Put(item)

	return item
}

func (r *Registry) Get(ref types.ManagedObjectReference) mo.Reference {
	r.m.Lock()
	defer r.m.Unlock()

	return r.objects[ref]
}

// Any returns the first instance of entity type specified by kind.
func (r *Registry) Any(kind string) mo.Entity {
	r.m.Lock()
	defer r.m.Unlock()

	for ref, val := range r.objects {
		if ref.Type == kind {
			return val.(mo.Entity)
		}
	}

	return nil
}

func (r *Registry) Put(item mo.Reference) mo.Reference {
	r.m.Lock()
	defer r.m.Unlock()

	ref := item.Reference()
	if ref.Type == "" || ref.Value == "" {
		ref = r.newReference(item)
		// mo.Reference() returns a value, not a pointer so use reflect to set the Self field
		reflect.ValueOf(item).Elem().FieldByName("Self").Set(reflect.ValueOf(ref))
	}

	r.objects[ref] = item

	return item
}

func (r *Registry) Remove(item types.ManagedObjectReference) {
	r.m.Lock()
	defer r.m.Unlock()

	delete(r.objects, item)
}

// getEntityParent traverses up the inventory and returns the first object of type kind.
// If no object of type kind is found, the method will panic when it reaches the
// inventory root Folder where the Parent field is nil.
func (r *Registry) getEntityParent(item mo.Entity, kind string) mo.Entity {
	for {
		parent := item.Entity().Parent

		item = r.Get(*parent).(mo.Entity)

		if item.Reference().Type == kind {
			return item
		}
	}
}

// getEntityDatacenter returns the Datacenter containing the given item
func (r *Registry) getEntityDatacenter(item mo.Entity) *mo.Datacenter {
	return r.getEntityParent(item, "Datacenter").(*mo.Datacenter)
}

func (r *Registry) getEntityFolder(item mo.Entity, kind string) *Folder {
	dc := Map.getEntityDatacenter(item)

	var ref types.ManagedObjectReference

	switch kind {
	case "datastore":
		ref = dc.DatastoreFolder
	}

	folder := r.Get(ref).(*Folder)

	// If Model was created with Folder option, use that Folder; else use top-level folder
	for _, child := range folder.ChildEntity {
		if child.Type == "Folder" {
			folder = Map.Get(child).(*Folder)
			break
		}
	}

	return folder
}

// getEntityComputeResource returns the ComputeResource parent for the given item.
// A ResourcePool for example may have N Parents of type ResourcePool, but the top
// most Parent pool is always a ComputeResource child.
func (r *Registry) getEntityComputeResource(item mo.Entity) mo.Entity {
	for {
		parent := item.Entity().Parent

		item = r.Get(*parent).(mo.Entity)

		switch item.Reference().Type {
		case "ComputeResource":
			return item
		case "ClusterComputeResource":
			return item
		}
	}
}

// FindByName returns the first mo.Entity of the given refs whose Name field is equal to the given name.
// If there is no match, nil is returned.
// This method is useful for cases where objects are required to have a unique name, such as Datastore with
// a HostStorageSystem or HostSystem within a ClusterComputeResource.
func (r *Registry) FindByName(name string, refs []types.ManagedObjectReference) mo.Entity {
	for _, ref := range refs {
		e := r.Get(ref).(mo.Entity)

		if name == e.Entity().Name {
			return e
		}
	}

	return nil
}

// FindReference returns the 1st match found in refs, or nil if not found.
func FindReference(refs []types.ManagedObjectReference, match ...types.ManagedObjectReference) *types.ManagedObjectReference {
	for _, ref := range refs {
		for _, m := range match {
			if ref == m {
				return &ref
			}
		}
	}

	return nil
}

// RemoveReference returns a slice with ref removed from refs
func RemoveReference(ref types.ManagedObjectReference, refs []types.ManagedObjectReference) []types.ManagedObjectReference {
	var result []types.ManagedObjectReference

	for i, r := range refs {
		if r == ref {
			result = append(result, refs[i+1:]...)
			break
		}

		result = append(result, r)
	}

	return result
}

// AddReference returns a slice with ref appended if not already in refs.
func AddReference(ref types.ManagedObjectReference, refs []types.ManagedObjectReference) []types.ManagedObjectReference {
	if FindReference(refs, ref) == nil {
		return append(refs, ref)
	}

	return refs
}

func (r *Registry) content() types.ServiceContent {
	return r.Get(serviceInstance).(*ServiceInstance).Content
}

// IsESX returns true if this Registry maps an ESX model
func (r *Registry) IsESX() bool {
	return r.content().About.ApiType == "HostAgent"
}

// IsVPX returns true if this Registry maps a VPX model
func (r *Registry) IsVPX() bool {
	return !r.IsESX()
}

// SearchIndex returns the SearchIndex singleton
func (r *Registry) SearchIndex() *SearchIndex {
	return r.Get(r.content().SearchIndex.Reference()).(*SearchIndex)
}

// FileManager returns the FileManager singleton
func (r *Registry) FileManager() *FileManager {
	return r.Get(r.content().FileManager.Reference()).(*FileManager)
}

// ViewManager returns the ViewManager singleton
func (r *Registry) ViewManager() *ViewManager {
	return r.Get(r.content().ViewManager.Reference()).(*ViewManager)
}
