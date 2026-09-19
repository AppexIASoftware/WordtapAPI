package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductType define el tipo de cobro (one_time_course o monthly_subscription).
type ProductType string

const (
	ProductTypeOneTimeCourse       ProductType = "one_time_course"
	ProductTypeMonthlySubscription ProductType = "monthly_subscription"
)

// PaymentStatus define el estado de la transacción (pending, approved, failed, refunded, cancelled).
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusApproved  PaymentStatus = "approved"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// Product representa cursos o suscripciones a la venta con su precio en centavos.
type Product struct {
	ID                string      `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name              string      `gorm:"type:varchar(180);not null" json:"name"`
	Description       *string     `gorm:"type:text" json:"description"`
	ProductType       ProductType `gorm:"type:varchar(40);not null" json:"product_type"`
	Provider          string      `gorm:"type:varchar(40);default:'stripe';not null" json:"provider"`
	ProviderProductID *string     `gorm:"type:varchar(255)" json:"provider_product_id"`
	ProviderPriceID   *string     `gorm:"type:varchar(255)" json:"provider_price_id"`
	PriceCents        int         `gorm:"not null" json:"price_cents"`
	Currency          string      `gorm:"type:char(3);default:'USD';not null" json:"currency"`
	AccessTier        AccessTier  `gorm:"type:varchar(20);not null" json:"access_tier"`
	IsActive          bool        `gorm:"default:true;not null" json:"is_active"`
	CreatedAt         time.Time   `json:"created_at"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

// Order almacena las intenciones y confirmaciones de compra del usuario.
type Order struct {
	ID                 string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID             string        `gorm:"type:varchar(36);not null;index:idx_orders_user,priority:1" json:"user_id"`
	ProductID          string        `gorm:"type:varchar(36);not null;index" json:"product_id"`
	Provider           string        `gorm:"type:varchar(40);default:'stripe';not null" json:"provider"`
	ProviderCheckoutID *string       `gorm:"type:varchar(255);uniqueIndex" json:"provider_checkout_id"`
	ProviderPaymentID  *string       `gorm:"type:varchar(255)" json:"provider_payment_id"`
	AmountCents        int           `gorm:"not null" json:"amount_cents"`
	Currency           string        `gorm:"type:char(3);default:'USD';not null" json:"currency"`
	Status             PaymentStatus `gorm:"type:varchar(20);default:'pending';not null;index:idx_orders_status,priority:1" json:"status"`
	IdempotencyKey     *string       `gorm:"type:varchar(255);uniqueIndex" json:"idempotency_key"`
	PaidAt             *time.Time    `json:"paid_at"`
	CreatedAt          time.Time     `gorm:"index:idx_orders_user,priority:2;index:idx_orders_status,priority:2" json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`

	User    *User    `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT" json:"user,omitempty"`
	Product *Product `gorm:"foreignKey:ProductID;constraint:OnDelete:RESTRICT" json:"product,omitempty"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	return nil
}

// UserEntitlement registra los derechos vigentes de acceso premium del usuario.
type UserEntitlement struct {
	ID              string      `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID          string      `gorm:"type:varchar(36);not null;index:idx_entitlements_user,priority:1" json:"user_id"`
	ProductID       string      `gorm:"type:varchar(36);not null;index" json:"product_id"`
	OrderID         *string     `gorm:"type:varchar(36);index" json:"order_id"`
	EntitlementType ProductType `gorm:"type:varchar(40);not null" json:"entitlement_type"`
	StartsAt        time.Time   `gorm:"not null" json:"starts_at"`
	EndsAt          *time.Time  `gorm:"index:idx_entitlements_user,priority:3" json:"ends_at"`
	IsActive        bool        `gorm:"default:true;not null;index:idx_entitlements_user,priority:2" json:"is_active"`
	CreatedAt       time.Time   `json:"created_at"`

	User    *User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Product *Product `gorm:"foreignKey:ProductID;constraint:OnDelete:RESTRICT" json:"product,omitempty"`
	Order   *Order   `gorm:"foreignKey:OrderID;constraint:OnDelete:SET NULL" json:"order,omitempty"`
}

func (e *UserEntitlement) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}

// Subscription controla el ciclo de facturación y renovación mensual.
type Subscription struct {
	ID                     string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID                 string     `gorm:"type:varchar(36);not null;index" json:"user_id"`
	ProductID              string     `gorm:"type:varchar(36);not null;index" json:"product_id"`
	ProviderSubscriptionID string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"provider_subscription_id"`
	Status                 string     `gorm:"type:varchar(30);not null" json:"status"`
	CurrentPeriodStart     *time.Time `json:"current_period_start"`
	CurrentPeriodEnd       *time.Time `json:"current_period_end"`
	CancelAtPeriodEnd      bool       `gorm:"default:false;not null" json:"cancel_at_period_end"`
	CancelledAt            *time.Time `json:"cancelled_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`

	User    *User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Product *Product `gorm:"foreignKey:ProductID;constraint:OnDelete:RESTRICT" json:"product,omitempty"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

// PaymentEvent bitácora inmutable de webhooks entrantes para evitar reprocesos.
type PaymentEvent struct {
	ID              string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	Provider        string     `gorm:"type:varchar(40);not null" json:"provider"`
	ProviderEventID string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"provider_event_id"`
	EventType       string     `gorm:"type:varchar(100);not null" json:"event_type"`
	Payload         string     `gorm:"type:json;not null" json:"payload"`
	ProcessedAt     *time.Time `json:"processed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (pe *PaymentEvent) BeforeCreate(tx *gorm.DB) error {
	if pe.ID == "" {
		pe.ID = uuid.NewString()
	}
	return nil
}
