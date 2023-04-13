//go:generate ../../../tools/readme_config_includer/generator
package azure_monitor_mi

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"

	receiver "github.com/jojokoshy/azure-monitor-metrics-receiver"
)

type AzureMonitorMI struct {
	SubscriptionID       string                 `toml:"subscription_id"`
	ClientID             string                 `toml:"client_id"`
	ClientSecret         string                 `toml:"client_secret"`
	TenantID             string                 `toml:"tenant_id"`
	ResourceTargets      []*ResourceTarget      `toml:"resource_target"`
	ResourceGroupTargets []*ResourceGroupTarget `toml:"resource_group_target"`
	SubscriptionTargets  []*Resource            `toml:"subscription_target"`
	Log                  telegraf.Logger        `toml:"-"`

	receiver     *receiver.AzureMonitorMetricsReceiver
	azureManager azureClientsCreatorMI
	azureClients *receiver.AzureClients
}

type ResourceTarget struct {
	ResourceID   string   `toml:"resource_id"`
	Metrics      []string `toml:"metrics"`
	Aggregations []string `toml:"aggregations"`
}

type ResourceGroupTarget struct {
	ResourceGroup string      `toml:"resource_group"`
	Resources     []*Resource `toml:"resource"`
}

type Resource struct {
	ResourceType string   `toml:"resource_type"`
	Metrics      []string `toml:"metrics"`
	Aggregations []string `toml:"aggregations"`
}

type azureClientsManager struct{}
type azureClientsManagerMI struct{}

type azureClientsCreator interface {
	createAzureClients(subscriptionID string, clientID string, clientSecret string, tenantID string) (*receiver.AzureClients, error)
}

type azureClientsCreatorMI interface {
	createAzureClientsMI(subscriptionID string) (*receiver.AzureClients, error)
}

//go:embed sample.conf
var sampleConfig string

func (r *AzureMonitorMI) Description() string {
	return "Azure Monitoring using MI"
}

func (am *AzureMonitorMI) SampleConfig() string {
	return sampleConfig
}

// Init is for setup, and validating config.
func (am *AzureMonitorMI) Init() error {
	var err error
	//am.azureClients, err = am.azureManager.createAzureClients()
	fmt.Println("Init method called")
	am.azureClients, err = am.azureManager.createAzureClientsMI(am.SubscriptionID)
	if err != nil {
		return err
	}

	if err = am.setReceiver(); err != nil {
		return fmt.Errorf("error setting Azure Monitor receiver: %w", err)
	}

	if err = am.receiver.CreateResourceTargetsFromResourceGroupTargets(); err != nil {
		return fmt.Errorf("error creating resource targets from resource group targets: %w", err)
	}

	if err = am.receiver.CreateResourceTargetsFromSubscriptionTargets(); err != nil {
		return fmt.Errorf("error creating resource targets from subscription targets: %w", err)
	}

	if err = am.receiver.CheckResourceTargetsMetricsValidation(); err != nil {
		return fmt.Errorf("error checking resource targets metrics validation: %w", err)
	}

	if err = am.receiver.SetResourceTargetsMetrics(); err != nil {
		return fmt.Errorf("error setting resource targets metrics: %w", err)
	}

	if err = am.receiver.SplitResourceTargetsMetricsByMinTimeGrain(); err != nil {
		return fmt.Errorf("error spliting resource targets metrics by min time grain: %w", err)
	}

	am.receiver.SplitResourceTargetsWithMoreThanMaxMetrics()
	am.receiver.SetResourceTargetsAggregations()

	am.Log.Debug("Total resource targets: ", len(am.receiver.Targets.ResourceTargets))

	return nil
}

func (am *AzureMonitorMI) Gather(acc telegraf.Accumulator) error {
	var waitGroup sync.WaitGroup

	for _, target := range am.receiver.Targets.ResourceTargets {
		am.Log.Debug("Collecting metrics for resource target ", target.ResourceID)
		waitGroup.Add(1)

		go func(target *receiver.ResourceTarget) {
			defer waitGroup.Done()

			collectedMetrics, notCollectedMetrics, err := am.receiver.CollectResourceTargetMetrics(target)
			if err != nil {
				acc.AddError(err)
			}

			for _, collectedMetric := range collectedMetrics {
				acc.AddFields(collectedMetric.Name, collectedMetric.Fields, collectedMetric.Tags)
			}

			for _, notCollectedMetric := range notCollectedMetrics {
				am.Log.Info("Did not get any metric value from Azure Monitor API for the metric ID ", notCollectedMetric)
			}
		}(target)
	}

	waitGroup.Wait()
	return nil
}

func (am *AzureMonitorMI) setReceiver() error {
	resourceTargets := make([]*receiver.ResourceTarget, 0, len(am.ResourceTargets))
	resourceGroupTargets := make([]*receiver.ResourceGroupTarget, 0, len(am.ResourceGroupTargets))
	subscriptionTargets := make([]*receiver.Resource, 0, len(am.SubscriptionTargets))

	for _, target := range am.ResourceTargets {
		resourceTargets = append(resourceTargets, receiver.NewResourceTarget(target.ResourceID, target.Metrics, target.Aggregations))
	}

	for _, target := range am.ResourceGroupTargets {
		resources := make([]*receiver.Resource, 0, len(target.Resources))
		for _, resource := range target.Resources {
			resources = append(resources, receiver.NewResource(resource.ResourceType, resource.Metrics, resource.Aggregations))
		}

		resourceGroupTargets = append(resourceGroupTargets, receiver.NewResourceGroupTarget(target.ResourceGroup, resources))
	}

	for _, target := range am.SubscriptionTargets {
		subscriptionTargets = append(subscriptionTargets, receiver.NewResource(target.ResourceType, target.Metrics, target.Aggregations))
	}

	targets := receiver.NewTargets(resourceTargets, resourceGroupTargets, subscriptionTargets)
	var err error
	am.receiver, err = receiver.NewAzureMonitorMetricsReceiver(am.SubscriptionID, am.ClientID, am.ClientSecret, am.TenantID, targets, am.azureClients)
	return err
}

func (acm *azureClientsManagerMI) createAzureClientsMI(
	subscriptionID string) (*receiver.AzureClients, error) {
	fmt.Println("JK - call made to createAzureClientsMI")
	azureClients, err := receiver.CreateMIAzureClients(subscriptionID)
	//azureClients, err := receiver.CreateMIAzureClients(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf(" JK - error creating Azure clients for Managed Identity: %w", err)
	}

	return azureClients, nil
}

func (acm *azureClientsManager) createAzureClients(
	subscriptionID string,
	clientID string,
	clientSecret string,
	tenantID string,
) (*receiver.AzureClients, error) {
	fmt.Println("JK - call made to Azure Client using clientid and secret")
	azureClients, err := receiver.CreateAzureClients(subscriptionID, clientID, clientSecret, tenantID)
	//azureClients, err := receiver.CreateMIAzureClients(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("JK - error creating Azure clients: %w", err)
	}

	return azureClients, nil
}

func init() {
	inputs.Add("azure_monitor_mi", func() telegraf.Input {
		fmt.Println("Call to init - JK")
		return &AzureMonitorMI{
			azureManager: &azureClientsManagerMI{},
		}
	})
}
