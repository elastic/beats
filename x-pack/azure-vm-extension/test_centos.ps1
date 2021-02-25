 docker build -f centos.dockerfile --tag vm_extension_centos .
 docker run  --name vm_extension_centos_run --rm -i -t vm_extension_centos bash -c "./handler/scripts/linux/install.sh;bash" --privileged=true
