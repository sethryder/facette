package backend

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/facette/maputil"
	"github.com/jinzhu/gorm"
)

// Provider represents a back-end provider item instance.
type Provider struct {
	Item
	Connector       string          `gorm:"type:varchar(32);not null" json:"connector"`
	Settings        maputil.Map     `gorm:"type:text" json:"settings"`
	Filters         ProviderFilters `gorm:"type:text" json:"filters"`
	RefreshInterval int             `gorm:"not null;default:0" json:"refresh_interval"`
	Priority        int             `gorm:"not null;default:0" json:"priority"`
	Enabled         bool            `gorm:"not null;default:true" json:"enabled"`
}

// NewProvider creates a new back-end provider item instance.
func (b *Backend) NewProvider() *Provider {
	return &Provider{Item: Item{backend: b}}
}

// BeforeSave handles the ORM 'BeforeSave' callback.
func (p *Provider) BeforeSave(scope *gorm.Scope) error {
	if err := p.Item.BeforeSave(scope); err != nil {
		return err
	} else if p.RefreshInterval < 0 {
		return ErrInvalidInterval
	} else if p.Priority < 0 {
		return ErrInvalidPriority
	}

	return nil
}

// ProviderFilters represents a list of back-end provider filters.
type ProviderFilters []ProviderFilter

// Value marshals the provider filter entries for compatibility with SQL drivers.
func (pf ProviderFilters) Value() (driver.Value, error) {
	data, err := json.Marshal(pf)
	return data, err
}

// Scan unmarshals the provider filter entries retrieved from SQL drivers.
func (pf *ProviderFilters) Scan(v interface{}) error {
	return scanValue(v, pf)
}

// ProviderFilter represents a back-end provider filter entry instance.
type ProviderFilter struct {
	Action  string `json:"action"`
	Target  string `json:"target"`
	Pattern string `json:"pattern"`
	Into    string `json:"into"`
}
