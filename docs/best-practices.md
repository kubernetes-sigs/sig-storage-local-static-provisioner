# Best Practices

* For IO isolation, a whole disk per volume is recommended
* For capacity isolation, separate partitions per volume is recommended
* Avoid recreating nodes with the same node name while there are still old PVs
  with that node's affinity specified. Otherwise, the system could think that
  the new node contains the old PVs.
* For volumes with a filesystem, it's recommended to utilize their UUID (e.g.
  the output from `ls -l /dev/disk/by-uuid`) both in fstab entries
  and in the directory name of that mount point. This practice ensures
  that the wrong local volume is not mistakenly mounted, even if its device path
  changes (e.g. if /dev/sda1 becomes /dev/sdb1 when a new disk is added).
  Additionally, this practice will ensure that if another node with the
  same name is created, that any volumes on that node are unique and not
  mistaken for a volume on another node with the same name.
* For raw block volumes without a filesystem, use a unique ID as the symlink
  name. Depending on your environment, the volume's ID in `/dev/disk/by-id/`
  may contain a unique hardware serial number. Otherwise, a unique ID should be 
  generated. The uniqueness of the symlink name will ensure that if another 
  node with the same name is created, that any volumes on that node are 
  unique and not mistaken for a volume on another node with the same name.
