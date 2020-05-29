#!/usr/bin/env python3

import os
import distutils
from distutils import dir_util

# Support filtering in the distutils.dir_util.copy_tree
#ORIG_COPY_TREE = distutils.dir_util.copy_tree
#
#
# def my_copy_tree(src, *args, **kwargs):
#    '''function my_copy_tree
#        Override distutils.dir_util.copy_tree to filter the system-tests
#    '''
#    if src.endswith('system-tests'):
#        return []
#    return ORIG_COPY_TREE(src, *args, **kwargs)
#
#
#distutils.dir_util.copy_tree = my_copy_tree

if __name__ == "__main__":

    if not os.path.exists('build'):
        os.makedirs('build')

    # Top level folders to be excluded
    EXCLUDE = set(['.ci', '.git', '.github', 'vendor', 'dev-tools'])
    for root, dirs, files in os.walk('.'):
        dirs[:] = [d for d in dirs if d not in EXCLUDE]
        if root.endswith(('build')) and not root.startswith((".{}build".format(os.sep))):
            dest = os.path.join('build', root.replace(".{}".format(os.sep), ''))
            print("Copy {} into {}".format(root, dest))
            distutils.dir_util.copy_tree(root, dest, preserve_symlinks=1)
