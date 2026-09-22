package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContentBankType define el catálogo dinámico y extensible de tipos de bancos pedagógicos
// (ej: vocabulary, phrase, collocation, minimal_pair, false_friend, dialogue, idioms, slang, business_english).
type ContentBankType struct {
	Slug        string    `gorm:"type:varchar(40);primaryKey" json:"slug"`
	DisplayName string    `gorm:"type:varchar(80);not null" json:"display_name"`
	ColorTheme  string    `gorm:"type:varchar(20);default:'emerald';not null" json:"color_theme"`
	Description *string   `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// ContentBankItem representa la Unidad Universal de Aprendizaje (Universal Learning Unit).
// Es 100% polimórfica y desacoplada de idiomas (source_lang -> target_lang),
// permitiendo escalar a múltiples idiomas (EN, FR, DE, PT, IT) sin migraciones de esquema.
type ContentBankItem struct {
	ID          string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	BankType    string        `gorm:"type:varchar(40);not null;index:idx_vault_bank_lang,priority:1" json:"bank_type"`
	CategoryID  *string       `gorm:"type:varchar(36);index" json:"category_id"`
	SourceLang  string        `gorm:"type:varchar(10);default:'es';not null;index:idx_vault_bank_lang,priority:2" json:"source_lang"` // L1: Idioma del alumno (ej: 'es', 'pt')
	TargetLang  string        `gorm:"type:varchar(10);default:'en';not null;index:idx_vault_bank_lang,priority:3" json:"target_lang"` // L2: Idioma a aprender (ej: 'en', 'fr', 'de')
	PromptText  string        `gorm:"type:text;not null" json:"prompt_text"`                                                          // Estímulo (español, pregunta, fonema o contexto)
	TargetText  string        `gorm:"type:text;not null" json:"target_text"`                                                          // Solución en idioma destino
	IPA         *string       `gorm:"type:varchar(180)" json:"ipa"`
	CEFRLevel   string        `gorm:"type:varchar(5);default:'A1';not null;index:idx_vault_difficulty,priority:1" json:"cefr_level"` // A1, A2, B1, B2, C1, C2
	Difficulty  int           `gorm:"default:1;not null;index:idx_vault_difficulty,priority:2" json:"difficulty"`                    // 1: Básico, 2: Medio, 3: Desafío
	AudioURL    *string       `gorm:"type:text" json:"audio_url"`
	Explanation *string       `gorm:"type:text" json:"explanation"`
	Metadata    string        `gorm:"type:json;not null" json:"metadata"` // Metadatos específicos dinámicos según el tipo de banco
	Status      ContentStatus `gorm:"type:varchar(20);default:'published';not null" json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	BankTypeRef *ContentBankType `gorm:"foreignKey:BankType;references:Slug;constraint:OnDelete:CASCADE" json:"bank_type_ref,omitempty"`
	Category    *ContentCategory `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`
}

func (cbi *ContentBankItem) BeforeCreate(tx *gorm.DB) error {
	if cbi.ID == "" {
		cbi.ID = uuid.NewString()
	}
	if cbi.Metadata == "" {
		cbi.Metadata = "{}"
	}
	if cbi.SourceLang == "" {
		cbi.SourceLang = "es"
	}
	if cbi.TargetLang == "" {
		cbi.TargetLang = "en"
	}
	return nil
}
