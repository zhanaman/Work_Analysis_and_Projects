package rbac

import (
	"testing"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
)

func TestCommandsForUser_Admin(t *testing.T) {
	user := &domain.User{Role: domain.RoleAdmin}
	cmds := CommandsForUser(user)

	if len(cmds) != len(Commands) {
		t.Errorf("admin should see all %d commands, got %d", len(Commands), len(cmds))
	}
}

func TestCommandsForUser_PBM(t *testing.T) {
	user := &domain.User{Role: domain.RolePBM}
	cmds := CommandsForUser(user)

	// PBM has SearchPartners only → should see: search, help (AlwaysShow)
	for _, cmd := range cmds {
		if cmd.Command == "stats" || cmd.Command == "users" {
			t.Errorf("PBM should NOT see /%s", cmd.Command)
		}
	}

	// Must see search and help
	found := map[string]bool{}
	for _, cmd := range cmds {
		found[cmd.Command] = true
	}
	if !found["search"] {
		t.Error("PBM must see /search")
	}
	if !found["help"] {
		t.Error("PBM must see /help")
	}
}

func TestCommandsForUser_Nil(t *testing.T) {
	cmds := CommandsForUser(nil)

	// nil user → only AlwaysShow commands (search, help)
	for _, cmd := range cmds {
		if !cmd.AlwaysShow && cmd.Permission != "" {
			t.Errorf("nil user should NOT see /%s", cmd.Command)
		}
	}
}

func TestHelpTextForUser_ContainsSearchForPBM(t *testing.T) {
	user := &domain.User{Role: domain.RolePBM}
	text := HelpTextForUser(user)

	if len(text) == 0 {
		t.Error("help text must not be empty")
	}
	if !contains(text, "/search") {
		t.Error("PBM help must contain /search")
	}
	if contains(text, "/stats") {
		t.Error("PBM help must NOT contain /stats")
	}
	if contains(text, "/users") {
		t.Error("PBM help must NOT contain /users")
	}
}

func TestTelegramCommandsForUser(t *testing.T) {
	user := &domain.User{Role: domain.RolePBM}
	tgCmds := TelegramCommandsForUser(user)

	for _, cmd := range tgCmds {
		if cmd.Command == "" || cmd.Description == "" {
			t.Errorf("telegram command has empty fields: %+v", cmd)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
