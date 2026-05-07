package app

import (
	"redshell/internal/marketplace"
)

type MarketplaceApp struct {
	svc *marketplace.Service
}

func NewMarketplaceApp(svc *marketplace.Service) *MarketplaceApp {
	return &MarketplaceApp{svc: svc}
}

func (a *MarketplaceApp) List() ([]marketplace.Marketplace, error) {
	return a.svc.List()
}

func (a *MarketplaceApp) Add(rawURL string) (marketplace.Marketplace, error) {
	return a.svc.Add(rawURL)
}

func (a *MarketplaceApp) Remove(id string) error {
	return a.svc.Remove(id)
}

type RefreshResult struct {
	Refreshed []string `json:"refreshed"`
	Errors    []string `json:"errors"`
}

func (a *MarketplaceApp) Refresh() RefreshResult {
	refreshed, errs := a.svc.RefreshAll()
	return RefreshResult{Refreshed: refreshed, Errors: errs}
}
