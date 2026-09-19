package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContentStatus define los estados de publicación (draft, published, archived).
type ContentStatus string

const (
	ContentStatusDraft     ContentStatus = "draft"
	ContentStatusPublished ContentStatus = "published"
	ContentStatusArchived  ContentStatus = "archived"
)

// LessonContentType define el tipo de paso dentro de una lección (word, phrase, grammar, tip, exercise).
type LessonContentType string

const (
	LessonContentTypeWord     LessonContentType = "word"
	LessonContentTypePhrase   LessonContentType = "phrase"
	LessonContentTypeGrammar  LessonContentType = "grammar"
	LessonContentTypeTip      LessonContentType = "tip"
	LessonContentTypeExercise LessonContentType = "exercise"
)

// ContentCategory agrupa lecciones y vocabulario por temáticas (ej: viajes, trabajo).
type ContentCategory struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Slug        string    `gorm:"type:varchar(120);uniqueIndex;not null" json:"slug"`
	Description *string   `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *ContentCategory) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

// Course representa un plan formativo completo (gratuito o premium).
type Course struct {
	ID            string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	Title         string        `gorm:"type:varchar(180);not null" json:"title"`
	Slug          string        `gorm:"type:varchar(200);uniqueIndex;not null" json:"slug"`
	Description   *string       `gorm:"type:text" json:"description"`
	AccessTier    AccessTier    `gorm:"type:varchar(20);default:'free';not null" json:"access_tier"`
	Level         string        `gorm:"type:varchar(30);default:'beginner';not null" json:"level"`
	Status        ContentStatus `gorm:"type:varchar(20);default:'draft';not null" json:"status"`
	SortOrder     int           `gorm:"default:0;not null" json:"sort_order"`
	CoverImageURL *string       `gorm:"type:text" json:"cover_image_url"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`

	Lessons []Lesson `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"lessons,omitempty"`
}

func (c *Course) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

// Lesson representa una lección de aprendizaje individual dentro de un curso.
type Lesson struct {
	ID               string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	CourseID         string        `gorm:"type:varchar(36);not null;index:idx_lessons_course_order,priority:1" json:"course_id"`
	CategoryID       *string       `gorm:"type:varchar(36);index" json:"category_id"`
	Title            string        `gorm:"type:varchar(180);not null" json:"title"`
	Slug             string        `gorm:"type:varchar(200);uniqueIndex;not null" json:"slug"`
	Description      *string       `gorm:"type:text" json:"description"`
	AccessTier       AccessTier    `gorm:"type:varchar(20);default:'free';not null;index:idx_lessons_status_tier,priority:2" json:"access_tier"`
	Status           ContentStatus `gorm:"type:varchar(20);default:'draft';not null;index:idx_lessons_status_tier,priority:1" json:"status"`
	SortOrder        int           `gorm:"default:0;not null;index:idx_lessons_course_order,priority:2" json:"sort_order"`
	EstimatedMinutes *int          `json:"estimated_minutes"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`

	Course   *Course          `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	Category *ContentCategory `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`
	Items    []LessonItem     `gorm:"foreignKey:LessonID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

func (l *Lesson) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	return nil
}

// VocabularyItem almacena palabras en inglés con su traducción, pronunciación y ejemplo.
type VocabularyItem struct {
	ID              string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	CategoryID      *string       `gorm:"type:varchar(36);index" json:"category_id"`
	Spanish         string        `gorm:"type:varchar(180);not null;index" json:"spanish"`
	English         string        `gorm:"type:varchar(180);not null;index" json:"english"`
	Pronunciation   *string       `gorm:"type:varchar(180)" json:"pronunciation"`
	PartOfSpeech    *string       `gorm:"type:varchar(50)" json:"part_of_speech"`
	Definition      *string       `gorm:"type:text" json:"definition"`
	ExampleSentence *string       `gorm:"type:text" json:"example_sentence"`
	AudioURL        *string       `gorm:"type:text" json:"audio_url"`
	Status          ContentStatus `gorm:"type:varchar(20);default:'published';not null" json:"status"`
	CreatedAt       time.Time     `json:"created_at"`

	Category *ContentCategory `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`
}

func (v *VocabularyItem) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	return nil
}

// Phrase representa expresiones y frases conversacionales completas.
type Phrase struct {
	ID          string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	CategoryID  *string       `gorm:"type:varchar(36);index" json:"category_id"`
	Spanish     string        `gorm:"type:text;not null" json:"spanish"`
	English     string        `gorm:"type:text;not null" json:"english"`
	AudioURL    *string       `gorm:"type:text" json:"audio_url"`
	Explanation *string       `gorm:"type:text" json:"explanation"`
	Status      ContentStatus `gorm:"type:varchar(20);default:'published';not null" json:"status"`
	CreatedAt   time.Time     `json:"created_at"`

	Category *ContentCategory `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`
}

func (p *Phrase) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

// LessonItem estructura la secuencia de pasos o tarjetas dentro de una lección.
type LessonItem struct {
	ID               string            `gorm:"type:varchar(36);primaryKey" json:"id"`
	LessonID         string            `gorm:"type:varchar(36);not null;index:idx_items_lesson_order,priority:1" json:"lesson_id"`
	VocabularyItemID *string           `gorm:"type:varchar(36);index" json:"vocabulary_item_id"`
	PhraseID         *string           `gorm:"type:varchar(36);index" json:"phrase_id"`
	ItemType         LessonContentType `gorm:"type:varchar(20);not null" json:"item_type"`
	ContentText      *string           `gorm:"type:text" json:"content_text"`
	SortOrder        int               `gorm:"default:0;not null;index:idx_items_lesson_order,priority:2" json:"sort_order"`
	IsRequired       bool              `gorm:"default:true;not null" json:"is_required"`

	Lesson         *Lesson         `gorm:"foreignKey:LessonID;constraint:OnDelete:CASCADE" json:"lesson,omitempty"`
	VocabularyItem *VocabularyItem `gorm:"foreignKey:VocabularyItemID;constraint:OnDelete:SET NULL" json:"vocabulary_item,omitempty"`
	Phrase         *Phrase         `gorm:"foreignKey:PhraseID;constraint:OnDelete:SET NULL" json:"phrase,omitempty"`
}

func (li *LessonItem) BeforeCreate(tx *gorm.DB) error {
	if li.ID == "" {
		li.ID = uuid.NewString()
	}
	return nil
}

// ContentAsset almacena recursos multimedia (audios, imágenes, documentos) de lecciones.
type ContentAsset struct {
	ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	LessonID     *string   `gorm:"type:varchar(36);index" json:"lesson_id"`
	LessonItemID *string   `gorm:"type:varchar(36);index" json:"lesson_item_id"`
	AssetType    string    `gorm:"type:varchar(30);not null" json:"asset_type"`
	URL          string    `gorm:"type:text;not null" json:"url"`
	Title        *string   `gorm:"type:varchar(180)" json:"title"`
	CreatedAt    time.Time `json:"created_at"`

	Lesson     *Lesson     `gorm:"foreignKey:LessonID;constraint:OnDelete:CASCADE" json:"lesson,omitempty"`
	LessonItem *LessonItem `gorm:"foreignKey:LessonItemID;constraint:OnDelete:CASCADE" json:"lesson_item,omitempty"`
}

func (ca *ContentAsset) BeforeCreate(tx *gorm.DB) error {
	if ca.ID == "" {
		ca.ID = uuid.NewString()
	}
	return nil
}
