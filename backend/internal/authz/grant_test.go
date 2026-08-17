package authz

import (
	"testing"
)

func TestCanIsTheOnlyQuestionHandlersAsk(t *testing.T) {
	g := NewGrant("contributor", []string{EventRead, MemberRead})
	if !g.Can(EventRead) || !g.Can(MemberRead) {
		t.Fatal("expected granted permissions")
	}
	if g.Can(UserEmailRead) {
		t.Fatal("missing row must be a deny, not a role-name check")
	}
	if g.Can("member.role.update") {
		t.Fatal("contributor must not change roles")
	}
}
