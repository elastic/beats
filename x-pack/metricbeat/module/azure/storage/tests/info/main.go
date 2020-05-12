package main

import (
	"github.com/pulumi/pulumi-azure/sdk/v3/go/azure/core"
	"github.com/pulumi/pulumi-azure/sdk/v3/go/azure/storage"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create an Azure Resource Group
		resourceGroup, err := core.NewResourceGroup(ctx, "observability-test-beats", &core.ResourceGroupArgs{
			Location: pulumi.String("WestEurope"),
		})
		if err != nil {
			return err
		}

		// Create an Azure resource (Storage Account)
		account, err := storage.NewAccount(ctx, "teststorageaccount", &storage.AccountArgs{
			AccessTier:             nil,
			AccountKind:            nil,
			AccountReplicationType: pulumi.String("LRS"),
			AccountTier:            pulumi.String("Standard"),
			BlobProperties:         nil,
			CustomDomain:           nil,
			EnableHttpsTrafficOnly: nil,
			Identity:               nil,
			IsHnsEnabled:           nil,
			Location:               pulumi.String("WestEurope"),
			Name:                   pulumi.String("teststorageaccount"),
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
