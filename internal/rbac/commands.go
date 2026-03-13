package rbac

import (
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/go-telegram/bot/models"
)

// BotCommand ties a slash command to its required permission.
// This is the single source of truth for commands, menu items, and help text.
type BotCommand struct {
	Command     string     // "search", "stats"
	Description string     // Telegram menu description
	HelpLine    string     // HTML-formatted line for /help
	Permission  Permission // Required permission ("" = no check)
	AlwaysShow  bool       // true = shown to all authorized users
}

// Commands is the single source of truth for all bot commands.
// To add a new feature:
//  1. Add Permission to rbac.go
//  2. Add to RolePermissions in rbac.go
//  3. Add BotCommand here
//  4. Write handler — menu + /help auto-update.
var Commands = []BotCommand{
	{
		Command:     "search",
		Description: "🔍 Поиск партнёра",
		HelpLine:    "🔍 /search <code>&lt;имя&gt;</code> — поиск партнёра",
		Permission:  SearchPartners,
	},
	{
		Command:     "status",
		Description: "🏢 Моя компания",
		HelpLine:    "🏢 /status — данные вашей компании",
		Permission:  ViewOwnCard,
	},
	{
		Command:     "help",
		Description: "❓ Справка",
		HelpLine:    "ℹ️ /help — справка",
		Permission:  "",
		AlwaysShow:  true,
	},
	{
		Command:     "stats",
		Description: "📊 Аналитика CCA",
		HelpLine:    "📊 /stats — аналитика CCA",
		Permission:  ViewStats,
	},
	{
		Command:     "users",
		Description: "👥 Управление пользователями",
		HelpLine:    "👥 /users — управление пользователями",
		Permission:  ManageUsers,
	},
}

// CommandsForUser returns commands the user has access to.
func CommandsForUser(user *domain.User) []BotCommand {
	var result []BotCommand
	for _, cmd := range Commands {
		if cmd.AlwaysShow || cmd.Permission == "" || Can(user, cmd.Permission) {
			result = append(result, cmd)
		}
	}
	return result
}

// TelegramCommandsForUser returns Telegram BotCommand slice for SetMyCommands.
func TelegramCommandsForUser(user *domain.User) []models.BotCommand {
	cmds := CommandsForUser(user)
	result := make([]models.BotCommand, 0, len(cmds))
	for _, cmd := range cmds {
		result = append(result, models.BotCommand{
			Command:     cmd.Command,
			Description: cmd.Description,
		})
	}
	return result
}

// HelpTextForUser returns HTML-formatted help lines for commands the user can access.
func HelpTextForUser(user *domain.User) string {
	cmds := CommandsForUser(user)
	text := "📋 <b>Команды:</b>\n"
	for _, cmd := range cmds {
		text += cmd.HelpLine + "\n"
	}
	return text
}
