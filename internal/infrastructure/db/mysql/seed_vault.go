package mysql

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

/**
 * seedContentBankItems siembra de forma idempotente las Unidades Universales de Aprendizaje
 * del Banco Oficial Wordtap (created_by = nil, status = published).
 */
func seedContentBankItems(db *gorm.DB) error {
	now := time.Now()

	type vaultSeedItem struct {
		bankType    string
		promptText  string
		targetText  string
		ipa         string
		cefr        string
		difficulty  int
		explanation string
		metadata    string
	}

	officialItems := []vaultSeedItem{
		// 1. Vocabulario Esencial (vocabulary)
		{
			bankType:    "vocabulary",
			promptText:  "ser / estar",
			targetText:  "be",
			ipa:         "/biː/",
			cefr:        "A1",
			difficulty:  1,
			explanation: "El verbo más fundamental en inglés; copulativo e irregular.",
			metadata:    `{"pos":"Verbo irregular","example":"\"I am happy today\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "tener",
			targetText:  "have",
			ipa:         "/hæv/",
			cefr:        "A1",
			difficulty:  1,
			explanation: "Indica posesión y funciona como auxiliar de tiempos perfectos.",
			metadata:    `{"pos":"Verbo irregular","example":"\"I have a car\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "hacer",
			targetText:  "do",
			ipa:         "/duː/",
			cefr:        "A1",
			difficulty:  1,
			explanation: "Verbo de acción general y auxiliar indispensable en preguntas y negaciones.",
			metadata:    `{"pos":"Verbo auxiliar","example":"\"Do your homework\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "decir",
			targetText:  "say",
			ipa:         "/seɪ/",
			cefr:        "A1",
			difficulty:  1,
			explanation: "Expresar palabras con la voz.",
			metadata:    `{"pos":"Verbo irregular","example":"\"Say hello to everyone\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "saber / conocer",
			targetText:  "know",
			ipa:         "/noʊ/",
			cefr:        "A1",
			difficulty:  1,
			explanation: "Tener certeza, información o familiaridad.",
			metadata:    `{"pos":"Verbo irregular","example":"\"I know the answer\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "abrumado / saturado",
			targetText:  "overwhelmed",
			ipa:         "/ˌoʊ.vɚˈwelmd/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Sensación de exceso de estímulos emocionales o carga de trabajo.",
			metadata:    `{"pos":"Adjetivo","example":"\"I feel overwhelmed with work today\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "ansioso / inquieto",
			targetText:  "anxious",
			ipa:         "/ˈæŋk.ʃəs/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Preocupado o intranquilo ante una situación incierta.",
			metadata:    `{"pos":"Adjetivo","example":"\"He felt anxious before the test\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "tarjeta de embarque",
			targetText:  "boarding pass",
			ipa:         "/ˈbɔːr.dɪŋ ˌpæs/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Documento oficial necesario para abordar un vuelo.",
			metadata:    `{"pos":"Sustantivo compuesto","example":"\"Show your boarding pass at gate 4\""}`,
		},
		{
			bankType:    "vocabulary",
			promptText:  "equipaje de mano",
			targetText:  "carry-on baggage",
			ipa:         "/ˈkær.i.ɑːn ˈbæɡ.ɪdʒ/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Equipaje permitido dentro de la cabina del avión.",
			metadata:    `{"pos":"Sustantivo","example":"\"One carry-on per passenger\""}`,
		},

		// 2. Oraciones y Estructuras (phrase)
		{
			bankType:    "phrase",
			promptText:  "Me siento abrumado con el trabajo hoy.",
			targetText:  "I feel overwhelmed with work today.",
			ipa:         "/aɪ fiːl ˌoʊvərˈwɛlmd wɪð wɜːrk təˈdeɪ/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Expresión cotidiana para comunicar sobrecarga laboral.",
			metadata:    `{"context":"Ambiente laboral","structure":"Sujeto + Feel + Adjetivo"}`,
		},
		{
			bankType:    "phrase",
			promptText:  "¿Dónde está la puerta de embarque?",
			targetText:  "Where is the departure gate?",
			ipa:         "/wɛr ɪz ðə dɪˈpɑːrtʃər ɡeɪt/",
			cefr:        "A2",
			difficulty:  1,
			explanation: "Pregunta esencial al transitar por un aeropuerto.",
			metadata:    `{"context":"Viajes y aeropuertos","structure":"Pregunta WH"}`,
		},
		{
			bankType:    "phrase",
			promptText:  "Ella estuvo sinceramente agradecida por tu ayuda.",
			targetText:  "She was genuinely grateful for your help.",
			ipa:         "/ʃiː wʌz ˈdʒɛnjuɪnli ˈɡreɪtfʊl fɔːr jʊər hɛlp/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Expresión formal de agradecimiento interpersonal.",
			metadata:    `{"context":"Relaciones interpersonales","structure":"Pasado simple + adverbio"}`,
		},
		{
			bankType:    "phrase",
			promptText:  "Trabajo como arquitecto en Londres.",
			targetText:  "I work as an architect in London.",
			ipa:         "/aɪ wɜːrk æz ən ˈɑːrkɪtɛkt ɪn ˈlʌndən/",
			cefr:        "A2",
			difficulty:  1,
			explanation: "Presentación laboral y profesión en contexto formal.",
			metadata:    `{"context":"Presentaciones personales","structure":"Presente simple profesional"}`,
		},

		// 3. Colocaciones Naturales (collocation)
		{
			bankType:    "collocation",
			promptText:  "tomar una decisión",
			targetText:  "make a decision",
			ipa:         "/meɪk ə dɪˈsɪʒ.ən/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Colocación fija: en inglés se usa el verbo 'make', no 'take'.",
			metadata:    `{"trap":"No usar 'take a decision' como calco del español"}`,
		},
		{
			bankType:    "collocation",
			promptText:  "prestar atención",
			targetText:  "pay attention",
			ipa:         "/peɪ əˈten.ʃən/",
			cefr:        "A1",
			difficulty:  1,
			explanation: "Colocación estándar con el verbo 'pay'.",
			metadata:    `{"explanation":"Colocación fija para concentrarse o escuchar"}`,
		},
		{
			bankType:    "collocation",
			promptText:  "cometer un error",
			targetText:  "make a mistake",
			ipa:         "/meɪk ə mɪˈsteɪk/",
			cefr:        "A1",
			difficulty:  1,
			explanation: "En inglés se utiliza 'make' para errores, nunca 'do'.",
			metadata:    `{"trap":"Se usa 'make', nunca 'do a mistake'"}`,
		},
		{
			bankType:    "collocation",
			promptText:  "hacer un esfuerzo",
			targetText:  "make an effort",
			ipa:         "/meɪk ən ˈef.ɚt/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Combinación natural para expresar empeño.",
			metadata:    `{"explanation":"Expresión fija para esforzarse"}`,
		},

		// 4. Falsos Amigos (false_friend)
		{
			bankType:    "false_friend",
			promptText:  "en realidad / de hecho (NO actualmente)",
			targetText:  "actually",
			ipa:         "/ˈæk.tʃu.ə.li/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Trampa común: 'Actually' significa de hecho. Para actualmente usar 'currently'.",
			metadata:    `{"trap":"'Actually' significa de hecho o en realidad. Actualmente es 'currently'."}`,
		},
		{
			bankType:    "false_friend",
			promptText:  "asistir / ir (NO atender al cliente)",
			targetText:  "attend",
			ipa:         "/əˈtend/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Trampa común: 'Attend' significa asistir a un evento. Atender es 'serve' o 'assist'.",
			metadata:    `{"trap":"'Attend' significa asistir a un evento. Atender es 'serve' o 'assist'."}`,
		},
		{
			bankType:    "false_friend",
			promptText:  "éxito (NO suceso)",
			targetText:  "success",
			ipa:         "/səkˈses/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Trampa común: 'Success' es logro o victoria. Suceso o evento es 'event' o 'incident'.",
			metadata:    `{"trap":"'Success' significa éxito. Suceso se traduce como 'event' o 'incident'."}`,
		},
		{
			bankType:    "false_friend",
			promptText:  "avergonzado / apenado (NO embarazada)",
			targetText:  "embarrassed",
			ipa:         "/ɪmˈbær.əst/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Trampa crítica: 'Embarrassed' significa apenado. Embarazada es 'pregnant'.",
			metadata:    `{"trap":"'Embarrassed' significa apenado. Embarazada es 'pregnant'."}`,
		},

		// 5. Pares Mínimos Fonéticos (minimal_pair)
		{
			bankType:    "minimal_pair",
			promptText:  "barco vs oveja",
			targetText:  "ship / sheep",
			ipa:         "/ʃɪp/ vs /ʃiːp/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Contraste fonético entre la vocal corta /ɪ/ y la vocal larga tensa /iː/.",
			metadata:    `{"contrast":"Vocal corta /ɪ/ frente a vocal larga /iː/"}`,
		},
		{
			bankType:    "minimal_pair",
			promptText:  "vivir vs marcharse",
			targetText:  "live / leave",
			ipa:         "/lɪv/ vs /liːv/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Diferenciación crucial para hispanohablantes al hablar en presente.",
			metadata:    `{"contrast":"Contraste fundamental en la vida diaria"}`,
		},

		// 6. Modismos e Idioms (idioms)
		{
			bankType:    "idioms",
			promptText:  "pan comido / muy fácil",
			targetText:  "piece of cake",
			ipa:         "/piːs əv keɪk/",
			cefr:        "A2",
			difficulty:  2,
			explanation: "Modismo popular que denota una tarea sencilla.",
			metadata:    `{"meaning":"Algo extremadamente sencillo de realizar"}`,
		},
		{
			bankType:    "idioms",
			promptText:  "raramente / casi nunca",
			targetText:  "once in a blue moon",
			ipa:         "/wʌns ɪn ə bluː muːn/",
			cefr:        "B1",
			difficulty:  2,
			explanation: "Expresión idiomática para eventos muy poco frecuentes.",
			metadata:    `{"meaning":"Un evento muy poco frecuente"}`,
		},
	}

	for _, item := range officialItems {
		var existing entities.ContentBankItem
		err := db.Where("prompt_text = ? AND target_text = ? AND created_by IS NULL", item.promptText, item.targetText).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newItem := entities.ContentBankItem{
				ID:          uuid.NewString(),
				BankType:    item.bankType,
				SourceLang:  "es",
				TargetLang:  "en",
				PromptText:  item.promptText,
				TargetText:  item.targetText,
				IPA:         &item.ipa,
				CEFRLevel:   item.cefr,
				Difficulty:  item.difficulty,
				Explanation: &item.explanation,
				Metadata:    item.metadata,
				Status:      entities.ContentStatusPublished,
				CreatedBy:   nil, // Banco Oficial Wordtap
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := db.Create(&newItem).Error; err != nil {
				return fmt.Errorf("failed to seed vault item %s: %w", item.targetText, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed checking vault item %s: %w", item.targetText, err)
		}
	}

	return nil
}
