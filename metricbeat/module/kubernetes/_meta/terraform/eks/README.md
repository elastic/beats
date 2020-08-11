Terraform scenario to start a Kubernetes cluster in AWS EKS.

`kubectl` will be configured to use this cluster if `awscli >= 1.18` is
available, you can find a requirements.txt file in this directory to prepare
a virtual environment for `awscli`.

To start this scenario:

```
$ terraform init
$ terraform apply
```

It will ask for a cluster name.

Remember to destroy the scenario once you don't need it:

```
$ terraform destroy
```
