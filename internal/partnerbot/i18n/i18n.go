package i18n

// Lang represents a supported language.
type Lang string

const (
	LangRU Lang = "ru"
	LangEN Lang = "en"
)

// messages holds all localized strings keyed by message ID → language.
var messages = map[string]map[Lang]string{
	// Onboarding
	"welcome": {
		LangRU: "👋 Добро пожаловать в HPE Partner Portal!\n\nЭтот бот позволяет вам видеть статус готовности вашей компании в программе HPE Partner Ready Vantage.\n\nДля начала, введите вашу корпоративную email-адрес для верификации:",
		LangEN: "👋 Welcome to HPE Partner Portal!\n\nThis bot lets you view your company's readiness status in the HPE Partner Ready Vantage program.\n\nTo get started, please enter your corporate email for verification:",
	},
	"welcome_back": {
		LangRU: "👋 С возвращением! Используйте /status чтобы посмотреть карточку вашей компании.",
		LangEN: "👋 Welcome back! Use /status to view your company card.",
	},

	// Email verification
	"email_prompt": {
		LangRU: "📧 Введите вашу корпоративную email:",
		LangEN: "📧 Please enter your corporate email:",
	},
	"email_invalid": {
		LangRU: "❌ Некорректный email. Пожалуйста, введите корпоративную почту (например, ivan@company.kz):",
		LangEN: "❌ Invalid email. Please enter a corporate email (e.g., ivan@company.kz):",
	},
	"email_sent": {
		LangRU: "✅ Запрос на верификацию отправлен администратору.\nВы получите уведомление после проверки вашего email.\n\n⏳ Обычно это занимает не более 24 часов.",
		LangEN: "✅ Verification request sent to administrator.\nYou will be notified once your email is verified.\n\n⏳ This usually takes up to 24 hours.",
	},
	"email_company_not_found": {
		LangRU: "❌ Компания не найдена по вашему email.\nОбратитесь к вашему HPE Partner Business Manager для помощи.",
		LangEN: "❌ Company not found by your email.\nPlease contact your HPE Partner Business Manager for assistance.",
	},
	"email_multiple_companies": {
		LangRU: "🔍 Найдено несколько компаний. Выберите вашу:",
		LangEN: "🔍 Multiple companies found. Please select yours:",
	},

	// Pending / Rejected
	"pending_approval": {
		LangRU: "⏳ Ваш запрос на доступ ожидает одобрения администратора.",
		LangEN: "⏳ Your access request is pending administrator approval.",
	},
	"approved": {
		LangRU: "🎉 Ваш доступ одобрен! Используйте /status чтобы посмотреть карточку.",
		LangEN: "🎉 Your access has been approved! Use /status to view your card.",
	},
	"rejected": {
		LangRU: "❌ Ваш запрос на доступ был отклонён. Обратитесь к вашему HPE PBM.",
		LangEN: "❌ Your access request was rejected. Please contact your HPE PBM.",
	},

	// Card display
	"status_ready": {
		LangRU: "🟢 <b>FY27 Ready</b>",
		LangEN: "🟢 <b>FY27 Ready</b>",
	},
	"status_not_ready": {
		LangRU: "🔴 <b>FY27 Not Ready</b>",
		LangEN: "🔴 <b>FY27 Not Ready</b>",
	},
	"btn_upgrade": {
		LangRU: "📈 Как повысить статус?",
		LangEN: "📈 How to upgrade?",
	},
	"btn_retention": {
		LangRU: "📉 Условия удержания",
		LangEN: "📉 Retention requirements",
	},
	"no_partner": {
		LangRU: "❌ Ваша компания не привязана. Используйте /start для верификации.",
		LangEN: "❌ Your company is not linked. Use /start to verify.",
	},

	// Language
	"lang_switched_ru": {
		LangRU: "🇷🇺 Язык переключён на русский.",
		LangEN: "🇷🇺 Язык переключён на русский.",
	},
	"lang_switched_en": {
		LangRU: "🇬🇧 Language switched to English.",
		LangEN: "🇬🇧 Language switched to English.",
	},
	"lang_choose": {
		LangRU: "🌐 Выберите язык / Choose language:",
		LangEN: "🌐 Выберите язык / Choose language:",
	},

	// Help
	"help": {
		LangRU: "📋 <b>Доступные команды:</b>\n\n" +
			"/status — Карточка вашей компании\n" +
			"/lang — Сменить язык (RU/EN)\n" +
			"/help — Список команд",
		LangEN: "📋 <b>Available commands:</b>\n\n" +
			"/status — Your company card\n" +
			"/lang — Change language (RU/EN)\n" +
			"/help — Command list",
	},
}

// T returns a localized string for the given key and language.
// Falls back to Russian if the key or language is not found.
func T(key string, lang Lang) string {
	if m, ok := messages[key]; ok {
		if s, ok := m[lang]; ok {
			return s
		}
		// Fallback to Russian
		if s, ok := m[LangRU]; ok {
			return s
		}
	}
	return key // Last resort: return the key itself
}

// ParseLang converts a string to Lang, defaulting to Russian.
func ParseLang(s string) Lang {
	switch s {
	case "en":
		return LangEN
	default:
		return LangRU
	}
}
