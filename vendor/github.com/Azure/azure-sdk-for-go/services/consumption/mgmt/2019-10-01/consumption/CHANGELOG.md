# Change History

## Breaking Changes

### Removed Constants

1. BillingFrequency.Month
1. BillingFrequency.Quarter
1. BillingFrequency.Year
1. Bound.Lower
1. Bound.Upper
1. Datagrain.DailyGrain
1. Datagrain.MonthlyGrain
1. EventType.NewCredit
1. EventType.PendingAdjustments
1. EventType.PendingCharges
1. EventType.PendingExpiredCredit
1. EventType.PendingNewCredit
1. EventType.SettledCharges
1. EventType.UnKnown
1. Grain.Daily
1. Grain.Monthly
1. Grain.Yearly
1. LookBackPeriod.Last07Days
1. LookBackPeriod.Last30Days
1. LookBackPeriod.Last60Days
1. LotSource.PromotionalCredit
1. LotSource.PurchasedCredit
1. Metrictype.ActualCostMetricType
1. Metrictype.AmortizedCostMetricType
1. Metrictype.UsageMetricType
1. OperatorType.EqualTo
1. OperatorType.GreaterThan
1. OperatorType.GreaterThanOrEqualTo
1. Scope11.Shared
1. Scope11.Single
1. Scope9.Scope9Shared
1. Scope9.Scope9Single
1. Term.P1Y
1. Term.P3Y
1. ThresholdType.Actual

### Removed Funcs

1. PossibleScope11Values() []Scope11
1. PossibleScope9Values() []Scope9

### Signature Changes

#### Funcs

1. CreditsClient.Get
	- Params
		- From: context.Context, string, string
		- To: context.Context, string
1. CreditsClient.GetPreparer
	- Params
		- From: context.Context, string, string
		- To: context.Context, string
1. EventsClient.List
	- Params
		- From: context.Context, string, string, string, string
		- To: context.Context, string, string, string
1. EventsClient.ListComplete
	- Params
		- From: context.Context, string, string, string, string
		- To: context.Context, string, string, string
1. EventsClient.ListPreparer
	- Params
		- From: context.Context, string, string, string, string
		- To: context.Context, string, string, string
1. LotsClient.List
	- Params
		- From: context.Context, string, string
		- To: context.Context, string
1. LotsClient.ListComplete
	- Params
		- From: context.Context, string, string
		- To: context.Context, string
1. LotsClient.ListPreparer
	- Params
		- From: context.Context, string, string
		- To: context.Context, string
1. ReservationRecommendationDetailsClient.Get
	- Params
		- From: context.Context, string, Scope11, string, Term, LookBackPeriod, string
		- To: context.Context, string, Scope14, string, Term, LookBackPeriod, string
1. ReservationRecommendationDetailsClient.GetPreparer
	- Params
		- From: context.Context, string, Scope11, string, Term, LookBackPeriod, string
		- To: context.Context, string, Scope14, string, Term, LookBackPeriod, string

#### Struct Fields

1. LegacyReservationRecommendationProperties.InstanceFlexibilityRatio changed type from *int32 to *float64
1. ModernReservationRecommendationProperties.InstanceFlexibilityRatio changed type from *int32 to *float64
1. ModernReservationRecommendationProperties.LookBackPeriod changed type from *string to *int32
1. ModernUsageDetailProperties.MeterID changed type from *uuid.UUID to *string

## Additive Changes

### New Constants

