package infrastructure

import "testing"

func TestEscapeLikeKeepsSearchWildcardsLiteral(t *testing.T) {
	if got, want := escapeLike("收益_20%!"), "收益!_20!%!!"; got != want {
		t.Fatalf("escapeLike() = %q, want %q", got, want)
	}
	filter, arguments := stockPoolFilter("收益_20%!")
	if filter != " WHERE name LIKE ? ESCAPE '!'" || len(arguments) != 1 || arguments[0] != "%收益!_20!%!!%" {
		t.Fatalf("stockPoolFilter() = %q/%#v", filter, arguments)
	}
}
