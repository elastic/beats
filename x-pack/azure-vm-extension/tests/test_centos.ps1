 docker build -f centos.dockerfile --tag vm_extension_centos .
 #docker run --env STACK_VERSION=7.10.2 --name vm_extension_centos_run --rm  -t vm_extension_centos  "./handler/scripts/linux/install.sh"
 docker run --env STACK_VERSION=7.10.2  --name vm_extension_centos_run --rm -i -t vm_extension_centos bash -c "./handler/scripts/linux/install.sh;bash" --privileged=true
