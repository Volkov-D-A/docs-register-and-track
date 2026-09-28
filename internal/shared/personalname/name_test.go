package personalname

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	for _, tc := range []struct {
		name, last, first, pat string
		none                   bool
		want                   string
		invalid                bool
	}{
		{name: "trim and compound", last: "  де ла Крус ", first: " Анна-Мария ", pat: " Ивановна ", want: "де ла Крус Анна-Мария Ивановна"},
		{name: "ignore disabled patronymic", last: "О’Нил", first: "Sean", pat: strings.Repeat("x", 101), none: true, want: "О’Нил Sean"},
		{name: "missing patronymic", last: "Иванов", first: "Иван", pat: " \t", invalid: true},
		{name: "missing first name", last: "Иванов", first: " \n", none: true, invalid: true},
		{name: "missing last name", first: "Иван", none: true, invalid: true},
		{name: "unicode boundary", last: strings.Repeat("Я", 100), first: "Иван", none: true, want: strings.Repeat("Я", 100) + " Иван"},
		{name: "too long", last: strings.Repeat("Я", 101), first: "Иван", none: true, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			last, first, pat, err := Normalize(tc.last, tc.first, tc.pat, tc.none)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, Display(last, first, pat, tc.none))
			if tc.none {
				require.Empty(t, pat)
			}
		})
	}
}
