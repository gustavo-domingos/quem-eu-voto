package transparencia

import (
	"time"
)

func idade(nascimento string, hoje time.Time) (int, bool) {
	t, err := time.Parse("2006-01-02", nascimento)
	if err != nil {
		return 0, false
	}
	anos := hoje.Year() - t.Year()
	if hoje.YearDay() < t.YearDay() {
		anos--
	}
	return anos, anos > 0 && anos < 120
}

func idadeEm(nascimento string) *int {
	if a, ok := idade(nascimento, time.Now()); ok {
		return &a
	}
	return nil
}

func generoExtenso(g string) string {
	switch g {
	case "F":
		return "Feminino"
	case "M":
		return "Masculino"
	}
	return ""
}