1. BillingFrequency.BillingFrequencyMonth
1. BillingFrequency.BillingFrequencyQuarter
1. BillingFrequency.BillingFrequencyYear
1. Bound.BoundLower
1. Bound.BoundUpper
1. CultureCode.CultureCodeCsCz
1. CultureCode.CultureCodeDaDk
1. CultureCode.CultureCodeDeDe
1. CultureCode.CultureCodeEnGb
1. CultureCode.CultureCodeEnUs
1. CultureCode.CultureCodeEsEs
1. CultureCode.CultureCodeFrFr
1. CultureCode.CultureCodeHuHu
1. CultureCode.CultureCodeItIt
1. CultureCode.CultureCodeJaJp
1. CultureCode.CultureCodeKoKr
1. CultureCode.CultureCodeNbNo
1. CultureCode.CultureCodeNlNl
1. CultureCode.CultureCodePlPl
1. CultureCode.CultureCodePtBr
1. CultureCode.CultureCodePtPt
1. CultureCode.CultureCodeRuRu
1. CultureCode.CultureCodeSvSe
1. CultureCode.CultureCodeTrTr
1. CultureCode.CultureCodeZhCn
1. CultureCode.CultureCodeZhTw
1. Datagrain.DatagrainDailyGrain
1. Datagrain.DatagrainMonthlyGrain
1. EventType.EventTypeNewCredit
1. EventType.EventTypePendingAdjustments
1. EventType.EventTypePendingCharges
1. EventType.EventTypePendingExpiredCredit
1. EventType.EventTypePendingNewCredit
1. EventType.EventTypeSettledCharges
1. EventType.EventTypeUnKnown
1. Grain.GrainDaily
1. Grain.GrainMonthly
1. Grain.GrainYearly
1. LookBackPeriod.LookBackPeriodLast07Days
1. LookBackPeriod.LookBackPeriodLast30Days
1. LookBackPeriod.LookBackPeriodLast60Days
1. LotSource.LotSourcePromotionalCredit
1. LotSource.LotSourcePurchasedCredit
1. Metrictype.MetrictypeActualCostMetricType
1. Metrictype.MetrictypeAmortizedCostMetricType
1. Metrictype.MetrictypeUsageMetricType
1. OperatorType.OperatorTypeEqualTo
1. OperatorType.OperatorTypeGreaterThan
1. OperatorType.OperatorTypeGreaterThanOrEqualTo
1. Scope12.Scope12Shared
1. Scope12.Scope12Single
1. Scope14.Scope14Shared
1. Scope14.Scope14Single
1. Term.TermP1Y
1. Term.TermP3Y
1. ThresholdType.ThresholdTypeActual

### New Funcs

1. AmountWithExchangeRate.MarshalJSON() ([]byte, error)
1. DownloadProperties.MarshalJSON() ([]byte, error)
1. ForecastSpend.MarshalJSON() ([]byte, error)
1. HighCasedErrorDetails.MarshalJSON() ([]byte, error)
1. PossibleCultureCodeValues() []CultureCode
1. PossibleScope12Values() []Scope12
1. PossibleScope14Values() []Scope14
1. Reseller.MarshalJSON() ([]byte, error)
1. TagProperties.MarshalJSON() ([]byte, error)

### Struct Changes

#### New Structs

1. AmountWithExchangeRate
1. DownloadProperties
1. ForecastSpend
1. HighCasedErrorDetails
1. HighCasedErrorResponse
1. Reseller

#### New Struct Fields

1. Balance.Etag
1. BudgetProperties.ForecastSpend
1. ChargeSummary.Etag
1. CreditBalanceSummary.CurrentBalanceInBillingCurrency
1. CreditBalanceSummary.EstimatedBalanceInBillingCurrency
1. CreditSummary.Etag
1. CreditSummaryProperties.BillingCurrency
1. CreditSummaryProperties.CreditCurrency
1. CreditSummaryProperties.Reseller
1. EventProperties.AdjustmentsInBillingCurrency
1. EventProperties.BillingCurrency
1. EventProperties.ChargesInBillingCurrency
1. EventProperties.ClosedBalanceInBillingCurrency
1. EventProperties.CreditCurrency
1. EventProperties.CreditExpiredInBillingCurrency
1. EventProperties.NewCreditInBillingCurrency
1. EventProperties.Reseller
1. EventSummary.Etag
1. Forecast.Etag
1. LegacyChargeSummary.Etag
1. LegacyReservationRecommendation.Etag
1. LegacyReservationRecommendationProperties.ResourceType
1. LegacyUsageDetail.Etag
1. LotProperties.BillingCurrency
1. LotProperties.ClosedBalanceInBillingCurrency
1. LotProperties.CreditCurrency
1. LotProperties.OriginalAmountInBillingCurrency
1. LotProperties.Reseller
1. LotSummary.Etag
1. ManagementGroupAggregatedCostResult.Etag
1. Marketplace.Etag
1. MarketplaceProperties.AdditionalInfo
1. ModernChargeSummary.Etag
1. ModernReservationRecommendation.ETag
1. ModernReservationRecommendation.Etag
1. ModernReservationRecommendationProperties.Location
1. ModernReservationRecommendationProperties.ResourceType
1. ModernReservationRecommendationProperties.SkuName
1. ModernReservationRecommendationProperties.SubscriptionID
1. ModernUsageDetail.Etag
1. ModernUsageDetailProperties.PayGPrice
1. Notification.Locale
1. Operation.ID
1. OperationDisplay.Description
1. PriceSheetModel.Download
1. PriceSheetResult.Etag
1. ReservationDetail.Etag
1. ReservationRecommendation.Etag
1. ReservationRecommendationDetailsModel.ETag
1. ReservationRecommendationDetailsModel.Etag
1. ReservationRecommendationsListResult.PreviousLink
1. ReservationRecommendationsListResult.TotalCost
1. ReservationSummary.Etag
1. Resource.Etag
1. Tag.Value
1. TagProperties.NextLink
1. TagProperties.PreviousLink
1. UsageDetail.Etag
