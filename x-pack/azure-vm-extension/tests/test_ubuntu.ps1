 docker build -f ubuntu.dockerfile --tag vm_extension_ubuntu .
 docker run  --name vm_extension_ubuntu_run --rm -i -t vm_extension_ubuntu bash -c "./handler/scripts/linux/install.sh;bash" --privileged=true
