package virtualmachine

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func init() {
	if err := mb.Registry.AddMetricSet("vsphere", "virtualmachine", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	HostURL         *url.URL
	Insecure        bool
	GetCustomFields bool
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The vsphere virtualmachine metricset is beta")

	config := struct {
		Username        string `config:"username"`
		Password        string `config:"password"`
		Insecure        bool   `config:"insecure"`
		GetCustomFields bool   `config:"get_custom_fields"`
	}{
		GetCustomFields: false,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	u, err := url.Parse(base.HostData().URI)
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(config.Username, config.Password)

	return &MetricSet{
		BaseMetricSet:   base,
		HostURL:         u,
		Insecure:        config.Insecure,
		GetCustomFields: config.GetCustomFields,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := []common.MapStr{}

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return nil, err
	}

	defer client.Logout(ctx)

	c := client.Client

	// Get custom fields (attributes) names if get_custom_fields is true.
	customFieldsMap := make(map[int32]string)
	if m.GetCustomFields {
		var err error
		customFieldsMap, err = setCustomFieldsMap(ctx, c)
		if err != nil {
			return nil, err
		}
	}

	// Create view of VirtualMachine objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}

	defer v.Destroy(ctx)

	// Retrieve summary property for all machines
	var vmt []mo.VirtualMachine
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary"}, &vmt)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, vm := range vmt {

		wg.Add(1)

		go func(vm mo.VirtualMachine) {

			defer wg.Done()

			freeMemory := (int64(vm.Summary.Config.MemorySizeMB) * 1024 * 1024) - (int64(vm.Summary.QuickStats.GuestMemoryUsage) * 1024 * 1024)

			event := common.MapStr{
				"host": vm.Summary.Runtime.Host.Value,
				"name": vm.Summary.Config.Name,
				"cpu": common.MapStr{
					"used": common.MapStr{
						"mhz": vm.Summary.QuickStats.OverallCpuUsage,
					},
				},
				"memory": common.MapStr{
					"used": common.MapStr{
						"guest": common.MapStr{
							"bytes": (int64(vm.Summary.QuickStats.GuestMemoryUsage) * 1024 * 1024),
						},
						"host": common.MapStr{
							"bytes": (int64(vm.Summary.QuickStats.HostMemoryUsage) * 1024 * 1024),
						},
					},
					"total": common.MapStr{
						"guest": common.MapStr{
							"bytes": (int64(vm.Summary.Config.MemorySizeMB) * 1024 * 1024),
						},
					},
					"free": common.MapStr{
						"guest": common.MapStr{
							"bytes": freeMemory,
						},
					},
				},
			}

			// Get custom fields (attributes) values if get_custom_fields is true.
			if m.GetCustomFields {
				customFields := getCustomFields(vm.Summary.CustomValue, customFieldsMap)

				if len(customFields) > 0 {
					event["custom_fields"] = customFields
				}
			}

			mutex.Lock()
			events = append(events, event)
			mutex.Unlock()
		}(vm)
	}

	wg.Wait()

	return events, nil
}

func getCustomFields(customFields []types.BaseCustomFieldValue, customFieldsMap map[int32]string) common.MapStr {
	outputFields := common.MapStr{}
	for _, v := range customFields {
		customFieldString := v.(*types.CustomFieldStringValue)
		key, ok := customFieldsMap[v.GetCustomFieldValue().Key]
		if ok {
			// If key has '.', is replaced with '_' to be compatible with ES2.x.
			fmtKey := strings.Replace(key, ".", "_", -1)
			outputFields.Put(fmtKey, customFieldString.Value)
		}
	}

	return outputFields
}

func setCustomFieldsMap(ctx context.Context, client *vim25.Client) (map[int32]string, error) {
	customFieldsMap := make(map[int32]string)

	customFieldsManager, err := object.GetCustomFieldsManager(client)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get custom fields manager")
	} else {
		field, err := customFieldsManager.Field(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get custom fields")
		}

		for _, def := range field {
			customFieldsMap[def.Key] = def.Name
		}
	}

	return customFieldsMap, nil
}
