package models

// ResPartner represents a customer/supplier from Odoo (res.partner)
type ResPartner struct {
	ID          int64      `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name        string     `gorm:"index" json:"name" xmlrpc:"name"`
	Street      OdooString `json:"street" xmlrpc:"street"`
	Street2     OdooString `json:"street2" xmlrpc:"street2"`
	Zip         OdooString `json:"zip" xmlrpc:"zip"`
	City        OdooString `json:"city" xmlrpc:"city"`
	StateID     *int64     `json:"state_id" xmlrpc:"state_id"`     // Federal state/region
	CountryID   *int64     `json:"country_id" xmlrpc:"country_id"` // Country (res.country)
	Phone       OdooString `json:"phone" xmlrpc:"phone"`
	Email       OdooString `json:"email" xmlrpc:"email"`
	Vat         OdooString `json:"vat" xmlrpc:"vat"`                   // Tax ID
	CompanyType string     `json:"company_type" xmlrpc:"company_type"` // 'person' or 'company'
	IsCompany   bool       `json:"is_company" xmlrpc:"is_company"`
}

func (ResPartner) TableName() string { return "res_partner" }
