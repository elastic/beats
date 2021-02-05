 docker build -f ubuntu.dockerfile --tag vm_extension_ubuntu .
 #docker run --env STACK_VERSION=7.10.2 --name vm_extension_ubuntu_run --rm  -t vm_extension_ubuntu  "./handler/scripts/linux/install.sh"
 docker run --env STACK_VERSION=7.10.2 --name vm_extension_ubuntu_run --rm -i -t vm_extension_ubuntu bash -c "./handler/scripts/linux/install.sh;bash"
