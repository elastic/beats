# Elastic Agent VM extension

The ElasticAgent VM extension is a small application that provides post-deployment configuration and automation on Azure VMs.
Once installed, it will download the elastic agent artifacts, install the elastic agent on the virtual machine, enroll it to Fleet and then start the agent service.

The Elastic Agent VM extension can be managed using the Azure CLI, PowerShell, Resource Manager templates, and in the future the Azure portal.
For a successful installation the following configuration settings are required:

Public settings:
 - username - a valid username that can have access to the elastic cloud cluster
 - cloudId - the elastic cloud ID (deployment ID)

Protected settings:
 - password - a valid password that can be used in combination with the username public setting to access the elastic cloud cluster




