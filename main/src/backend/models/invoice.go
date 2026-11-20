package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Invoice series: real sales and Test Mode purchases are numbered apart, so a
// test document can never take a number in the sales series.
const (
	InvoiceSeriesSales = "RYZE"
	InvoiceSeriesTest  = "TEST"

	InvoiceStatusIssued = "issued"

	// InvoiceTaxExemptionSmallBusiness is the VAT exemption applied to every
	// sale (owner decision, 2026-11-20): the Portuguese small-business scheme,
	// article 53 of the VAT Code (SAF-T exemption code M10).
	InvoiceTaxExemptionSmallBusiness = "M10"
)

// Invoice corresponds to the invoices table: the purchase document issued for
// one completed purchase (migration 000048). Until certified invoicing software
// is connected it is a purchase receipt with no tax value. It is a snapshot of
// the sale at issue time and is immutable: the database rejects any update or
// delete.
type Invoice struct {
	ID                  string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Number              string    `gorm:"column:number;type:varchar(32);not null" json:"number"`
	Series              string    `gorm:"column:series;type:varchar(8);not null" json:"series"`
	Year                int       `gorm:"column:year;type:smallint;not null" json:"year"`
	Sequence            int       `gorm:"column:sequence;not null" json:"sequence"`
	PurchaseID          string    `gorm:"column:purchase_id;type:varchar(36);not null" json:"purchase_id"`
	UserID              string    `gorm:"column:user_id;type:varchar(36);not null" json:"-"`
	ProgramID           string    `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	Test                bool      `gorm:"column:test;not null" json:"test"`
	Status              string    `gorm:"column:status;type:varchar(16);not null" json:"status"`
	IssuedAt            time.Time `gorm:"column:issued_at;not null" json:"issued_at"`
	PurchasedAt         time.Time `gorm:"column:purchased_at;not null" json:"purchased_at"`
	SellerName          string    `gorm:"column:seller_name;type:varchar(160);not null" json:"seller_name"`
	SellerTaxID         *string   `gorm:"column:seller_tax_id;type:varchar(32)" json:"seller_tax_id"`
	SellerAddress       *string   `gorm:"column:seller_address;type:varchar(255)" json:"seller_address"`
	BuyerName           string    `gorm:"column:buyer_name;type:varchar(255);not null" json:"buyer_name"`
	BuyerTaxID          *string   `gorm:"column:buyer_tax_id;type:varchar(9)" json:"buyer_tax_id"`
	Description         string    `gorm:"column:description;type:varchar(255);not null" json:"description"`
	ProductType         string    `gorm:"column:product_type;type:varchar(32);not null" json:"product_type"`
	Quantity            int       `gorm:"column:quantity;not null" json:"quantity"`
	UnitPriceMinorUnits int64     `gorm:"column:unit_price_minor_units;not null" json:"unit_price_minor_units"`
	DiscountMinorUnits  int64     `gorm:"column:discount_minor_units;not null" json:"discount_minor_units"`
	TaxRateBPS          int       `gorm:"column:tax_rate_bps;not null" json:"tax_rate_bps"`
	TaxMinorUnits       int64     `gorm:"column:tax_minor_units;not null" json:"tax_minor_units"`
	TaxExemptionCode    *string   `gorm:"column:tax_exemption_code;type:varchar(8)" json:"tax_exemption_code"`
	TotalMinorUnits     int64     `gorm:"column:total_minor_units;not null" json:"total_minor_units"`
	Currency            string    `gorm:"column:currency;type:varchar(3);not null" json:"currency"`
	PaymentMethod       *string   `gorm:"column:payment_method;type:varchar(20)" json:"payment_method"`
	CreatedAt           time.Time `gorm:"column:created_at" json:"-"`
}

func (i *Invoice) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}

// InvoicePurchase is what an invoice is issued from: a completed purchase with
// its buyer and programme, loaded even when the programme was later removed.
type InvoicePurchase struct {
	Purchase Purchase
	Buyer    User
	Program  Program
}
