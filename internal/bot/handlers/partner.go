package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// PartnerHandler handles partner detail views.
type PartnerHandler struct {
	partnerRepo *storage.PartnerRepo
}

// NewPartnerHandler creates a new PartnerHandler.
func NewPartnerHandler(repo *storage.PartnerRepo) *PartnerHandler {
	return &PartnerHandler{partnerRepo: repo}
}

// HandleCallback processes callback queries like "partner:123".
func (h *PartnerHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	// Parse partner ID from callback data
	data := update.CallbackQuery.Data
	if !strings.HasPrefix(data, "partner:") {
		return
	}

	idStr := strings.TrimPrefix(data, "partner:")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}

	partner, err := h.partnerRepo.GetByID(ctx, id)
	if err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Партнёр не найден",
			ShowAlert:       true,
		})
		return
	}

	// Answer callback to remove loading indicator
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// Send partner card
	card := formatPartnerCard(partner)

	var chatID int64
	if update.CallbackQuery.Message.Message != nil {
		chatID = update.CallbackQuery.Message.Message.Chat.ID
	}
	if chatID == 0 {
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      card,
		ParseMode: models.ParseModeHTML,
	})
}

// formatPartnerCard creates a rich HTML-formatted partner card.
func formatPartnerCard(p *domain.Partner) string {
	certIcon := func(cert string) string {
		if cert == "" || cert == "-" || cert == "N/A" {
			return "❌"
		}
		return "✅"
	}

	certLine := func(name, cert string) string {
		icon := certIcon(cert)
		if cert == "" || cert == "-" || cert == "N/A" {
			cert = "—"
		}
		return fmt.Sprintf("  %s %s: %s", icon, name, cert)
	}

	gapToNext := calculateGap(p)

	card := fmt.Sprintf(`🏢 <b>%s</b>
━━━━━━━━━━━━━━━━━
🆔 %s
🏅 Tier: <b>%s</b>
📍 %s, %s

📊 <b>Certifications:</b>
%s
%s
%s
%s

💰 <b>Revenue YTD:</b> $%s / $%s
%s`,
		p.Name,
		p.PartnerID,
		tierDisplay(p.Tier),
		p.City, p.Country,
		certLine("Compute", p.ComputeCert),
		certLine("Networking", p.NetworkingCert),
		certLine("Hybrid Cloud", p.HybridCloud),
		certLine("Storage", p.StorageCert),
		formatNumber(p.RevenueYTD),
		formatNumber(p.Target),
		gapToNext,
	)

	// Add contact info if available
	if p.ContactName != "" {
		card += fmt.Sprintf("\n\n👤 <b>Contact:</b> %s", p.ContactName)
		if p.ContactEmail != "" {
			card += fmt.Sprintf("\n📧 %s", p.ContactEmail)
		}
		if p.ContactPhone != "" {
			card += fmt.Sprintf("\n📱 %s", p.ContactPhone)
		}
	}

	return card
}

func tierDisplay(tier string) string {
	switch strings.ToLower(tier) {
	case "platinum":
		return "💎 Platinum"
	case "gold":
		return "🥇 Gold"
	case "silver":
		return "🥈 Silver"
	case "business":
		return "🏷️ Business"
	default:
		return tier
	}
}

func calculateGap(p *domain.Partner) string {
	if p.Target <= 0 {
		return ""
	}
	gap := p.Target - p.RevenueYTD
	if gap <= 0 {
		return "📈 🎉 Target reached!"
	}
	pct := (p.RevenueYTD / p.Target) * 100
	return fmt.Sprintf("📈 Gap: $%s (%.0f%% complete)", formatNumber(gap), pct)
}

func formatNumber(n float64) string {
	if n == 0 {
		return "0"
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.0fK", n/1_000)
	}
	return fmt.Sprintf("%.0f", n)
}
