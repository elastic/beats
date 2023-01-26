### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State persistent volume metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/persistentvolume.go):
  declaration and description

### Metrics insight

All metrics have the label:
- persistentvolume

Additionally:
- kube_persistentvolume_capacity_bytes
- kube_persistentvolume_status_phase
  - phase
- kube_persistentvolume_labels
- kube_persistentvolume_info
  - storageclass
  - gce_persistent_disk_name
  - ebs_volume_id
  - azure_disk_name
  - fc_wwids
  - fc_lun
  - fc_target_wwns
  - iscsi_target_portal
  - iscsi_iqn
  - iscsi_lun
  - iscsi_initiator_name
  - nfs_server
  - nfs_path
  - csi_driver
  - csi_volume_handle
  - local_path
  - local_fs
  - host_path
  - host_path_type



### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.

