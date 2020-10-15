package main

import (
	"github.com/pulumi/pulumi-azure/sdk/v3/go/azure/core"
	"github.com/pulumi/pulumi-azure/sdk/v3/go/azure/storage"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

const (
	resourceGroupName = "observability-beats-test-storage-account"
	location          = "WestEurope"
	storageAccount    = "storageobsaccount"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create an Azure Resource Group
		resourceGroup, err := core.NewResourceGroup(ctx, resourceGroupName, &core.ResourceGroupArgs{
			Location: pulumi.String(location),
		})
		if err != nil {
			return err
		}
		// Create an Azure resource (Storage Account)
		account, err := storage.NewAccount(ctx, storageAccount, &storage.AccountArgs{
			AccountReplicationType: pulumi.String("LRS"),
			AccountTier:            pulumi.String("Standard"),
			ResourceGroupName:      resourceGroup.Name,
		})
		if err != nil {
			return err
		}
		// Export the connection string for the storage account
		ctx.Export("connectionString", account.PrimaryConnectionString)
		return nil
	})
}
